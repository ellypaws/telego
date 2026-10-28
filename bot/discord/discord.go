package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"telegram-discord/lib"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
	"gopkg.in/telebot.v4"
)

// maxHeartbeatAge is well past Discord's ~41s heartbeat interval, so an older ACK means the
// gateway stopped answering.
const maxHeartbeatAge = 2 * time.Minute

type Bot struct {
	Session *discordgo.Session
	Channel string

	logger  *log.Logger
	tracked map[string]Tracked
	mutex   sync.Mutex
}

type Tracked struct {
	Discord  *discordgo.Message `json:"discord,omitempty"`
	Telegram *telebot.Message   `json:"telegram,omitempty"`

	Expiry time.Time `json:"expiry"`
}

func New(token string, discordChannelID string, output io.Writer) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	logger := log.NewWithOptions(output,
		log.Options{
			Level:           log.InfoLevel,
			ReportTimestamp: true,
			Prefix:          "[Discord]",
		},
	)
	logger.SetColorProfile(termenv.TrueColor)

	var channel string
	if discordChannelID == "" {
		logger.Printf("No channel ID provided, will not be able to send messages")
	} else {
		channel = discordChannelID
	}

	dg.State.MaxMessageCount = 64
	dg.Dialer = gatewayDialer()
	return &Bot{
		Session: dg,
		Channel: channel,

		logger:  logger,
		tracked: make(map[string]Tracked),
	}, nil
}

func (b *Bot) Logger() *log.Logger {
	return b.logger
}

// Health reports whether the gateway is connected and answering heartbeats. It never blocks:
// discordgo holds the session lock for all of Session.Open, so a stuck handshake reports as a
// held lock instead of freezing the caller.
func (b *Bot) Health() error {
	if !b.Session.TryRLock() {
		return errors.New("gateway session lock is held")
	}
	ready, lastAck := b.Session.DataReady, b.Session.LastHeartbeatAck
	b.Session.RUnlock()

	if !ready {
		return errors.New("gateway is not connected")
	}
	if age := time.Since(lastAck); age > maxHeartbeatAge {
		return fmt.Errorf("no heartbeat ACK for %s", age.Round(time.Second))
	}
	return nil
}

func (b *Bot) Start() error {
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		b.logger.Info(
			"Discord bot logged in",
			"user", fmt.Sprintf("%s#%s", s.State.User.Username, s.State.User.Discriminator),
		)
		b.logger.Debug(
			"Discord connection established",
			"channel", lib.ChannelNameID(b.Session, b.Channel),
		)
	})
	b.Session.AddHandler(func(*discordgo.Session, *discordgo.Disconnect) {
		b.logger.Warn("Discord gateway disconnected")
	})
	b.Session.AddHandler(func(*discordgo.Session, *discordgo.Resumed) {
		b.logger.Info("Discord gateway session resumed")
	})

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	err = b.load(err)
	if err != nil {
		b.logger.Warn("Error loading tracked messages", "error", err)
	}

	return nil
}

func (b *Bot) Stop() error {
	err := b.save()
	if err != nil {
		b.logger.Warn("Error saving tracked messages", "error", err)
	}

	b.logger.Info("Closing Discord connection")
	return b.Session.Close()
}

func (b *Bot) load(err error) error {
	b.logger.Debug("Loading tracked messages")
	f, err := os.Open("tracked.json")
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error opening tracked messages file: %w", err)
		}
		b.logger.Warn("No tracked messages found")
	} else {
		defer f.Close()
		dec := json.NewDecoder(f)
		if err := dec.Decode(&b.tracked); err != nil {
			return fmt.Errorf("error decoding tracked messages: %w", err)
		}
		b.Clean()
		b.logger.Info(
			"Tracked messages loaded",
			"count", len(b.tracked),
		)
	}
	return nil
}

func (b *Bot) save() error {
	b.logger.Info(
		"Saving tracked messages",
		"count", len(b.tracked),
	)
	f, err := os.Create("tracked.json")
	if err != nil {
		return fmt.Errorf("error creating tracked messages file: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	b.Clean()
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, v := range b.tracked {
		v.Telegram.Poll = nil // do not store polls because telebot.Message.Poll.Type does not marshal properly
	}
	if err := enc.Encode(b.tracked); err != nil {
		return fmt.Errorf("error encoding tracked messages: %w", err)
	}
	b.logger.Info(
		"Tracked messages saved",
		"count", len(b.tracked),
	)
	return nil
}
