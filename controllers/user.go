package controllers

import (
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/helpers"
	"github.com/amiraliio/tgbp-user/models"
	emoji "github.com/tmdvs/Go-Emoji-Utils"
	tb "gopkg.in/tucnak/telebot.v2"
)

//TODO add gmail.com yahoo to ban email address

func (service *BotService) RegisterUserWithemail(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) bool {
	userModel := new(tb.User)
	userModel.ID = userID
	options := new(tb.SendOptions)
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboardRemove = true
	options.ReplyMarkup = replyModel
	options.ParseMode = tb.ModeMarkdown
	if lastState != nil && lastState.State == config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL") {
		uniqueID := lastState.Data
		companyModel, channelModel, state := checkAndVerifyCompany(db, app, bot, userModel, uniqueID, userID)
		if state {
			return true
		}
		if !strings.Contains(text, "@") {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.PLEASE_ENTER_VALID_EMAIL"), options)
			return true
		}
		matched, _ := regexp.MatchString(`^([a-zA-Z0-9_\.\-])+\@(([a-zA-Z0-9\-])+\.)+([a-zA-Z0-9]{2,4})+$`, text)
		if !matched {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.PLEASE_ENTER_VALID_EMAIL"), options)
			return true
		}
		emails := []string{"hotmail.com", "outlook.com", "zoho.com", "icloud.com", "mail.com", "aol.com", "yandex.com"}
		emailSuffix := strings.Split(text, "@")
		if helpers.SortAndSearchInStrings(emails, emailSuffix[1]) {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.NOT_ALLOWED_PUBLIC_EMAIL"), options)
			return true
		}
		userDataModel := new(models.User)
		if err := db.QueryRow("SELECT us.id from `users` as us inner join users_channels as uc on us.id=uc.userID and uc.status='ACTIVE' where us.email=?", text).Scan(&userDataModel.ID); err == nil {
			bot.Send(userModel, config.LangConfig.GetString("MESSAGES.USER_WITH_THIS_EMAIL_EXIST"))
			return true
		}
		service.checkTheCompanyEmailSuffixExist(app, bot, text, "@"+emailSuffix[1], db, userModel, channelModel, companyModel, userID)
		return true
	}
	uniqueID := strings.TrimPrefix(text, config.LangConfig.GetString("COMMANDS.JOIN_TO_GROUP"))
	companyModel, channelModel, state := checkAndVerifyCompany(db, app, bot, userModel, uniqueID, userID)
	if state {
		return true
	}
	service.JoinFromGroup(db, app, bot, m, channelModel.ChannelID)
	SaveUserLastState(db, app, bot, uniqueID, userID, config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"))
	bot.Send(userModel, `Please enter your email I'd to register for
`+channelModel.ChannelType+` *`+channelModel.ChannelName+`*
company *`+companyModel.CompanyName+`*`, options)
	return true
}

func checkAndVerifyCompany(db *sql.DB, app *config.App, bot *tb.Bot, userModel *tb.User, uniqueID string, userID int) (*models.Company, *models.Channel, bool) {
	channelModel := new(models.Channel)
	companyModel := new(models.Company)
	channelSetting := new(models.ChannelSetting)
	options := new(tb.SendOptions)
	replyModel := new(tb.ReplyMarkup)
	replyModel.ReplyKeyboardRemove = true
	options.ReplyMarkup = replyModel
	if err := db.QueryRow("SELECT ch.id,ch.channelName,ch.channelID,ch.channelURL,ch.channelType,co.companyName,co.id,IFNULL(cs.joinVerify,0) as joinVerify from `channels` as ch inner join companies_channels as cc on ch.id=cc.channelID inner join companies as co on cc.companyID=co.id inner join channels_settings as cs on cs.channelID=ch.id where ch.uniqueID=?", uniqueID).Scan(&channelModel.ID, &channelModel.ChannelName, &channelModel.ChannelID, &channelModel.ChannelURL, &channelModel.ChannelType, &companyModel.CompanyName, &companyModel.ID, &channelSetting.JoinVerify); err != nil {
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.THERE_IS_NO_COMPANY_TO_JOIN"), options)
		return nil, nil, true
	}
	newOptions := new(tb.SendOptions)
	newReplyModel := new(tb.ReplyMarkup)
	gotoChannel := tb.InlineButton{
		Text: "Click Here to Start Commination",
		URL:  channelModel.ChannelURL,
	}
	replyKeys := [][]tb.InlineButton{
		[]tb.InlineButton{gotoChannel},
	}
	newReplyModel.InlineKeyboard = replyKeys
	newReplyModel.ReplyKeyboardRemove = true
	newOptions.ReplyMarkup = newReplyModel
	newOptions.ParseMode = tb.ModeMarkdown
	userDataModel := new(models.User)
	if !channelSetting.JoinVerify {
		bot.Send(userModel, "You trying to join to the "+channelModel.ChannelType+" *"+channelModel.ChannelName+"* belongs to the company *"+companyModel.CompanyName+"*, to start commination, go to "+channelModel.ChannelType+" via the blow button", newOptions)
		return nil, nil, true
	}
	if err := db.QueryRow("SELECT us.id from `users` as us inner join users_channels as uc on us.id=uc.userID and uc.channelID=? and uc.status='ACTIVE' where us.userID=?", channelModel.ID, userID).Scan(&userDataModel.ID); err == nil {
		bot.Send(userModel, "You have been registered in the "+channelModel.ChannelType+" *"+channelModel.ChannelName+"* belongs to the company *"+companyModel.CompanyName+"*, to start commination, go to "+channelModel.ChannelType+" via the blow button", newOptions)
		return nil, nil, true
	}
	if err := db.QueryRow("SELECT us.id from users as us inner join users_channels as uc on us.id=uc.userID and uc.status='ACTIVE' inner join companies_channels as cc on cc.channelID=uc.channelID and cc.companyID=? where us.userID=?", companyModel.ID, userID).Scan(&userDataModel.ID); err == nil {
		bot.Send(userModel, "You have been registered in the one of channels or groups blongs to company *"+companyModel.CompanyName+"*, and the verification is not required, to start commination, go to "+channelModel.ChannelType+" via the blow button", newOptions)
		return nil, nil, true
	}
	return companyModel, channelModel, false
}

func (service *BotService) checkTheCompanyEmailSuffixExist(app *config.App, bot *tb.Bot, email, emailSuffix string, db *sql.DB, userModel *tb.User, channelModel *models.Channel, companyModel *models.Company, userID int) {
	optionsModel := new(tb.SendOptions)
	replyNewModel := new(tb.ReplyMarkup)
	replyNewModel.ReplyKeyboardRemove = true
	optionsModel.ReplyMarkup = replyNewModel
	companiesSuffixesModel := new(models.CompanyEmailSuffixes)
	if err := db.QueryRow("SELECT id from `companies_email_suffixes` where suffix=? and companyID=? limit 1", emailSuffix, companyModel.ID).Scan(&companiesSuffixesModel.ID); err != nil {
		SaveUserLastState(db, app, bot, emailSuffix, userModel.ID, config.LangConfig.GetString("STATE.CONFIRM_REGISTER_COMPANY"))
		bot.Send(userModel, config.LangConfig.GetString("MESSAGES.COMPANY_WITH_THE_EMAIL_NOT_EXIST"), optionsModel)
		return
	}
	rand.Seed(time.Now().UnixNano())
	randomeNumber := rand.Intn(100000)
	hashedRandomNumber, err := helpers.HashPassword(strconv.Itoa(randomeNumber))
	if err != nil {
		log.Println(err)
		return
	}
	insertCompanyRequest, err := db.Query("INSERT INTO `users_activation_key` (`userID`,`activeKey`,`createdAt`) VALUES(?,?,?)", userID, hashedRandomNumber, app.CurrentTime)
	if err != nil {
		log.Println(err)
		return
	}
	defer insertCompanyRequest.Close()
	go helpers.SendEmail(strconv.Itoa(randomeNumber), strings.TrimSpace(email))
	SaveUserLastState(db, app, bot, strconv.FormatInt(channelModel.ID, 10)+"_"+email, userModel.ID, config.LangConfig.GetString("STATE.EMAIL_FOR_USER_REGISTRATION"))
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.ENTER_CODE_FROM_EMAIL"), optionsModel)
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
	SaveUserLastState(db, app, bot, "register_"+strconv.FormatInt(userModelData.ID, 10)+"_"+strconv.FormatInt(channelID, 10), m.Sender.ID, config.LangConfig.GetString("STATE.ADD_PSEUDONYM"))
	options := new(tb.SendOptions)
	options.ParseMode = tb.ModeMarkdown
	bot.Send(userModel, config.LangConfig.GetString("MESSAGES.YOU_ARE_MEMBER_OF_CHANNEL")+channelModel.ChannelType+" "+channelModel.ChannelName+" \n"+config.LangConfig.GetString("MESSAGES.USERNAME_MESSAGE"), options)
	return true
}

func (service *BotService) SetUserUserName(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	channelData := strings.Split(lastState.Data, "_")
	channelID, err := strconv.ParseInt(channelData[2], 10, 0)
	if err != nil {
		log.Println(err)
		return true
	}
	channelModel := new(models.Channel)
	if err := db.QueryRow("SELECT channelID,channelURL,manualChannelName,channelName,channelType FROM `channels` where id=?", channelID).Scan(&channelModel.ChannelID, &channelModel.ChannelURL, &channelModel.ManualChannelName, &channelModel.ChannelName, &channelModel.ChannelType); err != nil {
		log.Println(err)
		return true
	}
	emojiLetter := emoji.FindAll(m.Text)
	if emojiLetter == nil || len(emojiLetter) > 1 || (emojiLetter[0].Locations != nil && len(emojiLetter[0].Locations) == 1 && len(emojiLetter[0].Locations[0]) == 2 && emojiLetter[0].Locations[0][0] != 0 && emojiLetter[0].Locations[0][1] != 1) {
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USER_NAME_IS_WRONG"))
		return true
	}
	if m.Text == "" {
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USER_NAME_IS_WRONG"))
		return true
	}
	anotherUsernameLetters := emoji.RemoveAll(m.Text)
	hasRightFormat, err := regexp.MatchString(`([A-Za-z]+[0-9]|[0-9]+[A-Za-z])[A-Za-z0-9]*`, anotherUsernameLetters)
	if err != nil {
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USER_NAME_IS_WRONG"))
		return true
	}
	if len(anotherUsernameLetters) != 3 || !hasRightFormat {
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USER_NAME_IS_WRONG"))
		return true
	}
	usersUserNameModel := new(models.UserUserName)
	if err := db.QueryRow("SELECT id FROM `users_usernames` where channelID=? and username=?", channelID, m.Text).Scan(&usersUserNameModel.ID); err == nil {
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USERNAME_IS_TAKEN"))
		return true
	}
	userID := service.GetUserByTelegramID(db, app, m.Sender.ID)
	insertCompanyRequest, err := db.Query("INSERT INTO `users_usernames` (`userID`,`channelID`,`username`,`createdAt`) VALUES(?,?,?,?)", userID.ID, channelID, m.Text, app.CurrentTime)
	if err != nil {
		log.Println(err)
		return true
	}
	defer insertCompanyRequest.Close()
	switch channelData[0] {
	case "register":
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
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.JOIN_COMPLETED"))
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.YOU_HAVE_AN_PSEUDONYM")+channelModel.ChannelType+" "+channelModel.ChannelName, options)
		return true
	case "compose":
		newComposeMessage := new(tb.Message)
		newComposeMessage.Sender = m.Sender
		newComposeMessage.Text = config.LangConfig.GetString("COMMANDS.START_COMPOSE_IN_GROUP") + channelModel.ChannelID
		return generalEventsHandler(app, bot, newComposeMessage, &Event{
			UserState:  config.LangConfig.GetString("STATE.NEW_MESSAGE_TO_GROUP"),
			Command:    config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_COMPOSE_IN_GROUP"),
			Controller: "NewMessageGroupHandler",
		})
	case "reply":
		newReplyMessage := new(tb.Message)
		newReplyMessage.Sender = m.Sender
		newReplyMessage.Text = config.LangConfig.GetString("COMMANDS.START_REPLY") + channelModel.ChannelID + "_" + strconv.Itoa(m.Sender.ID) + "_" + channelData[3]
		return generalEventsHandler(app, bot, newReplyMessage, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE"),
			Command:    config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY"),
			Controller: "SendReply",
		})
	case "dm":
		newDMMessage := new(tb.Message)
		newDMMessage.Sender = m.Sender
		newDMMessage.Text = config.LangConfig.GetString("COMMANDS.START_REPLY_DM") + channelModel.ChannelID + "_" + channelData[1] + "_" + channelData[3]
		return generalEventsHandler(app, bot, newDMMessage, &Event{
			UserState:  config.LangConfig.GetString("STATE.REPLY_BY_DM"),
			Command:    config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_",
			Command1:   config.LangConfig.GetString("COMMANDS.START_REPLY_DM"),
			Controller: "SanedDM",
		})
	default:
		return true
	}
}

func (service *BotService) checkUserHaveUserName(db *sql.DB, app *config.App, channelID, userID int64) (*models.UserUserName, error) {
	usersUserNameModel := new(models.UserUserName)
	if err := db.QueryRow("SELECT id,username FROM `users_usernames` where userID=? and channelID=?", userID, channelID).Scan(&usersUserNameModel.ID, &usersUserNameModel.Username); err != nil {
		return nil, err
	}
	return usersUserNameModel, nil
}

func (service *BotService) getUserUsername(db *sql.DB, app *config.App, channelID, userID int64) bool {
	usersUserNameModel := new(models.UserUserName)
	if err := db.QueryRow("SELECT id FROM `users_usernames` where userID=? and channelID=?", userID, channelID).Scan(&usersUserNameModel.ID); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (service *BotService) GetUserByTelegramID(db *sql.DB, app *config.App, userID int) *models.User {
	userModel := new(models.User)
	if err := db.QueryRow("SELECT `id`,`userID` from `users` where `userID`=? ", userID).Scan(&userModel.ID, &userModel.UserID); err != nil {
		log.Println(err)
		userModel.Status = "INACTIVE"
		return userModel
	}
	return userModel
}

func GetUserLastState(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, user int) *models.UserLastState {
	userLastState := new(models.UserLastState)
	userModel := new(models.User)
	if err := db.QueryRow("SELECT ul.data,ul.state,ul.userID,us.id from `users_last_state` as ul inner join users as us on us.userID=ul.userID where ul.userId=? order by ul.id DESC limit 1", user).Scan(&userLastState.Data, &userLastState.State, &userLastState.UserID, &userModel.ID); err != nil {
		log.Println(err)
		userLastState.Status = "INACTIVE"
		return userLastState
	}
	userLastState.User = userModel
	return userLastState
}

func (service *BotService) CheckUserRegisteredOrNot(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int, section string) bool {
	channel := service.GetUserCurrentActiveChannel(db, app, bot, m, userID)
	if channel.ManualChannelName != "" {
		userModel := new(models.User)
		err := db.QueryRow("SELECT us.`id` from `users` as us inner join `users_channels` as uc on us.id=uc.userID and uc.channelID=? where us.userID=? and uc.status = 'ACTIVE' limit 1", channel.ID, userID).Scan(&userModel.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err := db.QueryRow("SELECT us.`id` from `users` as us inner join `users_channels` as uc on us.id=uc.userID inner join companies_channels as cc on cc.channelID=uc.channelID and cc.companyID=? where us.userID=? and uc.status = 'ACTIVE' limit 1", channel.Company.ID, userID).Scan(&userModel.ID)
			if errors.Is(err, sql.ErrNoRows) {
				switch {
				case section == config.LangConfig.GetString("GENERAL.REPLY_VERIFY") && channel.Setting.ReplyVerify:
					goto VERIFY
				case section == config.LangConfig.GetString("GENERAL.NEW_MESSAGE_VERIFY") && channel.Setting.NewMessageVerify:
					goto VERIFY
				case section == config.LangConfig.GetString("GENERAL.DIRECT_VERIFY") && channel.Setting.DirectVerify:
					goto VERIFY
				default:
					return false
				}
			VERIFY:
				if m.Sender != nil {
					SaveUserLastState(db, app, bot, channel.UniqueID, userID, config.LangConfig.GetString("STATE.REGISTER_USER_WITH_EMAIL"))
				}
				options := new(tb.SendOptions)
				replyModel := new(tb.ReplyMarkup)
				replyModel.ReplyKeyboardRemove = true
				options.ReplyMarkup = replyModel
				bot.Send(m.Sender, "You should first verify your account, Please enter your email in the "+channel.ChannelType+" "+channel.ChannelName+" belongs to the company "+channel.Company.CompanyName+" for verification", options)
				return true
			}
		}
	}
	return false
}
