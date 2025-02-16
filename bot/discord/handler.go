package discord

import (
	"fmt"

	"telegram-discord/lib"

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
			b.logger.Warn(
				"Command already registered",
				"command", cmd.Name,
			)
			continue
		}
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd)
		if err != nil {
			b.logger.Error(
				"Failed to register command",
				"error", err,
				"command", cmd.Name,
			)
			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)
		}
		b.logger.Info(
			"Command registered successfully",
			"command", cmd.Name,
		)
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
		b.logger.Debug(
			"Using specified channel for registration",
			"channel_id", channelID,
			"channel_name", channel.Name,
			"user", i.Member.User.Username,
		)
	} else {
		channelID = i.ChannelID
		b.logger.Debug(
			"Using current channel for registration",
			"channel_id", channelID,
			"user", i.Member.User.Username,
		)
	}

	if b.Channel == channelID {
		b.logger.Warn(
			"Channel already registered for message forwarding",
			"channel_id", channelID,
			"user", i.Member.User.Username,
		)
		b.respond(s, i, "This channel is already registered for message forwarding")
		return
	}
	b.Channel = channelID

	if err := lib.Set("DISCORD_CHANNEL_ID", channelID); err != nil {
		b.logger.Error(
			"Failed to save channel configuration",
			"error", err,
			"channel_id", channelID,
			"user", i.Member.User.Username,
		)
	}

	b.logger.Info(
		"Channel registered for message forwarding",
		"channel_id", channelID,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username,
	)

	b.respond(s, i, fmt.Sprintf("Successfully registered channel <#%s> for message forwarding", channelID))
}

func (b *Bot) handleUnregister(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.Channel == "" || b.Channel != i.ChannelID {
		b.logger.Debug(
			"Unregister attempt for non-registered channel",
			"channel_id", i.ChannelID,
			"registered_channel", b.Channel,
			"user", i.Member.User.Username,
		)
		b.respond(s, i, "This channel is not currently registered for message forwarding")
		return
	}

	oldChannel := b.Channel
	b.Channel = ""

	if err := lib.Set("DISCORD_CHANNEL_ID", ""); err != nil {
		b.logger.Error(
			"Failed to save channel configuration",
			"error", err,
			"old_channel_id", oldChannel,
			"user", i.Member.User.Username,
		)
	}

	b.logger.Info(
		"Channel unregistered from message forwarding",
		"old_channel_id", oldChannel,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username,
	)

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
		b.logger.Error(
			"Failed to respond to interaction",
			"error", err,
			"interaction_id", i.ID,
			"channel_id", i.ChannelID,
			"content", content,
			"user", i.Member.User.Username,
		)
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
		b.logger.Error(
			"Failed to respond to interaction with error",
			"error", err,
			"interaction_id", i.ID,
			"channel_id", i.ChannelID,
			"content", content,
			"user", i.Member.User.Username,
		)
	}
}
