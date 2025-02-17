package discord

import (
	"fmt"
	"io"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	Session *discordgo.Session
	Channel string

	logger  *log.Logger
	tracked map[string]Tracked
	mutex   sync.Mutex
}

type Tracked struct {
	Discord  *discordgo.Message
	Telegram *telebot.Message
}

func New(token string, discordChannelID string, output io.Writer) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	logger := log.NewWithOptions(output,
		log.Options{
			Level:           log.DebugLevel,
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

func (b *Bot) Start() error {
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		b.logger.Info(
			"Discord bot logged in",
			"user", fmt.Sprintf("%s#%s", s.State.User.Username, s.State.User.Discriminator),
		)
	})

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	b.logger.Debug(
		"Discord connection established",
		"channel_id", b.Channel,
	)
	return nil
}

func (b *Bot) Stop() error {
	b.logger.Info("Closing Discord connection")
	return b.Session.Close()
}
