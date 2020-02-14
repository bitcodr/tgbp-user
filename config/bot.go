package config

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func (app *App) Bot() *tb.Bot {
	if AppConfig.GetBool("APP.WEBHOOK_POLLER") {
		return app.webhookPoller()
	}
	return app.longPoller()
}

func (app *App) webhookPoller() *tb.Bot {
	webhookEndpoint := new(tb.WebhookEndpoint)
	webhookEndpoint.PublicURL = AppConfig.GetString("APP.BOT_WEBHOOK_PUBLIC_URL")
	poller := new(tb.Webhook)
	poller.Listen = AppConfig.GetString("APP.BOT_WEBHOOK_PORT")
	poller.Endpoint = webhookEndpoint
	bot, err := tb.NewBot(tb.Settings{
		Token:  app.BotToken,
		Poller: poller,
	})
	if err != nil {
		log.Fatalln(err)
	}
	return bot
}

func (app *App) longPoller() *tb.Bot {
	poller := &tb.LongPoller{Timeout: 15 * time.Second}
	bot, err := tb.NewBot(tb.Settings{
		Token:  app.BotToken,
		Poller: poller,
	})
	if err != nil {
		log.Fatalln(err)
	}
	return bot
}
