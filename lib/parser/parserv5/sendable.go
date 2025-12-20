package parserv5

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"telegram-discord/lib"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/telebot.v4"
)

type parser = func(text string) string

func Sendable(s *discordgo.Session, m *discordgo.Message, p parser) (any, error) {
	if p == nil {
		p = Parser(s, m)
	}

	if m.Poll != nil {
		return p(m.Poll.Question.Text), nil
	}

	if user := lib.GetUser(m); user != nil && !user.Bot {
		displayName := user.DisplayName()
		mentioned, err := regexp.Compile(fmt.Sprintf(`^\*%s\*: `, regexp.QuoteMeta(displayName)))
		if err == nil && !mentioned.MatchString(m.Content) && displayName != "" {
			m.Content = fmt.Sprintf("*%s*: %s", displayName, m.Content)
		}
	}

	if len(m.Embeds) > 0 {
		for _, embed := range m.Embeds {
			if embed.Image != nil {
				reader, err := lib.DefaultCache.Get(embed.Image.URL)
				if err != nil {
					return nil, err
				}
				return &telebot.Photo{
					File:    telebot.FromReader(bytes.NewReader(reader)),
					Caption: formatEmbedToMarkdownV2(embed, p),
				}, nil
			}
			if embed.Thumbnail != nil {
				reader, err := lib.DefaultCache.Get(embed.Thumbnail.URL)
				if err != nil {
					return nil, err
				}
				return &telebot.Photo{
					File:    telebot.FromReader(bytes.NewReader(reader)),
					Caption: formatEmbedToMarkdownV2(embed, p),
				}, nil
			}
		}

		return formatEmbedsToMarkdownV2(m.Embeds, p), nil
	}

	for _, attachment := range m.Attachments {
		reader, err := lib.DefaultCache.Get(attachment.URL)
		if err != nil {
			return nil, err
		}

		if isImage(attachment.ContentType) {
			return &telebot.Photo{
				File:    telebot.FromReader(bytes.NewReader(reader)),
				Caption: p(m.Content),
			}, nil
		}

		return &telebot.Document{
			File:    telebot.FromReader(bytes.NewReader(reader)),
			Caption: p(m.Content),
		}, nil
	}

	return p(m.Content), nil
}

func formatEmbedToMarkdownV2(e *discordgo.MessageEmbed, p parser) string {
	var text strings.Builder

	if e.Title != "" {
		text.WriteString(fmt.Sprintf("*%s*\n", p(e.Title)))
	}

	if e.Description != "" {
		text.WriteString(fmt.Sprintf("%s\n", p(e.Description)))
	}

	for _, field := range e.Fields {
		text.WriteString(fmt.Sprintf("\n*%s:*\n%s\n", p(field.Name), p(field.Value)))
	}

	if e.Footer != nil {
		text.WriteString(fmt.Sprintf("\n_%s_", p(e.Footer.Text)))
	}

	return text.String()
}

func formatEmbedsToMarkdownV2(embeds []*discordgo.MessageEmbed, p parser) string {
	var text strings.Builder
	for i, e := range embeds {
		text.WriteString(formatEmbedToMarkdownV2(e, p))
		if i < len(embeds)-1 {
			text.WriteString("\n\n")
		}
	}
	return text.String()
}

func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
