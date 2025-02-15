package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"telegram-discord/bot/discord"
	"telegram-discord/bot/telegram"

	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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

	b.Discord.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.ChannelID != b.Discord.Channel.ID {
			return
		}

		var forwardText string
		if m.Content != "" {
			forwardText += fmt.Sprintf("**%s**: %s\n", m.Author.Username, m.Content)
		}

		for _, embed := range m.Embeds {
			if embed.Title != "" {
				forwardText += fmt.Sprintf("\n**%s**", embed.Title)
			}
			if embed.Description != "" {
				forwardText += fmt.Sprintf("\n%s\n", embed.Description)
			}
		}

		if forwardText == "" {
			return
		}

		msg := tgbotapi.NewMessage(b.Telegram.Channel, forwardText)
		_, err := b.Telegram.Bot.Send(msg)
		if err != nil {
			log.Printf("Error sending message to Telegram: %v", err)
		}
	})

	log.Println("Discord to Telegram mirroring bot is running. Press CTRL+C to exit.")
	return nil
}

func (b *Bot) Wait() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}

func (b *Bot) Shutdown() error {
	log.Println("Shutting down...")
	return b.Discord.Session.Close()
}
