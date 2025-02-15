package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"telegram-discord/lib"
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
			b.logger.Info("Command already registered", "command", cmd.Name)
			continue
		}
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)
		}
		b.logger.Info("Command registered successfully", "command", cmd.Name)
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

	if err := lib.Set("DISCORD_CHANNEL_ID", channelID); err != nil {
		b.logger.Error("Failed to save channel ID to .env",
			"error", err,
			"channel_id", channelID,
		)
	}

	b.logger.Info("Discord channel registered",
		"channel_id", channelID,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username)

	b.respond(s, i, fmt.Sprintf("Successfully registered channel <#%s> for message forwarding", channelID))
}

func (b *Bot) handleUnregister(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.Channel == nil || *b.Channel != i.ChannelID {
		b.respond(s, i, "This channel is not currently registered for message forwarding")
		return
	}

	oldChannel := *b.Channel
	b.Channel = nil
	b.logger.Info("Discord channel unregistered",
		"channel_id", oldChannel,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username)

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
		b.logger.Error("Failed to respond to interaction",
			"error", err,
			"interaction_id", i.ID,
			"channel_id", i.ChannelID)
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
		b.logger.Error("Failed to respond to interaction with error",
			"error", err,
			"interaction_id", i.ID,
			"channel_id", i.ChannelID,
			"content", content)
	}
}
