package controllers

import (
	"github.com/amiraliio/tgbp-user/config"
	tb "gopkg.in/tucnak/telebot.v2"
	"strings"
)

func onActionEvents(app *config.App, bot *tb.Bot) {

	bot.Handle(tb.OnChannelPost, func(message *tb.Message) {
		if generalEventsHandler(app, bot, message, &Event{
			Event:      tb.OnChannelPost,
			UserState:  config.LangConfig.GetString("STATE.REGISTER_CHANNEL"),
			Command:    config.LangConfig.GetString("COMMANDS.ENABLE_CHAT"),
			Controller: "RegisterChannel",
		}) {
			Init(app, bot, true)
		}

		if strings.Contains(message.Text, config.LangConfig.GetString("STATE.UPDATE_CHANNEL_TITLE")) {
			if generalEventsHandler(app, bot, message, &Event{
				Event:      tb.OnNewGroupTitle,
				UserState:  config.LangConfig.GetString("STATE.UPDATE_GROUP_TITLE"),
				Controller: "UpdateGroupTitle",
			}) {
				Init(app, bot, true)
			}
		}

	})

	bot.Handle(tb.OnAddedToGroup, func(message *tb.Message) {
		if generalEventsHandler(app, bot, message, &Event{
			Event:      tb.OnAddedToGroup,
			UserState:  config.LangConfig.GetString("STATE.REGISTER_GROUP"),
			Controller: "RegisterGroup",
		}) {
			Init(app, bot, true)
		}
	})

	bot.Handle(tb.OnNewGroupTitle, func(message *tb.Message) {
		if generalEventsHandler(app, bot, message, &Event{
			Event:      tb.OnNewGroupTitle,
			UserState:  config.LangConfig.GetString("STATE.UPDATE_GROUP_TITLE"),
			Controller: "UpdateGroupTitle",
		}) {
			Init(app, bot, true)
		}
	})


	bot.Handle(tb.OnMigration, func(from, to int64) {
		if groupMigrationHandler(app, bot, from, to, &Event{
			Event:      tb.OnMigration,
			UserState:  config.LangConfig.GetString("STATE.UPDATE_GROUP_ID"),
			Controller: "UpdateGroupID",
		}) {
			Init(app, bot, true)
		}
	})

}
