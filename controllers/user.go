package controllers

import (
	"database/sql"
	"github.com/amiraliio/tgbp/config"
	"github.com/amiraliio/tgbp/helpers"
	"github.com/amiraliio/tgbp/models"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func (service *BotService) RegisterUserWithemail(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) bool {
	userModel := new(tb.User)
	userModel.ID = userID
	if lastState.State == config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL") {
		if !strings.Contains(text, "@") {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.PLEASE_ENTER_VALID_EMAIL"))
			return true
		}
		emails := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "zoho.com", "icloud.com", "mail.com", "aol.com", "yandex.com"}
		emailSuffix := strings.Split(text, "@")
		if helpers.SortAndSearchInStrings(emails, emailSuffix[1]) {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.NOT_ALLOWED_PUBLIC_EMAIL"))
			return true
		}
		service.checkTheCompanyEmailSuffixExist(app, bot, text, "@"+emailSuffix[1], db, userModel)
		return true
	}
	SaveUserLastState(db, app, bot, text, userID, config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"))
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.ENTER_COMPANY_EMAIL"), HomeKeyOption(db, app))
	return true
}

func (service *BotService) checkTheCompanyEmailSuffixExist(app *config.App, bot *tb.Bot, email, emailSuffix string, db *sql.DB, userModel *tb.User) {
	tempData, err := db.Prepare("SELECT co.companyName,ch.id,ch.channelName from `channels_email_suffixes` as cs inner join `channels` as ch on cs.channelID=ch.id inner join `companies_channels` as cc on ch.Id=cc.channelID inner join `companies` as co on cc.companyID=co.id where cs.suffix=? limit 1")
	if err != nil {
		log.Println(err)
		return
	}
	defer tempData.Close()
	companyModel := new(models.Company)
	channelModel := new(models.Channel)
	options := new(tb.SendOptions)
	yesBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.YES_TEXT"),
	}
	noBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.NO_TEXT"),
	}
	homeBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.HOME"),
	}
	replyKeys := [][]tb.ReplyButton{
		[]tb.ReplyButton{yesBTN, noBTN},
		[]tb.ReplyButton{homeBTN},
	}
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboard = replyKeys
	options.ReplyMarkup = replyModel
	if err = tempData.QueryRow(emailSuffix).Scan(&companyModel.CompanyName, &channelModel.ID, &channelModel.ChannelName); err != nil {
		SaveUserLastState(db, app, bot, emailSuffix, userModel.ID, config.LangConfig.GetString("STATE.CONFIRM_REGISTER_COMPANY"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.COMPANY_WITH_THE_EMAIL_NOT_EXIST"), options)
		return
	}
	SaveUserLastState(db, app, bot, strconv.FormatInt(channelModel.ID, 10)+"_"+email, userModel.ID, config.LangConfig.GetString("STATE.REGISTER_USER_FOR_COMPANY"))
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.CONFIRM_REGISTER_TO_CHANNEL")+channelModel.ChannelName+config.LangConfig.GetString("MESSAGES.BLONGS_TO_COMPANY")+companyModel.CompanyName+"?", options)
}

func (service *BotService) ConfirmRegisterCompanyRequest(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	switch m.Text {
	case config.LangConfig.GetString("GENERAL.YES_TEXT"):
		insertCompanyRequest, err := db.Query("INSERT INTO `companies_join_request` (`userID`,`emailSuffix`,`createdAt`) VALUES('" + strconv.FormatInt(lastState.UserID, 10) + "','" + lastState.Data + "','" + app.CurrentTime + "')")
		if err != nil {
			log.Println(err)
			return true
		}
		defer insertCompanyRequest.Close()
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_ADDED"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.SEND_REQUEST_TO_ADMIN"), HomeKeyOption(db, app))
	case config.LangConfig.GetString("GENERAL.NO_TEXT"):
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_DISMISSED"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.CONTINUE_IN_BOT"), HomeKeyOption(db, app))
	}
	return true
}

func HomeKeyOption(db *sql.DB, app *config.App) *tb.SendOptions {
	options := new(tb.SendOptions)
	homeBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.HOME"),
	}
	replyKeys := [][]tb.ReplyButton{
		[]tb.ReplyButton{homeBTN},
	}
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboard = replyKeys
	options.ReplyMarkup = replyModel
	return options
}

func (service *BotService) ConfirmRegisterUserForTheCompany(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	switch m.Text {
	case config.LangConfig.GetString("GENERAL.YES_TEXT"):
		if !strings.Contains(lastState.Data, "_") {
			log.Println(config.LangConfig.GetString("MESSAGES.STRING_MUST_BE_2_PART"))
			return true
		}
		channelData := strings.Split(lastState.Data, "_")
		if len(channelData) != 2 {
			log.Println(config.LangConfig.GetString("MESSAGES.LENGTH_OF_CHANNEL_DATA_MUST_BE_2"))
			return true
		}
		userExistOrNot, err := db.Prepare("SELECT us.id FROM `users` as us inner join `users_channels` as uc on us.id=uc.userID and uc.channelID=? and uc.status='ACTIVE' where us.userID=?")
		if err != nil {
			log.Println(err)
			return true
		}
		defer userExistOrNot.Close()
		userDBModel := new(models.User)
		if err := userExistOrNot.QueryRow(channelData[0], m.Sender.ID).Scan(&userDBModel.ID); err == nil {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.REGISTERED_IN_CHANNEL"))
			return true
		}
		rand.Seed(time.Now().UnixNano())
		randomeNumber := rand.Intn(100000)
		hashedRandomNumber, err := helpers.HashPassword(strconv.Itoa(randomeNumber))
		if err != nil {
			log.Println(err)
			return true
		}
		insertCompanyRequest, err := db.Query("INSERT INTO `users_activation_key` (`userID`,`activeKey`,`createdAt`) VALUES('" + strconv.FormatInt(lastState.UserID, 10) + "','" + hashedRandomNumber + "','" + app.CurrentTime + "')")
		if err != nil {
			log.Println(err)
			return true
		}
		defer insertCompanyRequest.Close()
		go helpers.SendEmail(strconv.Itoa(randomeNumber), channelData[1])
		SaveUserLastState(db, app, bot, lastState.Data, m.Sender.ID, config.LangConfig.GetString("STATE.EMAIL_FOR_USER_REGISTRATION"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.ENTER_CODE_FROM_EMAIL"), HomeKeyOption(db, app))
	case config.LangConfig.GetString("GENERAL.NO_TEXT"):
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.CANCEL_USER_REGISTRATION"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.CONTINUE_TO_HOME_BTN"), HomeKeyOption(db, app))
	}
	return true
}

func (service *BotService) RegisterUserWithEmailAndCode(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	userActiveKey, err := db.Prepare("SELECT `activeKey`,`createdAt` FROM `users_activation_key` where userID=? order by `id` DESC limit 1")
	if err != nil {
		log.Println(err)
		return true
	}
	defer userActiveKey.Close()
	userActiveKeyModel := new(models.UsersActivationKey)
	if err := userActiveKey.QueryRow(m.Sender.ID).Scan(&userActiveKeyModel.ActiveKey, &userActiveKeyModel.CreatedAt); err != nil {
		log.Println(err)
		return true
	}
	//TODO check token expire time
	if !helpers.CheckPasswordHash(m.Text, userActiveKeyModel.ActiveKey) {
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.KEY_IS_INVALID"), HomeKeyOption(db, app))
		return true
	}
	if !strings.Contains(lastState.Data, "_") {
		log.Println(config.LangConfig.GetString("MESSAGES.STRING_MUST_BE_2_PART"))
		return true
	}
	channelData := strings.Split(lastState.Data, "_")
	if len(channelData) != 2 {
		log.Println(config.LangConfig.GetString("MESSAGES.LENGTH_OF_CHANNEL_DATA_MUST_BE_2"))
		return true
	}
	resultsStatement, err := db.Prepare("SELECT channelID,channelURL,manualChannelName,channelName FROM `channels` where id=?")
	if err != nil {
		log.Println(err)
		return true
	}
	defer resultsStatement.Close()
	channelModel := new(models.Channel)
	channelID, err := strconv.ParseInt(channelData[0], 10, 0)
	if err != nil {
		log.Println(err)
		return true
	}
	if err := resultsStatement.QueryRow(channelID).Scan(&channelModel.ChannelID, &channelModel.ChannelURL, &channelModel.ManualChannelName, &channelModel.ChannelName); err != nil {
		log.Println(err)
		return true
	}
	service.JoinFromGroup(db, app, bot, m, channelModel.ChannelID)
	_, err = db.Query("update `users` set `email`=? where `userID`=?", channelData[1], m.Sender.ID)
	if err != nil {
		log.Println(err)
		return true
	}
	//TODO also update users channel state
	SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_ADDED"))
	options := new(tb.SendOptions)
	homeBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.HOME"),
	}
	replyBTN := [][]tb.ReplyButton{
		[]tb.ReplyButton{homeBTN},
	}
	startBTN := tb.InlineButton{
		Text: config.LangConfig.GetString("MESSAGES.CLICK_HERE_TO_START_COMMUNICATION"),
		URL:  channelModel.ChannelURL,
	}
	replyKeys := [][]tb.InlineButton{
		[]tb.InlineButton{startBTN},
	}
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboard = replyBTN
	replyModel.InlineKeyboard = replyKeys
	options.ReplyMarkup = replyModel
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.YOU_ARE_MEMBER_OF_CHANNEL")+channelModel.ChannelName, options)
	return true
}

func (service *BotService) GetUserByTelegramID(db *sql.DB, app *config.App, userID int) *models.User {
	userLastStateQueryStatement, err := db.Prepare("SELECT `id`,`userID` from `users` where `userID`=? ")
	if err != nil {
		log.Println(err)
	}
	defer userLastStateQueryStatement.Close()
	userLastStateQuery, err := userLastStateQueryStatement.Query(userID)
	if err != nil {
		log.Println(err)
	}
	userModel := new(models.User)
	if userLastStateQuery.Next() {
		if err := userLastStateQuery.Scan(&userModel.ID, &userModel.UserID); err != nil {
			log.Println(err)
		}
		return userModel
	}
	return userModel
}

func GetUserLastState(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, user int) *models.UserLastState {
	userLastStateQueryStatement, err := db.Prepare("SELECT `data`,`state`,`userID` from `users_last_state` where `userId`=? order by `id` DESC limit 1")
	if err != nil {
		log.Println(err)
	}
	defer userLastStateQueryStatement.Close()
	userLastStateQuery, err := userLastStateQueryStatement.Query(user)
	if err != nil {
		log.Println(err)
	}
	userLastState := new(models.UserLastState)
	if userLastStateQuery.Next() {
		if err := userLastStateQuery.Scan(&userLastState.Data, &userLastState.State, &userLastState.UserID); err != nil {
			log.Println(err)
		}
		return userLastState
	}
	return userLastState
}

func (service *BotService) CheckUserRegisteredOrNot(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) {
	channel := service.GetUserCurrentActiveChannel(db, app, bot, m)
	if channel.Setting != nil {

	}
	//TODO check the channel is registered or not
	//TODO if the channel is one of the company that user is registered verification is not necessary
	//TODO also check it according to event channel is required a action for instance reply is mandatory or not
	//TODO check if user is registered to company or not
}
