package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"gopkg.in/telebot.v4"
)

const middlewareFile = "middleware.json"

// MiddlewareConfig lists the middleware chain of each Discord handler, outermost first, and
// who to notify after the watchdog restarts the bot. It holds deployment-specific values such
// as notify targets, so it is kept out of the repo:
//
//	{
//	  "message_create": [
//	    {"type": "SkipperMiddleware", "skippers": [{"type": "SkipPrefixes", "prefixes": ["Movie: "]}]},
//	    {"type": "NotifyOnErrorMiddleware", "users": ["<user id>"], "channels": ["<channel id>"]},
//	    {"type": "RetryMiddleware", "retries": 3, "ignore": ["ErrEmptyText"]}
//	  ],
//	  "message_delete": [],
//	  "message_update": [],
//	  "notify_on_restart": {"users": ["<user id>"]}
//	}
type MiddlewareConfig struct {
	MessageCreate   MiddlewareSpecs `json:"message_create"`
	MessageDelete   MiddlewareSpecs `json:"message_delete"`
	MessageUpdate   MiddlewareSpecs `json:"message_update"`
	NotifyOnRestart NotifySpec      `json:"notify_on_restart"`
}

type MiddlewareSpecs []MiddlewareSpec

// MiddlewareSpec describes one middleware. Type selects it and decides which fields apply.
type MiddlewareSpec struct {
	Type string `json:"type"`

	Skippers []SkipperSpec `json:"skippers"` // SkipperMiddleware
	Users    []string      `json:"users"`    // NotifyOnErrorMiddleware, WhitelistMiddleware
	Channels []string      `json:"channels"` // NotifyOnErrorMiddleware
	Retries  int           `json:"retries"`  // RetryMiddleware
	Ignore   []string      `json:"ignore"`   // RetryMiddleware
}

// NotifySpec lists the users to DM and the channels to post in.
type NotifySpec struct {
	Users    []string `json:"users"`
	Channels []string `json:"channels"`
}

func (n NotifySpec) Notify(s *discordgo.Session, embeds []*discordgo.MessageEmbed, options ...discordgo.RequestOption) error {
	return errors.Join(
		notifyUsers(s, n.Users, embeds, options...),
		notifyChannels(s, n.Channels, embeds, options...),
	)
}

type SkipperSpec struct {
	Type     string   `json:"type"`
	Prefixes []string `json:"prefixes"` // SkipPrefixes
}

// ignorableErrors are the errors a RetryMiddleware spec can name under "ignore".
var ignorableErrors = map[string]error{
	"ErrEmptyText":          telebot.ErrEmptyText,
	"ErrEmptyMessage":       telebot.ErrEmptyMessage,
	"ErrMessageNotModified": telebot.ErrMessageNotModified,
	"ErrSameMessageContent": telebot.ErrSameMessageContent,
}

type Middlewares struct {
	MessageCreate []Middleware[*discordgo.MessageCreate]
	MessageDelete []Middleware[*discordgo.MessageDelete]
	MessageUpdate []Middleware[*discordgo.MessageUpdate]
}

func LoadMiddlewares(file string, logger *log.Logger) (Middlewares, NotifySpec, error) {
	f, err := os.Open(file)
	if err != nil {
		return Middlewares{}, NotifySpec{}, err
	}
	defer f.Close()

	var config MiddlewareConfig
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Middlewares{}, NotifySpec{}, fmt.Errorf("error decoding %s: %w", file, err)
	}

	var middlewares Middlewares
	if middlewares.MessageCreate, err = config.MessageCreate.Build[*discordgo.MessageCreate](logger); err != nil {
		return Middlewares{}, NotifySpec{}, fmt.Errorf("message_create: %w", err)
	}
	if middlewares.MessageDelete, err = config.MessageDelete.Build[*discordgo.MessageDelete](logger); err != nil {
		return Middlewares{}, NotifySpec{}, fmt.Errorf("message_delete: %w", err)
	}
	if middlewares.MessageUpdate, err = config.MessageUpdate.Build[*discordgo.MessageUpdate](logger); err != nil {
		return Middlewares{}, NotifySpec{}, fmt.Errorf("message_update: %w", err)
	}
	return middlewares, config.NotifyOnRestart, nil
}

func (specs MiddlewareSpecs) Build[T MessageEvent](logger *log.Logger) ([]Middleware[T], error) {
	middlewares := make([]Middleware[T], 0, len(specs))
	for i, spec := range specs {
		middleware, err := spec.Build[T](logger)
		if err != nil {
			return nil, fmt.Errorf("middleware %d (%s): %w", i, spec.Type, err)
		}
		middlewares = append(middlewares, middleware)
	}
	return middlewares, nil
}

func (spec MiddlewareSpec) Build[T MessageEvent](logger *log.Logger) (Middleware[T], error) {
	switch spec.Type {
	case "SkipperMiddleware":
		if len(spec.Skippers) == 0 {
			return nil, errors.New("needs at least one skipper")
		}
		skippers := make([]func(*discordgo.Session, T) error, 0, len(spec.Skippers))
		for _, skipper := range spec.Skippers {
			built, err := skipper.Build[T]()
			if err != nil {
				return nil, err
			}
			skippers = append(skippers, built)
		}
		return SkipperMiddleware(logger, skippers...), nil
	case "NotifyOnErrorMiddleware":
		if len(spec.Users) == 0 && len(spec.Channels) == 0 {
			return nil, errors.New("needs users or channels")
		}
		return NotifyOnErrorMiddleware(NotifyUsers[T](spec.Users...), NotifyChannels[T](spec.Channels...)), nil
	case "RetryMiddleware":
		if spec.Retries < 1 {
			return nil, errors.New("retries must be at least 1")
		}
		ignore := make([]error, 0, len(spec.Ignore))
		for _, name := range spec.Ignore {
			target, ok := ignorableErrors[name]
			if !ok {
				return nil, fmt.Errorf("unknown error %q in ignore", name)
			}
			ignore = append(ignore, target)
		}
		return RetryMiddleware[T](logger, spec.Retries, ignore...), nil
	case "WhitelistMiddleware":
		if len(spec.Users) == 0 {
			return nil, errors.New("needs users")
		}
		return WhitelistMiddleware[T](spec.Users...), nil
	default:
		return nil, fmt.Errorf("unknown middleware type %q", spec.Type)
	}
}

func (spec SkipperSpec) Build[T MessageEvent]() (func(*discordgo.Session, T) error, error) {
	switch spec.Type {
	case "SkipPrefixes":
		if len(spec.Prefixes) == 0 {
			return nil, errors.New("SkipPrefixes needs prefixes")
		}
		return SkipPrefixes[T](spec.Prefixes...), nil
	case "OnlyBots":
		return OnlyBots[T], nil
	default:
		return nil, fmt.Errorf("unknown skipper type %q", spec.Type)
	}
}
