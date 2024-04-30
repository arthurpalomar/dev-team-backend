package main

import (
	"log"
	"os"

	"test/internal/telegram"
)

func main() {
	// Get token from the environment variable
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatalln("TELEGRAM_TOKEN is not set")
	}

	id := int64(-4104756076)

	// Create a new bot
	bot, err := telegram.NewBot(token)
	if err != nil {
		log.Fatalln(err)
	}

	// Send a message
	_, err = bot.Api.SendMessage(id, "Hello, World!", nil)
}
