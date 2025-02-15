package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/telebot.v4"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func sendable(message *discordgo.Message) any {
	// Handle messages with attachments
	if len(message.Attachments) > 0 {
		for _, attachment := range message.Attachments {
			// Check if attachment is an image
			if isImage(attachment.ContentType) {
				return &telebot.Photo{
					File:    telebot.FromURL(attachment.URL),
					Caption: escapeMarkdownV2(message.Content),
				}
			}
			// For other types of attachments, send as document
			return &telebot.Document{
				File:    telebot.FromURL(attachment.URL),
				Caption: escapeMarkdownV2(message.Content),
			}
		}
	}

	if len(message.Embeds) > 0 {
		// Handle embeds with images
		for _, embed := range message.Embeds {
			if embed.Image != nil {
				caption := formatEmbedToMarkdownV2(embed)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Image.URL),
					Caption: caption,
				}
			}
			if embed.Thumbnail != nil {
				caption := formatEmbedToMarkdownV2(embed)
				return &telebot.Photo{
					File:    telebot.FromURL(embed.Thumbnail.URL),
					Caption: caption,
				}
			}
		}

		// If no images in embeds, send as text
		text := formatEmbedsToMarkdownV2(message.Embeds)
		return text
	}

	// Default to plain text message
	return escapeMarkdownV2(message.Content)
}

func formatEmbedToMarkdownV2(embed *discordgo.MessageEmbed) string {
	var text strings.Builder

	// Add title if present
	if embed.Title != "" {
		text.WriteString(fmt.Sprintf("*%s*\n", escapeMarkdownV2(embed.Title)))
	}

	// Add description if present
	if embed.Description != "" {
		text.WriteString(fmt.Sprintf("%s\n", escapeMarkdownV2(embed.Description)))
	}

	// Add fields
	for _, field := range embed.Fields {
		text.WriteString(fmt.Sprintf("\n*%s:*\n%s\n", escapeMarkdownV2(field.Name), escapeMarkdownV2(field.Value)))
	}

	// Add footer if present
	if embed.Footer != nil {
		text.WriteString(fmt.Sprintf("\n_%s_", escapeMarkdownV2(embed.Footer.Text)))
	}

	return text.String()
}

func formatEmbedsToMarkdownV2(embeds []*discordgo.MessageEmbed) string {
	var text strings.Builder
	for i, embed := range embeds {
		text.WriteString(formatEmbedToMarkdownV2(embed))
		if i < len(embeds)-1 {
			text.WriteString("\n\n")
		}
	}
	return text.String()
}

func escapeMarkdownV2(text string) string {
	// Characters that need escaping in MarkdownV2: _ * [ ] ( ) ~ ` > # + - = | { } . !
	text = parseTimestamps(text)
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!", "@"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

var timestamp = regexp.MustCompile(`<t:(\d+):([tTdDfFR])>`)

func parseTimestamps(text string) string {
	return timestamp.ReplaceAllStringFunc(text, func(match string) string {
		parts := timestamp.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		parseInt, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return match
		}
		return fmt.Sprintf("**%s**", formatTimestamp(parseInt, parts[2]))
	})
}

// formatTimestamp converts a Discord timestamp to a readable format
// style can be:
// t: Short Time (e.g., 9:41 PM)
// T: Long Time (e.g., 9:41:30 PM)
// d: Short Date (e.g., 30/06/2023)
// D: Long Date (e.g., June 30, 2023)
// f: Short Date/Time (e.g., June 30, 2023 9:41 PM)
// F: Long Date/Time (e.g., Friday, June 30, 2023 9:41 PM)
// R: Relative Time (e.g., 2 hours ago, in 3 days)
func formatTimestamp(timestamp int64, style string) string {
	t := time.Unix(timestamp, 0)

	switch style {
	case "t":
		return t.Format("3:04 PM")
	case "T":
		return t.Format("3:04:05 PM")
	case "d":
		return t.Format("02/01/2006")
	case "D":
		return t.Format("January 02, 2006")
	case "f":
		return t.Format("January 02, 2006 3:04 PM")
	case "F":
		return t.Format("Monday, January 02, 2006 3:04 PM")
	case "R":
		return formatRelativeTime(t) + fmt.Sprintf(" (%s)", t.Format("January 02, 2006 3:04 PM"))
	default:
		return t.Format(time.RFC3339)
	}
}

// formatRelativeTime returns a human-readable relative time string
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		diff = -diff
		switch {
		case diff < time.Minute:
			return "in a few seconds"
		case diff < time.Hour:
			return fmt.Sprintf("in %d minutes", int(diff.Minutes()))
		case diff < 24*time.Hour:
			return fmt.Sprintf("in %d hours", int(diff.Hours()))
		case diff < 30*24*time.Hour:
			return fmt.Sprintf("in %d days", int(diff.Hours()/24))
		case diff < 365*24*time.Hour:
			return fmt.Sprintf("in %d months", int(diff.Hours()/(24*30)))
		default:
			return fmt.Sprintf("in %d years", int(diff.Hours()/(24*365)))
		}
	}

	switch {
	case diff < time.Minute:
		return "a few seconds ago"
	case diff < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%d months ago", int(diff.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%d years ago", int(diff.Hours()/(24*365)))
	}
}
