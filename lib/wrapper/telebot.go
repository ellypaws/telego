package wrapper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/telebot.v4"
)

type edit interface {
	*telebot.ReplyMarkup | telebot.Location | inputtable | string
}

type inputtable interface {
	*telebot.Photo | *telebot.Audio | *telebot.Document | *telebot.Video | *telebot.Animation | *telebot.PaidInputtable
}

// Options using [telebot.Bot.extractOptions]
type Options interface {
	*telebot.SendOptions | *telebot.ReplyMarkup | *telebot.ReplyParams | *telebot.Topic | telebot.Option | telebot.ParseMode | telebot.Entities
}

func Edit[w edit, o Options](b *telebot.Bot, message *telebot.Message, what w, opts ...o) (*telebot.Message, error) {
	anyOpts := make([]any, len(opts))
	for i, opt := range opts {
		anyOpts[i] = opt
	}
	return b.Edit(message, what, anyOpts...)
}

type Sendable interface {
	*telebot.Game | *telebot.LocationResult | *telebot.VenueResult |
		*telebot.Photo | *telebot.Audio | *telebot.Document |
		*telebot.Video | *telebot.Animation | *telebot.Voice |
		*telebot.VideoNote | *telebot.Sticker | *telebot.Location |
		*telebot.Venue | *telebot.Dice | *telebot.Invoice | *telebot.Poll | String | string
}

type String string

func (s String) Send(b *telebot.Bot, to telebot.Recipient, opt *telebot.SendOptions) (*telebot.Message, error) {
	params := map[string]string{
		"chat_id": to.Recipient(),
		"text":    string(s),
	}
	embedSendOptions(params, opt)

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	return extractMessage(data)
}

func GetParsed(v any) string {
	switch sendable := v.(type) {
	case *telebot.Photo:
		return sendable.Caption
	case *telebot.Document:
		return sendable.Caption
	case *telebot.Poll:
		return sendable.Question
	case String:
		return string(sendable)
	case string:
		return sendable
	default:
		return ""
	}
}

func Send[s Sendable, o Options](b *telebot.Bot, to telebot.Recipient, what s, opts ...o) (*telebot.Message, error) {
	anyOpts := make([]any, len(opts))
	for i, opt := range opts {
		anyOpts[i] = opt
	}
	return b.Send(to, what, anyOpts...)
}

func embedSendOptions(params map[string]string, opt *telebot.SendOptions) {
	if opt == nil {
		return
	}

	if opt.ReplyTo != nil && opt.ReplyTo.ID != 0 {
		params["reply_to_message_id"] = strconv.Itoa(opt.ReplyTo.ID)
	}

	if opt.DisableWebPagePreview {
		params["disable_web_page_preview"] = "true"
	}

	if opt.DisableNotification {
		params["disable_notification"] = "true"
	}

	if opt.ParseMode != telebot.ModeDefault {
		params["parse_mode"] = opt.ParseMode
	}

	if len(opt.Entities) > 0 {
		delete(params, "parse_mode")
		entities, _ := json.Marshal(opt.Entities)

		if params["caption"] != "" {
			params["caption_entities"] = string(entities)
		} else {
			params["entities"] = string(entities)
		}
	}

	if opt.AllowWithoutReply {
		params["allow_sending_without_reply"] = "true"
	}

	if opt.ReplyMarkup != nil {
		processButtons(opt.ReplyMarkup.InlineKeyboard)
		replyMarkup, _ := json.Marshal(opt.ReplyMarkup)
		params["reply_markup"] = string(replyMarkup)
	}

	if opt.Protected {
		params["protect_content"] = "true"
	}

	if opt.ThreadID != 0 {
		params["message_thread_id"] = strconv.Itoa(opt.ThreadID)
	}

	if opt.HasSpoiler {
		params["has_spoiler"] = "true"
	}

	if opt.BusinessConnectionID != "" {
		params["business_connection_id"] = opt.BusinessConnectionID
	}

	if opt.EffectID != "" {
		params["message_effect_id"] = opt.EffectID
	}
}

func processButtons(keys [][]telebot.InlineButton) {
	if keys == nil || len(keys) < 1 || len(keys[0]) < 1 {
		return
	}

	for i := range keys {
		for j := range keys[i] {
			key := &keys[i][j]
			if key.Unique != "" {
				// Format: "\f<callback_name>|<data>"
				data := key.Data
				if data == "" {
					key.Data = "\f" + key.Unique
				} else {
					key.Data = "\f" + key.Unique + "|" + data
				}
			}
		}
	}
}

// extractMessage extracts common Message result from given data.
// Should be called after extractOk or b.Raw() to handle possible errors.
func extractMessage(data []byte) (*telebot.Message, error) {
	var resp struct {
		Result *telebot.Message
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		var resp struct {
			Result bool
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, wrapError(err)
		}
		if resp.Result {
			return nil, telebot.ErrTrueResult
		}
		return nil, wrapError(err)
	}
	return resp.Result, nil
}

// wrapError returns new wrapped telebot-related error.
func wrapError(err error) error {
	return fmt.Errorf("telebot: %w", err)
}
