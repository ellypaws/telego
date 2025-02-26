package lib

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

func GetUsername(entities ...any) string {
	for _, entity := range entities {
		v := reflect.ValueOf(entity)
		if v.Kind() == reflect.Pointer && v.IsNil() {
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
		v := reflect.ValueOf(entity)
		if v.Kind() == reflect.Pointer && v.IsNil() {
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

	channel, err := s.State.Channel(id)
	if err != nil {
		if errors.Is(err, discordgo.ErrStateNotFound) {
			channel, err = s.Channel(id)
			if err != nil {
				return fmt.Sprintf("unknown")
			}
			err = s.State.ChannelAdd(channel)
			if err != nil {
				return fmt.Sprintf("%s [%v]", channel.Name, err)
			}
		} else {
			return fmt.Sprintf("unknown")
		}
	}

	return channel.Name
}

func ChannelNameID(s *discordgo.Session, id string) string {
	if s == nil {
		return fmt.Sprintf("unknown (%s)", id)
	}

	channel, err := s.State.Channel(id)
	if err != nil {
		if errors.Is(err, discordgo.ErrStateNotFound) {
			channel, err = s.Channel(id)
			if err != nil {
				return fmt.Sprintf("unknown (%s)", id)
			}
			err = s.State.ChannelAdd(channel)
			if err != nil {
				return fmt.Sprintf("%s (%s) [%v]", channel.Name, channel.ID, err)
			}
		} else {
			return fmt.Sprintf("unknown (%s)", id)
		}
	}

	return fmt.Sprintf("%s (%s)", channel.Name, channel.ID)
}

func GetReference(logger *log.Logger, s *discordgo.Session, m *discordgo.MessageCreate) (*discordgo.Message, error) {
	logger.Debug(
		"Processing message with reference",
		"message_id", m.ID,
		"reference_id", m.MessageReference.MessageID,
		"author", GetUsername(m.MessageReference),
	)
	retrieve, err := s.State.Message(m.MessageReference.ChannelID, m.MessageReference.MessageID)
	if err != nil {
		if errors.Is(err, discordgo.ErrStateNotFound) {
			logger.Warn(
				"Message not found in state, attempting to retrieve",
				"channel", ChannelNameID(s, m.MessageReference.ChannelID),
				"message_id", m.MessageReference.MessageID,
				"author", GetUsername(m.MessageReference),
			)
			retrieve, err = s.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
			if err != nil {
				logger.Error(
					"Failed to retrieve referenced message",
					"error", err,
					"channel", ChannelNameID(s, m.MessageReference.ChannelID),
					"message_id", m.MessageReference.MessageID,
					"author", GetUsername(m.MessageReference),
				)
				return nil, err
			}
			err = s.State.MessageAdd(retrieve)
			if err != nil {
				logger.Warn(
					"Failed to add referenced message to state",
					"error", err,
					"channel", ChannelNameID(s, m.MessageReference.ChannelID),
					"message_id", m.MessageReference.MessageID,
					"author", GetUsername(m.MessageReference),
				)
			}
		} else {
			logger.Error(
				"Failed to retrieve referenced message",
				"error", err,
				"channel", ChannelNameID(s, m.MessageReference.ChannelID),
				"message_id", m.MessageReference.MessageID,
				"author", GetUsername(m.MessageReference),
			)
			return nil, err
		}
	}
	return retrieve, nil
}

func Or[T any](item ...*T) *T {
	for _, i := range item {
		if i != nil {
			return i
		}
	}
	return nil
}

func ErrorEmbed(handler string, errorContent ...any) []*discordgo.MessageEmbed {
	return []*discordgo.MessageEmbed{
		{
			Type: discordgo.EmbedTypeRich,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Error",
					Value:  formatError(errorContent),
					Inline: false,
				},
				{
					Name:   "Handler",
					Value:  handler,
					Inline: false,
				},
			},
			Color: 15548997,
		},
	}
}

func formatError(errorContent ...any) string {
	if errorContent == nil || len(errorContent) < 1 {
		errorContent = []any{"An unknown error has occurred"}
	}

	var errors []string
	for _, content := range errorContent {
		switch content := content.(type) {
		case string:
			errors = append(errors, content)
		case []string:
			errors = append(errors, content...)
		case error:
			errors = append(errors, content.Error())
		case []any:
			errors = append(errors, formatError(content...)) // Recursively format the error
		default:
			errors = append(errors, fmt.Sprintf("An unknown error has occured\nReceived: %v", content))
		}
	}

	errorString := strings.Join(errors, "\n")
	if len(errors) > 1 {
		errorString = fmt.Sprintf("Multiple errors have occurred:\n%s", errorString)
	}

	return errorString
}
