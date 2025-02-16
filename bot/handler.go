package bot

import (
	"telegram-discord/bot/parser"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			b.Discord.Logger().Warn("Skipping message - self message")
			return
		}

		if b.Discord.Channel == "" {
			b.Discord.Logger().Warn("Skipping message - Discord channel not registered")
			return
		}

		if b.Telegram.Channel == 0 {
			b.Telegram.Logger().Warn("Skipping message - Telegram channel not registered")
			return
		}

		if m.ChannelID != b.Discord.Channel {
			b.Discord.Logger().Debug(
				"Skipping message - wrong channel",
				"received_channel", m.ChannelID,
				"target_channel", b.Discord.Channel,
				"author", m.Author.Username,
			)
			return
		}

		var (
			message *discordgo.Message = m.Message
		)

		if m.MessageReference != nil {
			b.Discord.Logger().Debug(
				"Processing message with reference",
				"message_id", m.ID,
				"reference_id", m.MessageReference.MessageID,
				"author", m.Author.Username,
			)

			retrieve, err := b.Discord.Session.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
			if err != nil {
				b.Discord.Logger().Error(
					"Failed to retrieve referenced message",
					"error", err,
					"channel_id", m.MessageReference.ChannelID,
					"message_id", m.MessageReference.MessageID,
					"author", m.Author.Username,
				)
				return
			}
			message = retrieve
		}

		toSend := parser.Sendable(s, message)
		if toSend == nil {
			b.Telegram.Logger().Debug(
				"Skipping message - no content to forward",
				"message_id", message.ID,
				"channel_id", message.ChannelID,
				"author", message.Author.Username,
			)
			return
		}

		b.Discord.Logger().Info(
			"Forwarding message to Telegram",
			"message_id", message.ID,
			"channel_id", message.ChannelID,
			"author", message.Author.Username,
			"content_length", len(message.Content),
		)

		err := b.Telegram.Send(toSend)
		if err != nil {
			b.Discord.Logger().Error(
				"Failed to forward message to Telegram",
				"error", err,
				"message_id", message.ID,
				"channel_id", message.ChannelID,
				"author", message.Author.Username,
				"content_length", len(message.Content),
			)
		} else {
			b.Discord.Logger().Info(
				"Successfully forwarded message to Telegram",
				"message_id", message.ID,
				"channel_id", message.ChannelID,
				"author", message.Author.Username,
				"content_length", len(message.Content),
			)
		}
	})
}
