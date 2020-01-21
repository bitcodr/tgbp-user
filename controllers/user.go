package controllers

import (
	"database/sql"
	"errors"
	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	"github.com/amiraliio/tgbp-user/models"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

//TODO add gmail.com to ban email address

func (service *BotService) RegisterUserWithemail(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) bool {
	userModel := new(tb.User)
	userModel.ID = userID
	options := new(tb.SendOptions)
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboardRemove = true
	options.ReplyMarkup = replyModel
	if lastState.State == config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL") {
		uniqueID := lastState.Data
		companyModel, channelModel, state := checkAndVerifyCompany(db, app, bot, userModel, uniqueID, userID)
		if state {
			return true
		}
		if !strings.Contains(text, "@") {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.PLEASE_ENTER_VALID_EMAIL"), options)
			return true
		}
		emails := []string{"yahoo.com", "hotmail.com", "outlook.com", "zoho.com", "icloud.com", "mail.com", "aol.com", "yandex.com"}
		emailSuffix := strings.Split(text, "@")
		if helpers.SortAndSearchInStrings(emails, emailSuffix[1]) {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.NOT_ALLOWED_PUBLIC_EMAIL"), options)
			return true
		}
		service.checkTheCompanyEmailSuffixExist(app, bot, text, "@"+emailSuffix[1], db, userModel, channelModel, companyModel)
		return true
	}
	uniqueID := strings.TrimPrefix(text, config.LangConfig.GetString("COMMANDS.JOIN_TO_GROUP"))
	companyModel, channelModel, state := checkAndVerifyCompany(db, app, bot, userModel, uniqueID, userID)
	if state {
		return true
	}
	SaveUserLastState(db, app, bot, uniqueID, userID, config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"))
	bot.Send(userModel, "Please enter your email in the "+channelModel.ChannelType+" "+channelModel.ChannelName+" belongs to the company "+companyModel.CompanyName+" for verification", options)
	return true
}

func checkAndVerifyCompany(db *sql.DB, app *config.App, bot *tb.Bot, userModel *tb.User, uniqueID string, userID int) (*models.Company, *models.Channel, bool) {
	channelModel := new(models.Channel)
	companyModel := new(models.Company)
	options := new(tb.SendOptions)
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboardRemove = true
	options.ReplyMarkup = replyModel
	if err := db.QueryRow("SELECT ch.id,ch.channelName,ch.channelURL,ch.channelType,co.companyName,co.id from `channels` as ch inner join companies_channels as cc on ch.id=cc.channelID inner join companies as co on cc.companyID=co.id where ch.uniqueID=?", uniqueID).Scan(&channelModel.ID, &channelModel.ChannelName, &channelModel.ChannelURL, &channelModel.ChannelType, &companyModel.CompanyName, &companyModel.ID); err != nil {
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.THERE_IS_NO_COMPANY_TO_JOIN"), options)
		return nil, nil, true
	}
	userDataModel := new(models.User)
	if err := db.QueryRow("SELECT id from `users` as us inner join users_channels as uc on us.id=uc.userID and uc.channelID=? and uc.status='ACTIVE' where us.userID=?", channelModel.ID, userID).Scan(&userDataModel.ID); err == nil {
		bot.Send(userModel, "You have been registered in the "+channelModel.ChannelType+" "+channelModel.ChannelName+" belongs to the company "+companyModel.CompanyName+", to start commination, go to "+channelModel.ChannelType+" via "+channelModel.ChannelURL, options)
		return nil, nil, true
	}
	return companyModel, channelModel, false
}

func (service *BotService) checkTheCompanyEmailSuffixExist(app *config.App, bot *tb.Bot, email, emailSuffix string, db *sql.DB, userModel *tb.User, channelModel *models.Channel, companyModel *models.Company) {
	options := new(tb.SendOptions)
	yesBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.YES_TEXT"),
	}
	noBTN := tb.ReplyButton{
		Text: config.LangConfig.GetString("GENERAL.NO_TEXT"),
	}
	replyKeys := [][]tb.ReplyButton{
		[]tb.ReplyButton{yesBTN, noBTN},
	}
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboard = replyKeys
	options.ReplyMarkup = replyModel
	companiesSuffixesModel := new(models.CompanyEmailSuffixes)
	if err := db.QueryRow("SELECT id from `companies_email_suffixes` where suffix=? and companyID=? limit 1", emailSuffix, companyModel.ID).Scan(&companiesSuffixesModel.ID); err != nil {
		SaveUserLastState(db, app, bot, emailSuffix, userModel.ID, config.LangConfig.GetString("STATE.CONFIRM_REGISTER_COMPANY"))
		optionsModel := new(tb.SendOptions)
		replyNewModel := new(tb.ReplyMarkup)
		replyNewModel.ReplyKeyboardRemove = true
		optionsModel.ReplyMarkup = replyNewModel
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.COMPANY_WITH_THE_EMAIL_NOT_EXIST"), optionsModel)
		return
	}
	SaveUserLastState(db, app, bot, strconv.FormatInt(channelModel.ID, 10)+"_"+email, userModel.ID, config.LangConfig.GetString("STATE.REGISTER_USER_FOR_COMPANY"))
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.CONFIRM_REGISTER_TO_CHANNEL")+channelModel.ChannelType+" "+channelModel.ChannelName+config.LangConfig.GetString("MESSAGES.BLONGS_TO_COMPANY")+companyModel.CompanyName+"?", options)
}

func (service *BotService) ConfirmRegisterCompanyRequest(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	optionsModel := new(tb.SendOptions)
	replyNewModel := new(tb.ReplyMarkup)
	replyNewModel.ReplyKeyboardRemove = true
	optionsModel.ReplyMarkup = replyNewModel
	switch m.Text {
	case config.LangConfig.GetString("GENERAL.YES_TEXT"):
		insertCompanyRequest, err := db.Query("INSERT INTO `companies_join_request` (`userID`,`emailSuffix`,`createdAt`) VALUES('" + strconv.FormatInt(lastState.UserID, 10) + "','" + lastState.Data + "','" + app.CurrentTime + "')")
		if err != nil {
			log.Println(err)
			return true
		}
		defer insertCompanyRequest.Close()
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_ADDED"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.SEND_REQUEST_TO_ADMIN"), optionsModel)
	case config.LangConfig.GetString("GENERAL.NO_TEXT"):
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_DISMISSED"))
		bot.Send(userModel, "Your registration proccess cancelled", optionsModel)
	}
	return true
}

func (service *BotService) ConfirmRegisterUserForTheCompany(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	optionsModel := new(tb.SendOptions)
	replyNewModel := new(tb.ReplyMarkup)
	replyNewModel.ReplyKeyboardRemove = true
	optionsModel.ReplyMarkup = replyNewModel
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
		userDBModel := new(models.User)
		channelModel := new(models.Channel)
		if err := db.QueryRow("SELECT us.id,ch.channelType FROM `users` as us inner join `users_channels` as uc on us.id=uc.userID and uc.channelID=? and uc.status='ACTIVE' inner join `channels` as ch on uc.channelID=ch.id where us.userID=?", channelData[0], m.Sender.ID).Scan(&userDBModel.ID, &channelModel.ChannelType); err == nil {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.REGISTERED_IN_CHANNEL")+channelModel.ChannelType, optionsModel)
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
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.ENTER_CODE_FROM_EMAIL"), optionsModel)
	case config.LangConfig.GetString("GENERAL.NO_TEXT"):
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.CANCEL_USER_REGISTRATION"))
		bot.Send(userModel, "Your registration proccess cancelled", optionsModel)
	}
	return true
}

func (service *BotService) RegisterUserWithEmailAndCode(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	userModel := new(tb.User)
	userModel.ID = m.Sender.ID
	userActiveKeyModel := new(models.UsersActivationKey)
	optionsModel := new(tb.SendOptions)
	replyNewModel := new(tb.ReplyMarkup)
	replyNewModel.ReplyKeyboardRemove = true
	optionsModel.ReplyMarkup = replyNewModel
	if err := db.QueryRow("SELECT `activeKey`,`createdAt` FROM `users_activation_key` where userID=? order by `id` DESC limit 1", m.Sender.ID).Scan(&userActiveKeyModel.ActiveKey, &userActiveKeyModel.CreatedAt); err != nil {
		log.Println(err)
		return true
	}
	//TODO check token expire time
	if !helpers.CheckPasswordHash(m.Text, userActiveKeyModel.ActiveKey) {
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.KEY_IS_INVALID"), optionsModel)
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
	channelModel := new(models.Channel)
	channelID, err := strconv.ParseInt(channelData[0], 10, 0)
	if err != nil {
		log.Println(err)
		return true
	}
	if err := db.QueryRow("SELECT channelID,channelURL,manualChannelName,channelName,channelType FROM `channels` where id=?", channelID).Scan(&channelModel.ChannelID, &channelModel.ChannelURL, &channelModel.ManualChannelName, &channelModel.ChannelName, &channelModel.ChannelType); err != nil {
		log.Println(err)
		return true
	}
	service.JoinFromGroup(db, app, bot, m, channelModel.ChannelID)
	_, err = db.Query("update `users` set `email`=? where `userID`=?", channelData[1], m.Sender.ID)
	if err != nil {
		log.Println(err)
		return true
	}
	userModelData := new(models.User)
	if err := db.QueryRow("SELECT id FROM `users` where userID=?", m.Sender.ID).Scan(&userModelData.ID); err != nil {
		log.Println(err)
		return true
	}
	_, err = db.Query("update `users_channels` set `status`=? where `userID`=? and `channelID`=?", "ACTIVE", userModelData.ID, channelID)
	if err != nil {
		log.Println(err)
		return true
	}
	SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_REQUEST_ADDED"))
	options := new(tb.SendOptions)
	startBTN := tb.InlineButton{
		Text: config.LangConfig.GetString("MESSAGES.CLICK_HERE_TO_START_COMMUNICATION"),
		URL:  channelModel.ChannelURL,
	}
	replyKeys := [][]tb.InlineButton{
		[]tb.InlineButton{startBTN},
	}
	replyModel := new(tb.ReplyMarkup)
	replyModel.InlineKeyboard = replyKeys
	options.ReplyMarkup = replyModel
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.YOU_ARE_MEMBER_OF_CHANNEL")+channelModel.ChannelType+" "+channelModel.ChannelName, options)
	return true
}

func (service *BotService) GetUserByTelegramID(db *sql.DB, app *config.App, userID int) *models.User {
	userModel := new(models.User)
	if err := db.QueryRow("SELECT `id`,`userID`,`customID` from `users` where `userID`=? ", userID).Scan(&userModel.ID, &userModel.UserID, &userModel.CustomID); err != nil {
		log.Println(err)
		userModel.Status = "INACTIVE"
		return userModel
	}
	return userModel
}

func GetUserLastState(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, user int) *models.UserLastState {
	userLastState := new(models.UserLastState)
	if err := db.QueryRow("SELECT `data`,`state`,`userID` from `users_last_state` where `userId`=? order by `id` DESC limit 1", user).Scan(&userLastState.Data, &userLastState.State, &userLastState.UserID); err != nil {
		log.Println(err)
		userLastState.Status = "INACTIVE"
		return userLastState
	}
	return userLastState
}

//TODO check unique email

func (service *BotService) CheckUserRegisteredOrNot(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) bool {
	channel := service.GetUserCurrentActiveChannel(db, app, bot, m, userID)
	if channel.ManualChannelName != "" {
		userModel := new(models.User)
		err := db.QueryRow("SELECT us.`id` from `users` as us inner join `users_channels` as uc on us.id=uc.userID and uc.channelID=? where us.userID=? and uc.status = 'ACTIVE' limit 1", channel.ID, userID).Scan(&userModel.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err := db.QueryRow("SELECT us.`id` from `users` as us inner join `users_channels` as uc on us.id=uc.userID inner join companies_channels as cc on cc.channelID=uc.channelID and cc.companyID=? where us.userID=? and uc.status = 'ACTIVE' limit 1", channel.Company.ID, userID).Scan(&userModel.ID)
			if errors.Is(err, sql.ErrNoRows) {
				if m.Sender != nil {
					SaveUserLastState(db, app, bot, m.Text, m.Sender.ID, config.LangConfig.GetString("MESSAGES.USER_NOT_REGISTERED"))
				}
				bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.YOU_DO_NOT_HAVE_PREMISSION_TO_SEND_A_MESSAGE"))
				return true
			}
		}
	}
	return false
	//TODO also check it according to event channel is required a action for instance reply is mandatory or not
}
