package helpers

import (
	"crypto/tls"
	"log"

	"github.com/amiraliio/tgbp-user/config"
	ses "github.com/srajelli/ses-go"
	"gopkg.in/gomail.v2"
)

func SendEmail(body, to string) {
	switch config.AppConfig.GetString("EMAIL.DRIVER") {
	case "SIMPLE":
		simple(body, to)
	case "AWS":
		aws(body, to)
	}
}

func aws(body, to string) {
	ses.SetConfiguration(config.AppConfig.GetString("EMAIL.USERNAME"), config.AppConfig.GetString("EMAIL.PASSWORD"), config.AppConfig.GetString("EMAIL.REGION"))

	emailData := ses.Email{
		To:      to,
		From:    config.AppConfig.GetString("EMAIL.FROM"),
		Text:    config.LangConfig.GetString("MESSAGES.YOU_ACTIVE_KEY") + " " + body + " " + config.LangConfig.GetString("MESSAGES.ACTIVE_EXPIRE_PEROID"),
		Subject: config.AppConfig.GetString("APP.BOT_USERNAME") + config.LangConfig.GetString("MESSAGES.ACTIVE_KEY"),
	}
	ses.SendEmail(emailData)
}

func simple(body, to string) {
	from := config.AppConfig.GetString("EMAIL.FROM")
	pass := config.AppConfig.GetString("EMAIL.PASSWORD")
	userName := config.AppConfig.GetString("EMAIL.USERNAME")
	serverAddress := config.AppConfig.GetString("EMAIL.PROVIDER")
	serverPort := config.AppConfig.GetInt("EMAIL.PORT")
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", config.AppConfig.GetString("APP.BOT_USERNAME")+config.LangConfig.GetString("MESSAGES.ACTIVE_KEY"))
	m.SetBody("text/html", config.LangConfig.GetString("MESSAGES.YOU_ACTIVE_KEY")+"<b>"+body+"</b>"+config.LangConfig.GetString("MESSAGES.ACTIVE_EXPIRE_PEROID"))
	d := gomail.NewDialer(serverAddress, serverPort, userName, pass)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		log.Println(err.Error())
		return
	}
}
