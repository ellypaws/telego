package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	Channel *string
}

func New(token string, discordChannelID string) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	var channel *string
	if discordChannelID == "" {
		logger.Printf("No channel ID provided, will not be able to send messages")
	} else {
		channel = &discordChannelID
	}

	return &Bot{
		Session: dg,
		Channel: channel,
	}, nil
}

func (b *Bot) Start() error {
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %s#%s", s.State.User.Username, s.State.User.Discriminator)
	})

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	return nil
}

func (b *Bot) Stop() error {
	return b.Session.Close()
}
