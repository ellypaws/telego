package parserv5

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func Parser(s *discordgo.Session, m *discordgo.Message) func(string) string {
	return func(text string) string {
		return Parse(s, m, text)
	}
}

// Parse is the entry point. It first pre-processes the text to replace Discord-specific
// markers (timestamps, mentions) with temporary markers, builds the AST, then renders it.
func Parse(s *discordgo.Session, m *discordgo.Message, text string) string {
	// Preprocess: replace Discord timestamps and mentions with marker strings.
	text = preprocess(s, text)
	// Build AST from the resulting text.
	if !strings.HasSuffix(text, "\n") {
		text = fmt.Sprintf("%s\n", text)
	}
	nodes := buildAST(text)
	// Render AST back into a Telegram MarkdownV2 string.
	return strings.TrimSpace(renderNodes(nodes, len(text)))
}

func AST(text string) []Node {
	text = preprocess(nil, text)
	return buildAST(text)
}

// renderNodes concatenates the rendered output of all AST nodes.
func renderNodes(nodes []Node, length int) string {
	var sb strings.Builder
	sb.Grow(length)
	for _, n := range nodes {
		sb.WriteString(n.String())
	}
	return sb.String()
}

// preprocess converts tokens like <t:...> and <@...> into unique markers.
func preprocess(s *discordgo.Session, text string) string {
	text = parseTimestampsToString(text)
	text = replaceMentionsToString(s, text)
	text = removeCustomEmojis(text)
	return text
}

// Replace Discord timestamp tags with markers of the form [[TIMESTAMP:unix:style]].
var tsRe = regexp.MustCompile(`<t:(\d+):([tTdDfFR])>`)

func parseTimestampsToString(text string) string {
	return tsRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := tsRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		return fmt.Sprintf("[[TIMESTAMP:%s:%s]]", parts[1], parts[2])
	})
}

var (
	userRe    = regexp.MustCompile(`<@!?(?P<ID>\d+)>`)
	channelRe = regexp.MustCompile(`<#(?P<ID>\d+)>`)
	emojiRe   = regexp.MustCompile(`<:(?P<name>\w+):(?P<id>\d+)>`)
)

func replaceMentionsToString(s *discordgo.Session, text string) string {
	// User mentions.
	text = userRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := userRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		if s == nil {
			return match
		}
		user, err := s.User(matches[1])
		if err != nil {
			return match
		}
		return fmt.Sprintf("[[MENTION:@%s]]", user.Username)
	})
	// Channel mentions.
	text = channelRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := channelRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		if s == nil {
			return match
		}
		channel, err := s.Channel(matches[1])
		if err != nil {
			return match
		}
		return fmt.Sprintf("[[CHANNEL:#%s]]", channel.Name)
	})
	// Additional markers (e.g. for roles) can be handled similarly.
	return text
}

func removeCustomEmojis(text string) string {
	return emojiRe.ReplaceAllString(text, "")
}
