//Package controllers ...
package controllers

import (
	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	tb "gopkg.in/tucnak/telebot.v2"
)

type BotService struct{}

type Event struct {
	UserState, Command, Command1, Controller string
	Event                                    interface{}
}

func Init(app *config.App, bot *tb.Bot, state interface{}) {
	if state != nil {
		return
	}
	onActionEvents(app, bot)
	onTextEvents(app, bot)
	onCallbackEvents(app, bot)
	onMediaEvents(app, bot)

}

func generalEventsHandler(app *config.App, bot *tb.Bot, message *tb.Message, request *Event) bool {
	var result bool
	helpers.Invoke(new(BotService), &result, request.Controller, app, bot, message, request)
	return result
}

func groupMigrationHandler(app *config.App, bot *tb.Bot, from, to int64, request *Event) bool {
	var result bool
	helpers.Invoke(new(BotService), &result, request.Controller, app, bot, from, to, request)
	return result
}
