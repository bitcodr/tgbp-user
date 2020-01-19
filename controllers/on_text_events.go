package controllers

import (
	"database/sql"
	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	"github.com/amiraliio/tgbp-user/models"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
)

func onTextEvents(app *config.App, bot *tb.Bot) {

	bot.Handle(tb.OnText, func(message *tb.Message) {
		if !message.Private() {
			return
		}

		db := app.DB()
		defer db.Close()
		lastState := GetUserLastState(db, app, bot, message, message.Sender.ID)

		//check incoming text
		incomingMessage := message.Text
		switch {
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE")+"_"):
			goto SendReply
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.REPLY_BY_DM")+"_"):
			goto SanedDM
		case strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.COMPOSE_MESSAGE")+"_"):
			goto NewMessageGroupHandler
		default:
			goto CheckState
		}

	SendReply:
		if generalEventsHandler(app, bot, message, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE"),
			Command:    config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY"),
			Controller: "SendReply",
		}) {
			Init(app, bot, true)
		}
		goto END

	SanedDM:
		if generalEventsHandler(app, bot, message, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_BY_DM"),
			Command:    config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY_DM"),
			Controller: "SanedDM",
		}) {
			Init(app, bot, true)
		}
		goto END

	NewMessageGroupHandler:
		if generalEventsHandler(app, bot, message, &Event{
			UserState:  config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP"),
			Command:    config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_COMPOSE_IN_GROUP"),
			Controller: "NewMessageGroupHandler",
		}) {
			Init(app, bot, true)
		}
		goto END

		/////////////////////////////////////////////
		////////check the user state////////////////
		///////////////////////////////////////////
	CheckState:
		switch {
		case lastState.State == config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP") || strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.COMPOSE_MESSAGE")+"_"):
			goto SaveAndSendMessage
		case lastState.State == config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") || strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE")+"_"):
			goto SendAndSaveReplyMessage
		case lastState.State == config.LangConfig.GetString("STATE.REPLY_BY_DM") || strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.REPLY_BY_DM")+"_"):
			goto SendAndSaveDirectMessage
		case lastState.State == config.LangConfig.GetString("STATE.ANSWER_TO_DM") || strings.Contains(incomingMessage, config.LangConfig.GetString("STATE.ANSWER_TO_DM")+"_"):
			goto SendAnswerAndSaveDirectMessage
		// case lastState.State == config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL") || incomingMessage == joinCompanyChannels.Text:
		// 	goto RegisterUserWithemail
		case lastState.State == config.LangConfig.GetString("STATE.CONFIRM_REGISTER_COMPANY"):
			goto ConfirmRegisterCompanyRequest
		case lastState.State == config.LangConfig.GetString("STATE.REGISTER_USER_FOR_COMPANY"):
			goto ConfirmRegisterUserForTheCompany
		case lastState.State == config.LangConfig.GetString("STATE.EMAIL_FOR_USER_REGISTRATION"):
			goto RegisterUserWithEmailAndCode
		default:
			bot.Send(message.Sender, "Your message "+message.Text+" is not being processed or sent to any individual, channel or group.")
			goto END
		}

	SaveAndSendMessage:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP"),
			Command:    config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_",
			Controller: "SaveAndSendMessage",
		}) {
			Init(app, bot, true)
		}
		goto END

	SendAndSaveReplyMessage:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE"),
			Command:    config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY"),
			Controller: "SendAndSaveReplyMessage",
		}) {
			Init(app, bot, true)
		}
		goto END

	SendAndSaveDirectMessage:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_BY_DM"),
			Command:    config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY_DM"),
			Controller: "SendAndSaveDirectMessage",
		}) {
			Init(app, bot, true)
		}
		goto END

	SendAnswerAndSaveDirectMessage:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.ANSWER_TO_DM"),
			Command:    config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_",
			Controller: "SendAnswerAndSaveDirectMessage",
		}) {
			Init(app, bot, true)
		}
		goto END

	// RegisterUserWithemail:
	// 	if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
	// 		UserState:  config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"),
	// 		Command:    joinCompanyChannels.Text,
	// 		Controller: "RegisterUserWithemail",
	// 	}) {
	// 		Init(app, bot, true)
	// 	}
	// 	goto END

	ConfirmRegisterCompanyRequest:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.CONFIRM_REGISTER_COMPANY"),
			Controller: "ConfirmRegisterCompanyRequest",
		}) {
			Init(app, bot, true)
		}
		goto END

	ConfirmRegisterUserForTheCompany:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.REGISTER_USER_FOR_COMPANY"),
			Controller: "ConfirmRegisterUserForTheCompany",
		}) {
			Init(app, bot, true)
		}
		goto END

	RegisterUserWithEmailAndCode:
		if inlineOnTextEventsHandler(app, bot, message, db, lastState, &Event{
			UserState:  config.LangConfig.GetString("STATE.EMAIL_FOR_USER_REGISTRATION"),
			Controller: "RegisterUserWithEmailAndCode",
		}) {
			Init(app, bot, true)
		}
		goto END

	END:
	})
}

func inlineOnTextEventsHandler(app *config.App, bot *tb.Bot, message *tb.Message, db *sql.DB, lastState *models.UserLastState, request *Event) bool {
	var result bool
	switch {
	case request.Controller == "RegisterUserWithemail":
		helpers.Invoke(new(BotService), &result, request.Controller, db, app, bot, message, request, lastState, strings.TrimSpace(message.Text), message.Sender.ID)
	default:
		helpers.Invoke(new(BotService), &result, request.Controller, db, app, bot, message, request, lastState)
	}
	return result
}
