package controllers

import (
	"github.com/amiraliio/tgbp-user/config"
	tb "gopkg.in/tucnak/telebot.v2"
)

func onMediaEvents(app *config.App, bot *tb.Bot) {

	bot.Handle(tb.OnPhoto, func(message *tb.Message) {
		if !message.Private() {
			return
		}
		db := app.DB()
		defer db.Close()
		lastState := GetUserLastState(db, app, bot, message, message.Sender.ID)

		switch {
		case lastState.State == config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP"):
			goto SaveAndSendMessage
		case lastState.State == config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE"):
			goto SendAndSaveReplyMessage
		case lastState.State == config.LangConfig.GetString("STATE.REPLY_BY_DM"):
			goto SendAndSaveDirectMessage
		case lastState.State == config.LangConfig.GetString("STATE.ANSWER_TO_DM"):
			goto SendAnswerAndSaveDirectMessage
		case lastState.State == config.LangConfig.GetString("STATE.ADD_PSEUDONYM"):
			bot.Send(message.Sender, config.LangConfig.GetString("MESSAGES.USER_NAME_IS_WRONG"))
			goto END
		default:
			bot.Send(message.Sender, "Your message is not being processed or sent to any individual, channel or group.")
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

	END:
	})

	bot.Handle(tb.OnSticker, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnDocument, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnVenue, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnVideo, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnAudio, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnLocation, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnContact, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnVideoNote, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

	bot.Handle(tb.OnVoice, func(m *tb.Message) {
		bot.Send(m.Sender, "Your message is not being processed or sent to any individual, channel or group.")
		return
	})

}
