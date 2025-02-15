package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"telegram-discord/bot/discord"
	"telegram-discord/bot/telegram"
)

type Bot struct {
	Discord  *discord.Bot
	Telegram *telegram.Bot
}

type Config struct {
	DiscordToken     string
	DiscordChannelID string

	TelegramToken     string
	TelegramChannelID string
}

func New(config Config) (*Bot, error) {
	discordBot, err := discord.New(config.DiscordToken, config.DiscordChannelID)
	if err != nil {
		return nil, err
	}

	tgChannelID, err := strconv.ParseInt(config.TelegramChannelID, 10, 64)
	if err != nil {
		return nil, err
	}

	tgBot, err := telegram.New(config.TelegramToken, tgChannelID)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Discord:  discordBot,
		Telegram: tgBot,
	}, nil
}

func (b *Bot) Start() error {
	err := b.Discord.Start()
	if err != nil {
		return err
	}
	defer b.Discord.Session.Close()

	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.ChannelID != b.Discord.Channel.ID {
			return
		}

		var forwardText string

		// 3a. If the message has content
		if m.Content != "" {
			forwardText += fmt.Sprintf("**%s**: %s\n", m.Author.Username, m.Content)
		}

		// 3b. If there are embeds, include some embed data
		for _, embed := range m.Embeds {
			if embed.Title != "" {
				forwardText += fmt.Sprintf("\n**%s**", embed.Title)
			}
			if embed.Description != "" {
				forwardText += fmt.Sprintf("\n%s\n", embed.Description)
			}
		}

		// If there's nothing to forward (e.g. no text, no embed info), skip
		if forwardText == "" {
			return
		}

		// 4. Send to Telegram
		msg := tgbotapi.NewMessage(b.Telegram.Channel, forwardText)
		// Telegram’s Markdown parsing can be tricky, so you may need
		// to escape characters or switch to HTML parsing.
		// msg.ParseMode = "Markdown"
		_, err := b.Telegram.Bot.Send(msg)
		if err != nil {
			log.Printf("Error sending message to Telegram: %v", err)
		}
	})

	log.Println("Discord to Telegram mirroring bot is running. Press CTRL+C to exit.")

	// 6. Wait here until CTRL-C or other term signal is received.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("Shutting down...")
	return nil
}
