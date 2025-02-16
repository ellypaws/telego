package parserv3

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/delthas/discord-formatting"
)

// Parse is the entrypoint for parserv3. It parses the Discord-formatted text into an AST and renders it into Telegram MarkdownV2.
func Parse(text string) string {
	parser := formatting.NewParser(nil)
	ast := parser.Parse(text)
	return Render(ast)
}

func AST(text string) formatting.Node {
	parser := formatting.NewParser(nil)
	return parser.Parse(text)
}

// Render converts the AST into a Telegram MarkdownV2 string using formatting.Walk.
func Render(root formatting.Node) string {
	var b strings.Builder
	formatting.Walk(root, func(n formatting.Node, entering bool) {
		switch node := n.(type) {
		case *formatting.TextNode:
			// Text nodes are leaf nodes; only render on entering.
			if entering {
				b.WriteString(safeEscapeTelegram(node.Content))
			}
		case *formatting.BoldNode:
			if entering {
				b.WriteString("*")
			} else {
				b.WriteString("*")
			}
		case *formatting.ItalicsNode:
			if entering {
				b.WriteString("_")
			} else {
				b.WriteString("_")
			}
		case *formatting.UnderlineNode:
			if entering {
				b.WriteString("__")
			} else {
				b.WriteString("__")
			}
		case *formatting.StrikethroughNode:
			if entering {
				b.WriteString("~")
			} else {
				b.WriteString("~")
			}
		case *formatting.CodeNode:
			// Code nodes are leaf nodes; output everything on entering.
			if entering {
				b.WriteString("`")
				code := strings.ReplaceAll(node.Content, "\\", "\\\\")
				code = strings.ReplaceAll(code, "`", "\\`")
				b.WriteString(code)
				b.WriteString("`")
			}
		case *formatting.BlockQuoteNode:
			// For block quotes, prepend the marker on entering.
			if entering {
				b.WriteString("> ")
			}
		case *formatting.SpoilerNode:
			if entering {
				b.WriteString("||")
			} else {
				b.WriteString("||")
			}
		case *formatting.URLNode:
			if entering {
				if node.Mask != "" {
					b.WriteString("[")
					b.WriteString(safeEscapeTelegram(node.Mask))
					b.WriteString("](")
					b.WriteString(escapeURL(node.URL))
					b.WriteString(")")
				} else {
					b.WriteString(escapeURL(node.URL))
				}
			}
		case *formatting.EmojiNode:
			if entering {
				b.WriteString(":" + node.Text + ":")
			}
		case *formatting.ChannelMentionNode:
			if entering {
				b.WriteString("\\#" + node.ID)
			}
		case *formatting.UserMentionNode:
			if entering {
				b.WriteString("\\@" + node.ID)
			}
		case *formatting.RoleMentionNode:
			if entering {
				b.WriteString("\\@" + node.ID)
			}
		case *formatting.SpecialMentionNode:
			if entering {
				b.WriteString("\\@" + node.Mention)
			}
		case *formatting.TimestampNode:
			if entering {
				ts, err := strconv.ParseInt(node.Stamp, 10, 64)
				if err != nil {
					b.WriteString(safeEscapeTelegram(node.Stamp))
				} else {
					t := time.Unix(ts, 0)
					var formatted string
					switch node.Format {
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
						formatted = fmt.Sprintf("%s (%s)", formatRelativeTime(t), t.Format("January 02, 2006 3:04 PM MST"))
					default:
						formatted = t.Format(time.RFC3339)
					}
					b.WriteString(safeEscapeTelegram(formatted))
				}
			}
		case *formatting.HeaderNode:
			// Render headers as bold text.
			if entering {
				b.WriteString("*")
			} else {
				b.WriteString("*\n")
			}
		case *formatting.BulletListNode:
			if entering {
				b.WriteString("- ")
			} else {
				b.WriteString("\n")
			}
		default:
			// For unknown node types, do nothing.
		}
	})
	return b.String()
}

// safeEscapeTelegram escapes Telegram MarkdownV2 special characters, but skips already escaped ones.
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

// isEscaped checks whether the rune at pos is already escaped by counting preceding backslashes.
func isEscaped(text string, pos int) bool {
	count := 0
	for pos-1 >= 0 && text[pos-1] == '\\' {
		count++
		pos--
	}
	return count%2 == 1
}

// escapeURL escapes characters in URLs as required by Telegram MarkdownV2.
func escapeURL(url string) string {
	url = strings.ReplaceAll(url, "\\", "\\\\")
	url = strings.ReplaceAll(url, ")", "\\)")
	return url
}

// formatRelativeTime formats time differences in a human-readable relative style.
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
