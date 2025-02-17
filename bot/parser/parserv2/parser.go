package parserv2

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
)

// Parse is the entry point. It first pre-processes the text to replace Discord-specific
// markers (timestamps, mentions) with temporary markers, builds the AST, then renders it.
func Parse(s *discordgo.Session, text string) string {
	// Preprocess: replace Discord timestamps and mentions with marker strings.
	text = preprocess(text, s)
	// Build AST from the resulting text.
	nodes := buildAST(text)
	// Render AST back into a Telegram MarkdownV2 string.
	return renderNodes(nodes)
}

func AST(text string) []Node {
	text = preprocess(text, nil)
	return buildAST(text)
}

// preprocess converts tokens like <t:...> and <@...> into unique markers.
func preprocess(text string, s *discordgo.Session) string {
	text = parseTimestampsToString(text)
	text = replaceMentionsToString(s, text)
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
)

func replaceMentionsToString(s *discordgo.Session, text string) string {
	// User mentions.
	text = userRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := userRe.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		if s == nil {
			return fmt.Sprintf("[[MENTION:%s]]", matches[1])
		}
		user, err := s.User(matches[1])
		if err != nil {
			return fmt.Sprintf("[[MENTION:%s]]", matches[1])
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
			return fmt.Sprintf("[[CHANNEL:%s]]", matches[1])
		}
		channel, err := s.Channel(matches[1])
		if err != nil {
			return fmt.Sprintf("[[CHANNEL:%s]]", matches[1])
		}
		return fmt.Sprintf("[[CHANNEL:#%s]]", channel.Name)
	})
	// Additional markers (e.g. for roles) can be handled similarly.
	return text
}
