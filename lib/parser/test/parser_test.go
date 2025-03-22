package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"telegram-discord/bot"
	"telegram-discord/lib/parser"
	"telegram-discord/lib/parser/parserv2"
	"telegram-discord/lib/parser/parserv3"
	"telegram-discord/lib/parser/parserv5"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var useSession = os.Getenv("USE_SESSION") == "true"

func init() {
	if !useSession {
		return
	}
	if err := godotenv.Load(); err != nil {
		panic("error loading .env file")
	}

	bot, err := bot.New(bot.Config{
		DiscordToken:      os.Getenv("DISCORD_TOKEN"),
		DiscordChannelID:  os.Getenv("DISCORD_CHANNEL_ID"),
		DiscordLogger:     nil,
		TelegramToken:     os.Getenv("TELEGRAM_TOKEN"),
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		TelegramThreadID:  os.Getenv("TELEGRAM_THREAD_ID"),
		TelegramLogger:    nil,
	})
	if err != nil {
		panic(err)
	}

	err = bot.Start()
	if err != nil {
		panic(err)
	}

	session = bot.Discord.Session
}

var session *discordgo.Session

// TestParse tests the parser.Parse function that balances and fixes markdown tokens.
func TestParse(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unclosed Bold",
			input:    "This is *bold text",
			expected: "This is *bold text*", // auto-closes the bold marker.
		},
		{
			name:     "Balanced Bold",
			input:    "This is *bold text*",
			expected: "This is *bold text*",
		},
		{
			name:     "Nested Formatting",
			input:    "*bold _italic",
			expected: "*bold _italic_*", // both italic and bold get closed in LIFO order.
		},
		{
			name:     "Inline Code Unclosed",
			input:    "Some `code",
			expected: "Some `code`", // auto-closes the inline code.
		},
		{
			name:     "Escaped Star",
			input:    "Escape \\*star*",
			expected: "Escape \\*star**", // the escaped asterisk is output literally.
		},
		{
			name:     "Code Block Unchanged",
			input:    "```\ncode block\n```",
			expected: "```\ncode block\n```", // code blocks are output verbatim.
		},
		{
			name:     "Mixed Formatting",
			input:    "*bold _italic ~strike",
			expected: "*bold _italic ~strike~_*", // all unclosed tokens are closed at the end.
		},
		{
			name:     "Unclosed Formatting Tokens",
			input:    "This is _italic and *bold",
			expected: "This is _italic and *bold*_", // both italic and bold are closed.
		},
		{
			name:     "Mixed Markdown with Code Block",
			input:    "Here is some code: ```go\nfmt.Println(\"Hello\")\n``` and some *bold",
			expected: "Here is some code: ```go\nfmt.Println(\"Hello\")\n``` and some *bold*",
		},
		{
			name: "Combined example",
			input: `**bold \*text**
_italic \*text_
__underline__
~~strikethrough~~
||spoiler||
**bold _italic bold ~~italic bold strikethrough ||italic bold strikethrough spoiler||~~ __underline italic bold___ bold**
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```\n",
			expected: `*bold \*text*
_italic \*text_
__underline__
~strikethrough~
||spoiler||
*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.Parse(session, tt.input)
			if got != tt.expected {
				t.Errorf("Parse(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestParseV2 tests parserv2.Parse,
// which first converts Discord-specific constructs (like timestamps and mentions)
// then calls the markdown parser.
func TestParseV2(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unclosed Bold",
			input:    "This is **bold text",
			expected: "This is \\*\\*bold text",
		},
		{
			name:     "Balanced Bold",
			input:    "This is **bold text**",
			expected: "This is *bold text*",
		},
		{
			name:     "Unclosed Italic",
			input:    "This is _italic text *italic text",
			expected: "This is \\_italic text \\*italic text",
		},
		{
			name:     "Balanced Italic",
			input:    "This is _italic text_ *italic text*",
			expected: "This is _italic text_ _italic text_",
		},
		{
			name:     "Broken Formatting",
			input:    "*bold _italic",
			expected: "\\*bold \\_italic",
		},
		{
			name:     "Nested Balanced Formatting",
			input:    "**bold* _italic_**",
			expected: "*bold\\* _italic_*",
		},
		{
			name:     "Inline Code Unclosed",
			input:    "Some `code",
			expected: "Some \\`code",
		},
		{
			name:     "Escaped Star",
			input:    "*Escape \\*star*",
			expected: "_Escape \\*star_",
		},
		{
			name:     "Code Block Unchanged",
			input:    "```\ncode block\n```",
			expected: "```\ncode block\n```",
		},
		{
			name:     "Strikethrough",
			input:    "This is ~~strikethrough~~ and this is ~not~",
			expected: "This is ~strikethrough~ and this is \\~not\\~",
		},
		{
			name:     "Mixed Formatting",
			input:    "*bold _italic ~strike",
			expected: "\\*bold \\_italic \\~strike",
		},
		{
			name:     "Unclosed Formatting Tokens",
			input:    "This is _italic and *bold",
			expected: "This is \\_italic and \\*bold",
		},
		{
			name:     "Mention",
			input:    "<@1234567890> <@!123456789>",
			expected: "\\@1234567890 \\@123456789",
		},
		{
			name:     "Channel Mention",
			input:    "This is a channel <#1335581350731972648> testing",
			expected: "This is a channel \\#🔔・𝙑𝙍\\-announcements testing",
		},
		{
			name:     "URL",
			input:    "This is [an example](http://www.example.com/) link.",
			expected: "This is [an example](http://www.example.com/) link\\.",
		},
		{
			name:     "Mixed Markdown with Code Block",
			input:    "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some *bold",
			expected: "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some \\*bold",
		},
		{
			name:     "Italic underline",
			input:    "_italic __underlined___",
			expected: "_italic __underlined___",
		},
		{
			name: "Combined example",
			input: `**bold \*text**
*italic \*text*
__underline__
~~strikethrough~~
||spoiler||
**bold _italic bold ~~italic bold strikethrough ||italic bold strikethrough spoiler||~~ __underline italic bold___ bold**
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
			expected: `*bold \*text*
_italic \*text_
__underline__
~strikethrough~
||spoiler||
*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parserv2.Parse(session, tt.input)
			if got != tt.expected {
				t.Errorf("DiscordToTelegramMarkdown(%q) = %q; want %q", tt.input, got, tt.expected)
				saveAsJSONFile(t, fmt.Sprintf("./bot/parser/test/debug_%s.json", tt.name), parserv2.AST(tt.input))
				saveResult(t, fmt.Sprintf("./bot/parser/test/debug_%s.txt", tt.name), got, tt.expected)
			}
		})
	}
}

func saveResult(t *testing.T, name string, got, expected string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(name), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "Got: \n%v\n\nExpected: \n%v", got, expected)
}

func saveAsJSONFile(t *testing.T, name string, v any) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(name), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
}

// TestParseV3 tests parserv3.Parse,
// which first converts Discord-specific constructs (like timestamps and mentions)
// then calls the markdown parser.
func TestParseV3(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unclosed Bold",
			input:    "This is **bold text",
			expected: "This is \\*\\*bold text",
		},
		{
			name:     "Balanced Bold",
			input:    "This is **bold text**",
			expected: "This is *bold text*",
		},
		{
			name:     "Unclosed Italic",
			input:    "This is _italic text *italic text",
			expected: "This is \\_italic text \\*italic text",
		},
		{
			name:     "Balanced Italic",
			input:    "This is _italic text_ *italic text*",
			expected: "This is _italic text_ _italic text_",
		},
		{
			name:     "Broken Formatting",
			input:    "*bold _italic",
			expected: "\\*bold \\_italic",
		},
		{
			name:     "Nested Balanced Formatting",
			input:    "**bold* _italic_**",
			expected: "*bold\\* _italic_*",
		},
		{
			name:     "Inline Code Unclosed",
			input:    "Some `code",
			expected: "Some \\`code",
		},
		{
			name:     "Escaped Star",
			input:    "*Escape \\*star*",
			expected: "_Escape \\*star_",
		},
		{
			name:     "Code Block Unchanged",
			input:    "```\ncode block\n```",
			expected: "```\ncode block\n```",
		},
		{
			name:     "Strikethrough",
			input:    "This is ~~strikethrough~~ and this is ~not~",
			expected: "This is ~strikethrough~ and this is \\~not\\~",
		},
		{
			name:     "Mixed Formatting",
			input:    "*bold _italic ~strike",
			expected: "\\*bold \\_italic \\~strike",
		},
		{
			name:     "Unclosed Formatting Tokens",
			input:    "This is _italic and *bold",
			expected: "This is \\_italic and \\*bold",
		},
		{
			name:     "Mixed Markdown with Code Block",
			input:    "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some *bold",
			expected: "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some \\*bold",
		},
		{
			name:     "Italic underline",
			input:    "_italic __underlined___",
			expected: "_italic __underlined___",
		},
		{
			name: "Combined example",
			input: `**bold \*text**
*italic \*text*
__underline__
~~strikethrough~~
||spoiler||
**bold _italic bold ~~italic bold strikethrough ||italic bold strikethrough spoiler||~~ __underline italic bold___ bold**
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
			expected: `*bold \*text*
_italic \*text_
__underline__
~strikethrough~
||spoiler||
*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parserv3.Parse(tt.input)
			if got != tt.expected {
				t.Errorf("DiscordToTelegramMarkdown(%q) = %q; want %q", tt.input, got, tt.expected)
				ast := parserv3.AST(tt.input)
				t.Logf("AST: %v", ast)
			}
		})
	}
}

// TestParseV5 tests parserv5.Parse,
// which first converts Discord-specific constructs (like timestamps and mentions)
// then calls the markdown parser.
func TestParseV5(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unclosed Bold",
			input:    "This is **bold text",
			expected: "This is \\*\\*bold text",
		},
		{
			name:     "Balanced Bold",
			input:    "This is **bold text**",
			expected: "This is *bold text*",
		},
		{
			name:     "Unclosed Italic",
			input:    "This is _italic text *italic text",
			expected: "This is \\_italic text \\*italic text",
		},
		{
			name:     "Balanced Italic",
			input:    "This is _italic text_ *italic text*",
			expected: "This is _italic text_ _italic text_",
		},
		{
			name:     "Broken Formatting",
			input:    "*bold _italic",
			expected: "\\*bold \\_italic",
		},
		{
			name:     "Nested Balanced Formatting",
			input:    "**bold* _italic_**",
			expected: "*bold\\* _italic_*",
		},
		{
			name:     "Inline Code Unclosed",
			input:    "Some `code",
			expected: "Some \\`code",
		},
		{
			name:     "Escaped Star",
			input:    "*Escape \\*star*",
			expected: "_Escape \\*star_",
		},
		{
			name:     "Code Block Unchanged",
			input:    "```\ncode block\n```",
			expected: "```\ncode block\n```",
		},
		{
			name:     "Strikethrough",
			input:    "This is ~~strikethrough~~ and this is ~not~",
			expected: "This is ~strikethrough~ and this is \\~not\\~",
		},
		{
			name:     "Mixed Formatting",
			input:    "*bold _italic ~strike",
			expected: "\\*bold \\_italic \\~strike",
		},
		{
			name:     "Unclosed Formatting Tokens",
			input:    "This is _italic and *bold",
			expected: "This is \\_italic and \\*bold",
		},
		{
			name:     "Mention",
			input:    "<@1234567890> <@!123456789>",
			expected: "\\@1234567890 \\@123456789",
		},
		{
			name:     "Channel Mention",
			input:    "This is a channel <#1335581350731972648> testing",
			expected: "This is a channel \\#🔔・𝙑𝙍\\-announcements testing",
		},
		{
			name:     "URL",
			input:    "This is [an example](http://www.example.com/) link.",
			expected: "This is [an example](http://www.example.com/) link\\.",
		},
		{
			name:     "Mixed Markdown with Code Block",
			input:    "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some *bold",
			expected: "Here is some code: ```go \nfmt.Println(\"Hello\")\n``` and some \\*bold",
		},
		{
			name:     "Italic underline",
			input:    "_italic __underlined___",
			expected: "_italic __underlined___",
		},
		{
			name: "Combined example",
			input: `**bold \*text**
*italic \*text*
__underline__
~~strikethrough~~
||spoiler||
**bold _italic bold ~~italic bold strikethrough ||italic bold strikethrough spoiler||~~ __underline italic bold___ bold**
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
			expected: `*bold \*text*
_italic \*text_
__underline__
~strikethrough~
||spoiler||
*bold _italic bold ~italic bold strikethrough ||italic bold strikethrough spoiler||~ __underline italic bold___ bold*
[inline URL](http://www.example.com/)
` + "`inline fixed-width code`" +
				"```\npre-formatted fixed-width code block\n```\n" +
				"```python\npre-formatted fixed-width code block written in the Python programming language\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parserv5.Parse(session, nil, tt.input)
			if got != tt.expected {
				t.Errorf("DiscordToTelegramMarkdown(%q) = %q; want %q", tt.input, got, tt.expected)
				saveAsJSONFile(t, fmt.Sprintf("./bot/parser/test/debug_%s.json", tt.name), parserv5.AST(tt.input))
				saveResult(t, fmt.Sprintf("./bot/parser/test/debug_%s.txt", tt.name), got, tt.expected)
			}
		})
	}
}
