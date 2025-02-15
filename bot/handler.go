package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerDiscordHandlers() {
	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.ChannelID != b.Discord.Channel.ID {
			return
		}

		var forwardText string
		if m.Content != "" {
			forwardText += fmt.Sprintf("**%s**: %s\n", m.Author.Username, m.Content)
		}

		for _, embed := range m.Embeds {
			if embed.Title != "" {
				forwardText += fmt.Sprintf("\n**%s**", embed.Title)
			}
			if embed.Description != "" {
				forwardText += fmt.Sprintf("\n%s\n", embed.Description)
			}
			if embed.Image != nil {
				forwardText += fmt.Sprintf("\n%s\n", embed.Image.URL)
			}
		}

		if forwardText == "" {
			return
		}

		err := b.Telegram.Send(forwardText)
		if err != nil {
			log.Printf("Error forwarding message to Telegram: %v", err)
		}
	})
}
