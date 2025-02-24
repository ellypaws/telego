package lib

import (
	"fmt"
	"reflect"

	"github.com/bwmarrin/discordgo"
)

func GetUsername(entities ...any) string {
	for _, entity := range entities {
		if reflect.ValueOf(entity).IsNil() {
			continue
		}
		switch e := entity.(type) {
		case *discordgo.User:
			return e.Username
		case *discordgo.Member:
			return e.User.Username
		case *discordgo.Message:
			return GetUsername(e.Author, e.Member)
		case *discordgo.MessageCreate:
			return GetUsername(e.Message)
		case *discordgo.MessageUpdate:
			return GetUsername(e.Message, e.BeforeUpdate)
		case *discordgo.MessageDelete:
			return GetUsername(e.Message, e.BeforeDelete)
		case *discordgo.Interaction:
			return GetUsername(e.Member, e.User)
		case *discordgo.InteractionCreate:
			return GetUsername(e.Interaction)
		default:
			continue
		}
	}
	return "unknown"
}

func GetUser(entities ...any) *discordgo.User {
	for _, entity := range entities {
		if reflect.ValueOf(entity).IsNil() {
			continue
		}
		switch e := entity.(type) {
		case *discordgo.User:
			return e
		case *discordgo.Member:
			return e.User
		case *discordgo.Message:
			return GetUser(e.Author, e.Member)
		case *discordgo.MessageCreate:
			return GetUser(e.Message)
		case *discordgo.MessageUpdate:
			return GetUser(e.Message, e.BeforeUpdate)
		case *discordgo.MessageDelete:
			return GetUser(e.Message, e.BeforeDelete)
		case *discordgo.Interaction:
			return GetUser(e.Member, e.User)
		case *discordgo.InteractionCreate:
			return GetUser(e.Interaction)
		default:
			continue
		}
	}
	return nil
}

func ChannelName(s *discordgo.Session, id string) string {
	if s == nil {
		return "unknown"
	}

	channel, err := s.Channel(id)
	if err != nil {
		return "unknown"
	}

	return channel.Name
}

func ChannelNameID(s *discordgo.Session, id string) string {
	if s == nil {
		return fmt.Sprintf("unknown (%s)", id)
	}

	channel, err := s.Channel(id)
	if err != nil {
		return fmt.Sprintf("unknown (%s)", id)
	}

	return fmt.Sprintf("%s (%s)", channel.Name, channel.ID)
}

func Or[T any](item ...*T) *T {
	for _, i := range item {
		if i != nil {
			return i
		}
	}
	return nil
}
