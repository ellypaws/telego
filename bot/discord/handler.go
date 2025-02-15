package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) Commands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "register",
			Description: "Register a channel for message forwarding (uses current channel if no ID provided)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "The channel ID to register (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "unregister",
			Description: "Unregister the current channel from message forwarding",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "The channel ID to unregister (optional)",
					Required:    false,
				},
			},
		},
	}

	registeredCommands, err := b.Session.ApplicationCommands(b.Session.State.User.ID, "")
	if err != nil {
		return fmt.Errorf("error fetching registered commands: %w", err)
	}

	isRegistered := make(map[string]struct{})
	for _, cmd := range registeredCommands {
		isRegistered[cmd.Name] = struct{}{}
	}

	for _, cmd := range commands {
		if _, ok := isRegistered[cmd.Name]; ok {
			b.logger.Info("Command %s is already registered, skipping...", "command", cmd.Name)
			continue
		}
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)
		}
	}

	return nil
}

func (b *Bot) Handlers() {
	b.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "register":
			b.handleRegister(s, i)
		case "unregister":
			b.handleUnregister(s, i)
		}
	})
}

func (b *Bot) handleRegister(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var channelID string
	if len(i.ApplicationCommandData().Options) > 0 {
		channel := i.ApplicationCommandData().Options[0].ChannelValue(s)
		channelID = channel.ID
	} else {
		channelID = i.ChannelID
	}

	b.Channel = &channelID
	b.logger.Info("Registered new Discord channel", "channel", channelID)

	b.respond(s, i, fmt.Sprintf("Successfully registered channel <#%s> for message forwarding", channelID))
}

func (b *Bot) handleUnregister(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.Channel == nil || *b.Channel != i.ChannelID {
		b.respond(s, i, "This channel is not currently registered for message forwarding")
		return
	}

	b.Channel = nil
	b.logger.Info("Unregistered Discord channel", "id", i.ChannelID)

	b.respond(s, i, "Successfully unregistered this channel from message forwarding")
}

func (b *Bot) respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		b.logger.Errorf("error responding to interaction: %v", err)
	}
}

func (b *Bot) respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		b.logger.Errorf("error responding to interaction: %v", err)
	}
}
