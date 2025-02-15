package bot

import (
	"telegram-discord/bot/parser"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if b.Discord.Channel == nil {
			b.Discord.Logger().Info("Message ignored - no channel registered")
			return
		}

		if m.ChannelID != *b.Discord.Channel {
			b.Discord.Logger().Debug("Message ignored - wrong channel",
				"received_channel", m.ChannelID,
				"target_channel", *b.Discord.Channel)
			return
		}

		var (
			message *discordgo.Message = m.Message
		)

		if m.MessageReference != nil {
			retrieve, err := b.Discord.Session.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
			if err != nil {
				b.Discord.Logger().Error("Failed to retrieve message reference",
					"error", err,
					"channel_id", m.MessageReference.ChannelID,
					"message_id", m.MessageReference.MessageID)
				return
			}
			message = retrieve
		}

		toSend := parser.Sendable(s, message)
		if toSend == nil {
			b.Telegram.Logger().Info("Message ignored - no content to send",
				"message_id", message.ID,
				"channel_id", message.ChannelID)
			return
		}

		err := b.Telegram.Send(toSend)
		if err != nil {
			b.Telegram.Logger().Error("Failed to forward message to Telegram",
				"error", err,
				"message_id", message.ID,
				"channel_id", message.ChannelID)
		} else {
			b.Telegram.Logger().Info("Message forwarded to Telegram",
				"message_id", message.ID,
				"channel_id", message.ChannelID,
				"author", message.Author.Username)
		}
	})
}
