package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Parse converts Discord markdown into Telegram markdown.
// It uses the discordgo.Session to look up users, roles, channels, etc.
func Parse(s *discordgo.Session, text string) string {
	// First, convert Discord timestamp tags like <t:1234567890:...>
	text = parseTimestamps(text)
	// Replace Discord mentions with Telegram-friendly text.
	text = replaceMentions(s, text)
	// Now run the markdown parser to balance tokens and convert nested formatting.
	return parseMarkdown(text)
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
		return formatTimestamp(parseInt, parts[2])
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
		return fmt.Sprintf("*%s* (%s)", formatRelativeTime(t), t.Format("January 02, 2006 3:04 PM"))
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

// replaceMentions uses regex to convert Discord mention syntax into Telegram style.
// It covers user (<@123456789> or <@!123456789>), role (<@&ID>) and channel (<#ID>) mentions.
func replaceMentions(s *discordgo.Session, text string) string {
	// User mentions: e.g. <@123456789> or <@!123456789>
	userRe := regexp.MustCompile(`<@!?(?P<ID>\d+)>`)
	text = userRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := userRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		user, err := s.User(matches[1])
		if err != nil {
			return fmt.Sprintf(`<@%s>`, matches[1])
		}

		return fmt.Sprintf(`@%s`, user.Username)
	})

	// Role mentions: e.g. <@&ID>
	roleRe := regexp.MustCompile(`<@&(?P<ID>\d+)>`)
	text = roleRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := roleRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		roles, err := s.GuildRoles(matches[1])
		if err != nil {
			return fmt.Sprintf(`<@&%s>`, matches[1])
		}
		for _, role := range roles {
			if role.ID == matches[1] {
				return fmt.Sprintf(`@%s`, role.Name)
			}
		}
		return fmt.Sprintf(`<@&%s>`, matches[1])
	})

	// Channel mentions: e.g. <#ID>
	channelRe := regexp.MustCompile(`<#(?P<ID>\d+)>`)
	text = channelRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := channelRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		channel, err := s.Channel(matches[1])
		if err != nil {
			return fmt.Sprintf(`<#%s>`, matches[1])
		}

		return fmt.Sprintf(`#%s`, channel.Name)
	})

	return text
}

// escapeSpecialChars escapes special characters for Telegram MarkdownV2 format
func escapeSpecialChars(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

// parseMarkdown uses a simple state‐machine to walk over the text and "balance" markdown tokens.
// It recognizes inline code (with `) and code blocks (with ```), and formatting tokens:
// bold ("*"), italic ("_"), underline ("__"), strikethrough ("~") and spoiler ("||").
// Unclosed tokens are closed automatically at the end.
func parseMarkdown(text string) string {
	var (
		sb          strings.Builder
		formatStack []string
		i           int
	)

	const (
		normal = iota
		inlineCode
		codeBlock
	)

	state := normal
	runes := []rune(text)

	sb.Grow(len(text))
	for i < len(runes) {
		switch state {
		case normal:
			// Check for code block start: triple backticks.
			if i <= len(runes)-3 && string(runes[i:i+3]) == "```" {
				sb.WriteString("```")
				i += 3
				state = codeBlock
				continue
			}
			// Check for inline code start: a single backtick.
			if runes[i] == '`' {
				sb.WriteRune('`')
				i++
				state = inlineCode
				continue
			}
			// Handle escapes: if we see a backslash, output the next character literally.
			if runes[i] == '\\' {
				if i+1 < len(runes) {
					sb.WriteRune(runes[i+1])
					i += 2
				} else {
					i++
				}
				continue
			}
			// Look ahead for multi-character tokens.
			if i <= len(runes)-2 {
				sub := string(runes[i : i+2])
				if sub == "||" || sub == "__" {
					// Toggle spoiler or underline.
					if len(formatStack) > 0 && formatStack[len(formatStack)-1] == sub {
						// It is a closing token.
						formatStack = formatStack[:len(formatStack)-1]
					} else {
						// Opening token.
						formatStack = append(formatStack, sub)
					}
					sb.WriteString(sub)
					i += 2
					continue
				}
			}
			// Check for single-character formatting tokens: *, _ and ~.
			if runes[i] == '*' || runes[i] == '_' || runes[i] == '~' {
				token := string(runes[i])
				if len(formatStack) > 0 && formatStack[len(formatStack)-1] == token {
					// Found a matching closing token.
					formatStack = formatStack[:len(formatStack)-1]
				} else {
					// Opening token.
					formatStack = append(formatStack, token)
				}
				sb.WriteRune(runes[i])
				i++
				continue
			}
			// Otherwise, escape special characters and output the current character.
			if !strings.ContainsRune("*_~`|", runes[i]) { // Don't escape formatting characters
				sb.WriteString(escapeSpecialChars(string(runes[i])))
			} else {
				sb.WriteRune(runes[i])
			}
			i++
		case inlineCode:
			// In inline code, we output everything literally until a (non-escaped) backtick is found.
			if runes[i] == '`' {
				sb.WriteRune('`')
				i++
				state = normal
			} else {
				sb.WriteRune(runes[i])
				i++
			}
		case codeBlock:
			// In a code block, we output everything until we find a triple backtick.
			if i <= len(runes)-3 && string(runes[i:i+3]) == "```" {
				sb.WriteString("```")
				i += 3
				state = normal
			} else {
				sb.WriteRune(runes[i])
				i++
			}
		}
	}
	// If inline code was left open, close it.
	if state == inlineCode {
		sb.WriteRune('`')
		state = normal
	}
	// If a code block was left open, close it.
	if state == codeBlock {
		sb.WriteString("```")
		state = normal
	}
	// Append any unclosed formatting tokens (closing them in reverse order).
	for j := len(formatStack) - 1; j >= 0; j-- {
		sb.WriteString(formatStack[j])
	}
	return sb.String()
}
