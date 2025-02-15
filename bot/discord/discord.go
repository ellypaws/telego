package discord

import (
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type Bot struct {
	Session *discordgo.Session
	Channel *string

	logger *log.Logger
}

func New(token string, discordChannelID string, output io.Writer) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	logger := log.New(output)

	var channel *string
	if discordChannelID == "" {
		logger.Printf("No channel ID provided, will not be able to send messages")
	} else {
		channel = &discordChannelID
	}

	return &Bot{
		Session: dg,
		Channel: channel,

		logger: logger,
	}, nil
}

func (b *Bot) Logger() *log.Logger {
	return b.logger
}

func (b *Bot) Start() error {
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		b.logger.Info("Discord bot logged in", "user", fmt.Sprintf("%s#%s", s.State.User.Username, s.State.User.Discriminator))
	})

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	b.logger.Info("Discord connection established")
	return nil
}

func (b *Bot) Stop() error {
	b.logger.Info("Closing Discord connection")
	return b.Session.Close()
}
