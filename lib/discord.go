package lib

import (
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

func Or[T any](item ...*T) *T {
	for _, i := range item {
		if i != nil {
			return i
		}
	}
	return nil
}
