package telegram

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Bot struct {
	Api *gotgbot.Bot
}

func NewBot(token string) (*Bot, error) {
	api, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Api: api,
	}, nil
}
