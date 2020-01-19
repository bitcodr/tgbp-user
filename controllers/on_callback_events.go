package controllers

import (
	"database/sql"
	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	"github.com/amiraliio/tgbp-user/models"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
)

func onCallbackEvents(app *config.App, bot *tb.Bot) {
	bot.Handle(tb.OnCallback, func(c *tb.Callback) {

		db := app.DB()
		defer db.Close()
		lastState := GetUserLastState(db, app, bot, c.Message, c.Sender.ID)

		//check incoming text
		incomingMessage := c.Data
		switch {
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.ANSWER_TO_DM")+"_"):
			goto SanedAnswerDM
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.COMPOSE_MESSAGE")+"_"):
			goto NewMessageGroupHandlerCallback
		default:
			goto CheckState
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

		/////////////////////////////////////////////
		////////check the user state////////////////
		///////////////////////////////////////////
	CheckState:
		switch lastState.State {
		case config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"):
			goto RegisterUserWithemail
		default:
			bot.Send(c.Sender, "Your message "+c.Data+" is not being processed or sent to any individual, channel or group. Please use inline buttons or use the /home command.")
			goto END
		}

	RegisterUserWithemail:
		if inlineOnCallbackEventsHandler(app, bot, c, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"),
			Controller: "RegisterUserWithemail",
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

func inlineOnCallbackEventsHandler(app *config.App, bot *tb.Bot, c *tb.Callback, db *sql.DB, lastState *models.UserLastState, request *Event) bool {
	var result bool
	switch {
	case request.Controller == "RegisterUserWithemail":
		helpers.Invoke(new(BotService), &result, request.Controller, db, app, bot, c.Message, request, lastState, strings.TrimSpace(c.Data), c.Sender.ID)
	default:
		helpers.Invoke(new(BotService), &result, request.Controller, app, bot, c, request)
	}
	return result
}
