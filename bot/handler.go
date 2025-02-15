package bot

import (
	"telegram-discord/bot/parser"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if b.Discord.Channel == nil {
			b.Discord.Logger().Printf("No channel registered, ignoring message")
			return
		}

		if m.ChannelID != *b.Discord.Channel {
			b.Discord.Logger().Printf("Ignoring message from channel %s, want %s", m.ChannelID, *b.Discord.Channel)
			return
		}

		var (
			message *discordgo.Message = m.Message
		)

		if m.MessageReference != nil {
			retrieve, err := b.Discord.Session.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
			if err != nil {
				b.Discord.Logger().Printf("Error retrieving message reference: %v", err)
				return
			}
			message = retrieve
		}

		toSend := parser.Sendable(s, message)
		if toSend == nil {
			b.Telegram.Logger().Printf("No content to send")
			return
		}

		err := b.Telegram.Send(toSend)
		if err != nil {
			b.Telegram.Logger().Printf("Error forwarding message to Telegram: %v", err)
		}
		b.Telegram.Logger().Printf("Message forwarded to Telegram")
	})
}
