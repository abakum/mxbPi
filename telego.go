package main

import (
	"fmt"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// getMessageID извлекает ID сообщения
func getMessageID(message *maxigo.Message) string {
	if message == nil {
		return ""
	}
	return message.Body.MID
}

// send error message to first chatID in args
func SendError(bot *maxigo.Client, err error) {
	if bot != nil && len(chats) > 0 && err != nil {
		text := fmt.Sprintf("💥\n`%s`", err.Error())
		bot.SendMessage(mainCtx, chats[0], &maxigo.NewMessageBody{
			Text: maxigo.Some(text),
		})
	}
}
