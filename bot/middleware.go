package bot

import (
	"errors"
	"fmt"
	"strings"

	"telegram-discord/lib"

	"github.com/bwmarrin/discordgo"
)

// HandlerFunc is a generic event handler that processes events of type T.
type HandlerFunc[T any] func(*discordgo.Session, T) error

// Middleware is a function that wraps a HandlerFunc.
type Middleware[T any] func(HandlerFunc[T]) HandlerFunc[T]

// Chain applies the given middlewares to a handler.
// The first middleware in the slice will be the outermost.
func Chain[T any](handler HandlerFunc[T], middlewares ...Middleware[T]) func(*discordgo.Session, T) {
	h := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return func(s *discordgo.Session, event T) {
		_ = h(s, event)
	}
}

// RetryMiddleware retries the inner handler up to 'retries' times.
func RetryMiddleware[T any](b *Bot, retries int) Middleware[T] {
	return func(next HandlerFunc[T]) HandlerFunc[T] {
		return func(s *discordgo.Session, event T) error {
			var err error
			for i := 0; i < retries; i++ {
				err = next(s, event)
				if err == nil {
					return nil
				}
				if i < retries-1 {
					b.Discord.Logger().Warn(
						"Failed to handle event, retrying...",
						"error", err,
						"attempt", i+1,
						"type", fmt.Sprintf("%T", event),
					)
				}
			}
			b.Discord.Logger().Error(
				fmt.Sprintf("Failed to handle event after %d retries", retries),
				"error", err,
				"type", fmt.Sprintf("%T", event),
			)
			return err
		}
	}
}

func SkipperMiddleware[T any](b *Bot, skippers ...func(*discordgo.Session, T) error) Middleware[T] {
	return func(next HandlerFunc[T]) HandlerFunc[T] {
		return func(s *discordgo.Session, event T) error {
			for _, skipper := range skippers {
				if err := skipper(s, event); err != nil {
					b.Discord.Logger().Debug(
						"Skipping event",
						"type", fmt.Sprintf("%T", event),
						"reason", err,
					)
					return nil
				}
			}
			return next(s, event)
		}
	}
}

func SkipPrefixes(prefixes ...string) func(*discordgo.Session, *discordgo.MessageCreate) error {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		for _, prefix := range prefixes {
			if strings.HasPrefix(m.Message.Content, prefix) {
				return fmt.Errorf("message starts with prefix %q", prefix)
			}
		}
		return nil
	}
}

var (
	ErrUserNotBot = errors.New("user is not a bot")
)

func OnlyBots(s *discordgo.Session, m *discordgo.MessageCreate) error {
	user := lib.GetUser(m)
	if user == nil {
		return nil
	}

	if !user.Bot {
		return ErrUserNotBot
	}
	return nil
}

// WhitelistMiddleware allows only messages from whitelisted user IDs.
// If a message's author is not in the whitelist, the event is ignored.
func WhitelistMiddleware(whitelist map[string]bool) Middleware[*discordgo.MessageCreate] {
	return func(next HandlerFunc[*discordgo.MessageCreate]) HandlerFunc[*discordgo.MessageCreate] {
		return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
			if !whitelist[m.Author.ID] {
				// Optionally log that the user is not allowed.
				return nil
			}
			return next(s, m)
		}
	}
}

func NotifyOnErrorMiddleware[T any](ids ...string) Middleware[T] {
	return func(next HandlerFunc[T]) HandlerFunc[T] {
		return func(s *discordgo.Session, event T) error {
			handlerError := next(s, event)
			if handlerError == nil {
				return nil
			}
			for _, id := range ids {
				channel, channelErr := s.UserChannelCreate(id)
				if channelErr != nil {
					return channelErr
				}
				_, sendErr := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
					Embeds: lib.ErrorEmbed(fmt.Sprintf("%T", event), handlerError),
				})
				return sendErr
			}
			return nil
		}
	}
}
