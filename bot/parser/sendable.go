package parser

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/telebot.v4"
	"strings"
)

func Sendable(s *discordgo.Session, m *discordgo.Message) any {
	if len(m.Attachments) > 0 {
		for _, attachment := range m.Attachments {
			if isImage(attachment.ContentType) {
				return &telebot.Photo{
					File:    telebot.FromURL(attachment.URL),
					Caption: Parse(s, m.Content),
				}
			}

			return &telebot.Document{
				File:    telebot.FromURL(attachment.URL),
				Caption: Parse(s, m.Content),
			}
		}
	}

	if len(m.Embeds) > 0 {
		for _, embed := range m.Embeds {
			if embed.Image != nil {
				caption := formatEmbedToMarkdownV2(s, embed)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Image.URL),
					Caption: caption,
				}
			}
			if embed.Thumbnail != nil {
				caption := formatEmbedToMarkdownV2(s, embed)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Thumbnail.URL),
					Caption: caption,
				}
			}
		}

		text := formatEmbedsToMarkdownV2(s, m.Embeds)
		return text
	}

	return Parse(s, m.Content)
}

func formatEmbedToMarkdownV2(s *discordgo.Session, e *discordgo.MessageEmbed) string {
	var text strings.Builder

	if e.Title != "" {
		text.WriteString(fmt.Sprintf("*%s*\n", Parse(s, e.Title)))
	}

	if e.Description != "" {
		text.WriteString(fmt.Sprintf("%s\n", Parse(s, e.Description)))
	}

	for _, field := range e.Fields {
		text.WriteString(fmt.Sprintf("\n*%s:*\n%s\n", Parse(s, field.Name), Parse(s, field.Value)))
	}

	if e.Footer != nil {
		text.WriteString(fmt.Sprintf("\n_%s_", Parse(s, e.Footer.Text)))
	}

	return text.String()
}

func formatEmbedsToMarkdownV2(s *discordgo.Session, embeds []*discordgo.MessageEmbed) string {
	var text strings.Builder
	for i, e := range embeds {
		text.WriteString(formatEmbedToMarkdownV2(s, e))
		if i < len(embeds)-1 {
			text.WriteString("\n\n")
		}
	}
	return text.String()
}

func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
