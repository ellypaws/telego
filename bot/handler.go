package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			log.Printf("Ignoring message from bot %s", m.Author.Username)
			return
		}

		if m.ChannelID != b.Discord.Channel.ID {
			log.Printf("Ignoring message from channel %s, want %s", m.ChannelID, b.Discord.Channel.ID)
			return
		}

		var forwardText strings.Builder
		if m.Content != "" {
			forwardText.WriteString(fmt.Sprintf("**%s**: %s\n", m.Author.Username, m.Content))
		}

		for _, embed := range m.Embeds {
			if embed.Title != "" {
				forwardText.WriteString(fmt.Sprintf("\n**%s**", embed.Title))
			}
			if embed.Description != "" {
				forwardText.WriteString(fmt.Sprintf("\n%s\n", embed.Description))
			}
			if embed.Image != nil {
				forwardText.WriteString(fmt.Sprintf("\n%s\n", embed.Image.URL))
			}
		}

		if forwardText.Len() == 0 {
			log.Printf("Ignoring message with no content")
			return
		}

		log.Printf("Forwarding message to Telegram: %s", forwardText.String())
		err := b.Telegram.Send(forwardText.String())
		if err != nil {
			log.Printf("Error forwarding message to Telegram: %v", err)
		}
		log.Printf("Message forwarded to Telegram")
	})
}
