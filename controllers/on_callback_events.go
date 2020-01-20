package controllers

import (
	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
)

func onCallbackEvents(app *config.App, bot *tb.Bot) {
	bot.Handle(tb.OnCallback, func(c *tb.Callback) {

		//check incoming text
		incomingMessage := c.Data
		switch {
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.ANSWER_TO_DM")+"_"):
			goto SanedAnswerDM
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.COMPOSE_MESSAGE")+"_"):
			goto NewMessageGroupHandlerCallback
		default:
			bot.Send(c.Sender, "Your message "+c.Data+" is not being processed or sent to any individual, channel or group.")
			goto END
		}

	SanedAnswerDM:
		if onCallbackEventsHandler(app, bot, c, &Event{
			UserState:  config.LangConfig.GetString("STATE.ANSWER_TO_DM"),
			Command:    config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_",
			Controller: "SanedAnswerDM",
		}) {
			Init(app, bot, true)
		}
		goto END

	NewMessageGroupHandlerCallback:
		if onCallbackEventsHandler(app, bot, c, &Event{
			UserState:  config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP"),
			Command:    config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_",
			Controller: "NewMessageGroupHandlerCallback",
		}) {
			Init(app, bot, true)
		}
		goto END

	END:
	})
}

func onCallbackEventsHandler(app *config.App, bot *tb.Bot, c *tb.Callback, request *Event) bool {
	var result bool
	helpers.Invoke(new(BotService), &result, request.Controller, app, bot, c, request)
	return result
}
