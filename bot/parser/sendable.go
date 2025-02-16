package parser

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/telebot.v4"
)

type Parser = func(s *discordgo.Session, text string) string

func Sendable(s *discordgo.Session, m *discordgo.Message, p Parser) any {
	if p == nil {
		p = Parse
	}

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
				caption := formatEmbedToMarkdownV2(s, embed, p)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Image.URL),
					Caption: caption,
				}
			}
			if embed.Thumbnail != nil {
				caption := formatEmbedToMarkdownV2(s, embed, p)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Thumbnail.URL),
					Caption: caption,
				}
			}
		}

		text := formatEmbedsToMarkdownV2(s, m.Embeds, p)
		return text
	}

	return Parse(s, m.Content)
}

func formatEmbedToMarkdownV2(s *discordgo.Session, e *discordgo.MessageEmbed, p Parser) string {
	var text strings.Builder

	if e.Title != "" {
		text.WriteString(fmt.Sprintf("*%s*\n", p(s, e.Title)))
	}

	if e.Description != "" {
		text.WriteString(fmt.Sprintf("%s\n", p(s, e.Description)))
	}

	for _, field := range e.Fields {
		text.WriteString(fmt.Sprintf("\n*%s:*\n%s\n", p(s, field.Name), p(s, field.Value)))
	}

	if e.Footer != nil {
		text.WriteString(fmt.Sprintf("\n_%s_", p(s, e.Footer.Text)))
	}

	return text.String()
}

func formatEmbedsToMarkdownV2(s *discordgo.Session, embeds []*discordgo.MessageEmbed, p Parser) string {
	var text strings.Builder
	for i, e := range embeds {
		text.WriteString(formatEmbedToMarkdownV2(s, e, p))
		if i < len(embeds)-1 {
			text.WriteString("\n\n")
		}
	}
	return text.String()
}

func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
