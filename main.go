package main

import (
	"log"
	"os"

	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/controllers"
)

func main() {

	//get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	//initial app config
	app := new(config.App)
	app.ProjectDir = currentDir

	//initial environment variables
	app.Environment()

	//set app configs
	app = app.SetAppConfig()

	//init bot
	bot := app.Bot()

	//handle bot events
	controllers.Init(app, bot, nil)

	//start the bot
	bot.Start()
}
