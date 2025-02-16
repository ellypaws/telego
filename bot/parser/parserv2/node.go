package parserv2

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Node represents an AST node.
type Node interface {
	String() string
}

// TextNode holds plain text.
type TextNode struct {
	Text string
	Raw  bool
}

func (n *TextNode) String() string {
	if n.Raw {
		return n.Text
	}
	return escapeTelegram(n.Text)
}

// FormattingNode represents a formatting block (bold, italic, underline, etc.).
type FormattingNode struct {
	Format   string // e.g. "*", "_", "__", "~~" (for Discord strikethrough, rendered as "~") or "||" for spoiler.
	Children []Node
}

func (n *FormattingNode) String() string {
	var inner strings.Builder
	for _, child := range n.Children {
		inner.WriteString(child.String())
	}
	// Map Discord tokens to Telegram MarkdownV2 equivalents.
	switch n.Format {
	case "~~":
		// Discord strikethrough becomes a single '~' pair in Telegram.
		return "~" + inner.String() + "~"
	case "**":
		// Discord bold becomes '*' in Telegram.
		return "*" + inner.String() + "*"
	case "*":
		// Discord italic becomes '_' in Telegram.
		return "_" + inner.String() + "_"
	default:
		// For tokens like "*" (bold), "_" (italic) or "__" (underline) and "||" (spoiler),
		// we wrap the inner text. (Adjust as needed if Telegram requires changes.)
		return n.Format + inner.String() + n.Format
	}
}

// CodeNode represents inline code.
type CodeNode struct {
	Text string
}

func (n *CodeNode) String() string {
	// Inside inline code, escape '`' and '\' per Telegram MarkdownV2.
	text := strings.ReplaceAll(n.Text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "`", "\\`")
	return "`" + text + "`"
}

// CodeBlockNode represents a code block.
type CodeBlockNode struct {
	Text string
}

func (n *CodeBlockNode) String() string {
	text := strings.ReplaceAll(n.Text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "`", "\\`")
	return "```" + text + "```"
}

// LinkNode represents an inline link [text](url).
type LinkNode struct {
	Text string
	URL  string
}

func (n *LinkNode) String() string {
	// Escape ')' and '\' in the URL.
	url := strings.ReplaceAll(n.URL, "\\", "\\\\")
	url = strings.ReplaceAll(url, ")", "\\)")
	return fmt.Sprintf("[%s](%s)", escapeTelegram(n.Text), url)
}

// MentionNode represents a mention (user, channel, role).
type MentionNode struct {
	Text string
}

func (n *MentionNode) String() string {
	return escapeTelegram(n.Text)
}

// TimestampNode represents a Discord timestamp.
type TimestampNode struct {
	Timestamp int64
	Style     string
}

func (n *TimestampNode) String() string {
	t := time.Unix(n.Timestamp, 0).UTC()
	var formatted string
	switch n.Style {
	case "t":
		formatted = t.Format("3:04 PM MST")
	case "T":
		formatted = t.Format("3:04:05 PM MST")
	case "d":
		formatted = t.Format("02/01/2006")
	case "D":
		formatted = t.Format("January 02, 2006")
	case "f":
		formatted = t.Format("January 02, 2006 3:04 PM MST")
	case "F":
		formatted = t.Format("Monday, January 02, 2006 3:04 PM MST")
	case "R":
		return formatRelativeFull(t)
	default:
		formatted = t.Format(time.RFC3339)
	}
	return escapeTelegram(formatted)
}

func formatRelativeFull(t time.Time) string {
	return fmt.Sprintf("*%s* \\(%s\\)", formatRelativeTime(t), escapeTelegram(t.Format("January 02, 2006 3:04 PM MST")))
}

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

// escapeTelegram escapes characters special to Telegram MarkdownV2.
// Deprecated: use safeEscapeTelegram instead.
func escapeTelegram(text string) string {
	return safeEscapeTelegram(text)
	// List of characters to escape in Telegram MarkdownV2.
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

// safeEscapeTelegram escapes Telegram MarkdownV2 special characters in text
// but skips characters that are already escaped.
func safeEscapeTelegram(text string) string {
	specialChars := map[rune]bool{
		'_': true, '*': true, '[': true, ']': true, '(': true, ')': true,
		'~': true, '`': true, '>': true, '#': true, '+': true, '-': true,
		'=': true, '|': true, '{': true, '}': true, '.': true, '!': true,
	}
	runes := []rune(text)
	var sb strings.Builder
	for i, r := range runes {
		if specialChars[r] && !isEscaped(text, i) {
			sb.WriteRune('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// isEscaped returns true if the character at pos in text is escaped by an odd number of preceding backslashes.
func isEscaped(text string, pos int) bool {
	count := 0
	// Count how many consecutive backslashes precede pos.
	for pos-1 >= 0 && text[pos-1] == '\\' {
		count++
		pos--
	}
	return count%2 == 1
}

func buildAST(text string) []Node {
	// helper to find a non-escaped closing token starting at position start.
	findClosing := func(text string, start int, token string) int {
		for j := start; j < len(text); j++ {
			if strings.HasPrefix(text[j:], token) && !isEscaped(text, j) {
				return j
			}
		}
		return -1
	}

	var nodes []Node
	i := 0
	for i < len(text) {
		// Check for escaped tokens.
		if text[i] == '\\' && i+1 < len(text) {
			nodes = append(nodes, &TextNode{Text: text[i : i+2], Raw: true})
			i += 2
			continue
		}
		// (Timestamp, mention, channel, inline code, code block, and link handling remain unchanged.)
		if strings.HasPrefix(text[i:], "[[TIMESTAMP:") {
			end := strings.Index(text[i:], "]]")
			if end != -1 {
				marker := text[i : i+end+2]
				parts := strings.Split(strings.Trim(marker, "[]"), ":")
				if len(parts) == 3 {
					ts, err := strconv.ParseInt(parts[1], 10, 64)
					if err == nil {
						nodes = append(nodes, &TimestampNode{Timestamp: ts, Style: parts[2]})
						i += end + 2
						continue
					}
				}
			}
		}
		if strings.HasPrefix(text[i:], "[[MENTION:") {
			end := strings.Index(text[i:], "]]")
			if end != -1 {
				marker := text[i : i+end+2]
				content := strings.TrimPrefix(strings.TrimSuffix(marker, "]]"), "[[MENTION:")
				nodes = append(nodes, &MentionNode{Text: content})
				i += end + 2
				continue
			}
		}
		if strings.HasPrefix(text[i:], "[[CHANNEL:") {
			end := strings.Index(text[i:], "]]")
			if end != -1 {
				marker := text[i : i+end+2]
				content := strings.TrimPrefix(strings.TrimSuffix(marker, "]]"), "[[CHANNEL:")
				nodes = append(nodes, &MentionNode{Text: content})
				i += end + 2
				continue
			}
		}
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				code := text[i+1 : i+1+end]
				nodes = append(nodes, &CodeNode{Text: code})
				i += end + 2
				continue
			}
		}
		if strings.HasPrefix(text[i:], "```") {
			j := i + 3
			end := strings.Index(text[j:], "```")
			if end != -1 {
				code := text[j : j+end]
				nodes = append(nodes, &CodeBlockNode{Text: code})
				i = j + end + 3
				continue
			}
		}
		if text[i] == '[' {
			closeBracket := strings.Index(text[i:], "]")
			if closeBracket != -1 && len(text) > i+closeBracket+1 && text[i+closeBracket+1] == '(' {
				closeParen := strings.Index(text[i+closeBracket+2:], ")")
				if closeParen != -1 {
					linkText := text[i+1 : i+closeBracket]
					linkURL := text[i+closeBracket+2 : i+closeBracket+2+closeParen]
					nodes = append(nodes, &LinkNode{Text: linkText, URL: linkURL})
					i += closeBracket + closeParen + 3
					continue
				}
			}
		}
		// Multi-character formatting tokens: **, __, ||, ~~
		if i <= len(text)-2 {
			tok := text[i : i+2]
			if tok == "**" || tok == "__" || tok == "||" || tok == "~~" {
				closing := findClosing(text, i+2, tok)
				if closing == -1 {
					nodes = append(nodes, &TextNode{Text: tok})
					i += 2
					continue
				}
				innerContent := text[i+2 : closing]
				children := buildAST(innerContent)
				nodes = append(nodes, &FormattingNode{Format: tok, Children: children})
				i = closing + 2
				continue
			}
		}
		// Single-character formatting tokens: * and _
		if text[i] == '*' || text[i] == '_' {
			tok := string(text[i])
			closing := findClosing(text, i+1, tok)
			if closing == -1 {
				nodes = append(nodes, &TextNode{Text: tok})
				i++
				continue
			}
			innerContent := text[i+1 : closing]
			children := buildAST(innerContent)
			nodes = append(nodes, &FormattingNode{Format: tok, Children: children})
			i = closing + 1
			continue
		}
		// Default: treat the current character as plain text.
		nodes = append(nodes, &TextNode{Text: string(text[i])})
		i++
	}
	return nodes
}

// renderNodes concatenates the rendered output of all AST nodes.
func renderNodes(nodes []Node) string {
	var sb strings.Builder
	for _, n := range nodes {
		sb.WriteString(n.String())
	}
	return sb.String()
}
