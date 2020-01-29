//Package controllers ...
package controllers

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/amiraliio/tgbp-user/config"
	"github.com/amiraliio/tgbp-user/lang"
	"github.com/amiraliio/tgbp-user/models"
	tb "gopkg.in/tucnak/telebot.v2"
)

//TODO save user hashed id in db

//RegisterChannel
func (service *BotService) RegisterChannel(app *config.App, bot *tb.Bot, m *tb.Message, request *Event) bool {
	if strings.TrimSpace(m.Text) == request.Command {
		db := app.DB()
		defer db.Close()
		if m.Sender != nil {
			SaveUserLastState(db, app, bot, m.Text, m.Sender.ID, request.UserState)
		}
		//channel private url
		inviteLink, err := bot.GetInviteLink(m.Chat)
		if err != nil {
			log.Println(err)
			return true
		}
		channelURL := inviteLink
		channelID := strconv.FormatInt(m.Chat.ID, 10)
		channelModel := new(models.Channel)
		err = db.QueryRow("SELECT id FROM `channels` where channelID=?", channelID).Scan(&channelModel.ID)
		if errors.Is(err, sql.ErrNoRows) {
			//start transaction
			transaction, err := db.Begin()
			if err != nil {
				log.Println(err)
				return true
			}
			uniqueID := uuid.New().String()
			//insert channel
			channelInserted, err := transaction.Exec("INSERT INTO `channels` (`channelType`,`channelURL`,`channelID`,`channelName`,`uniqueID`,`createdAt`,`updatedAt`) VALUES(?,?,?,?,?,?,?)", "channel", channelURL, channelID, m.Chat.Title, uniqueID, app.CurrentTime, app.CurrentTime)
			if err != nil {
				transaction.Rollback()
				log.Println(err)
				return true
			}
			insertedChannelID, err := channelInserted.LastInsertId()
			if err == nil {
				//company name
				companyFlag := channelID
				//check if company is not exist
				companyModel := new(models.Company)
				err := db.QueryRow("SELECT id FROM `companies` where `companyName`=?", companyFlag).Scan(&companyModel.ID)
				if errors.Is(err, sql.ErrNoRows) {
					//insert company
					companyInserted, err := transaction.Exec("INSERT INTO `companies` (`companyName`,`createdAt`,`updatedAt`) VALUES(?,?,?)", companyFlag, app.CurrentTime, app.CurrentTime)
					if err != nil {
						transaction.Rollback()
						log.Println(err)
						return true
					}
					insertedCompanyID, err := companyInserted.LastInsertId()
					if err == nil {
						companyModelID := strconv.FormatInt(insertedCompanyID, 10)
						channelModelID := strconv.FormatInt(insertedChannelID, 10)
						//insert company channel pivot
						_, err := transaction.Exec("INSERT INTO `companies_channels` (`companyID`,`channelID`,`createdAt`) VALUES(?,?,?)", companyModelID, channelModelID, app.CurrentTime)
						if err != nil {
							transaction.Rollback()
							log.Println(err)
							return true
						}
					}
				} else {
					companyModelID := strconv.FormatInt(companyModel.ID, 10)
					channelModelID := strconv.FormatInt(insertedChannelID, 10)
					//insert company channel pivot
					_, err := transaction.Exec("INSERT INTO `companies_channels` (`companyID`,`channelID`,`createdAt`) VALUES(?,?,?)", companyModelID, channelModelID, app.CurrentTime)
					if err != nil {
						transaction.Rollback()
						log.Println(err)
						return true
					}
				}
				transaction.Commit()
				successMessage, _ := bot.Send(m.Chat, config.LangConfig.GetString("MESSAGES.CHANNEL_REGISTERED_SUCCESSFULLY"))
				time.Sleep(2 * time.Second)
				if err := bot.Delete(successMessage); err != nil {
					log.Println(err)
					return true
				}
				sendOptionModel := new(tb.SendOptions)
				sendOptionModel.ParseMode = tb.ModeHTML
				_, err = bot.Send(m.Chat, config.LangConfig.GetString("MESSAGES.CHANNEL_UNIQUE_ID_MESSAGE")+" <code> "+uniqueID+" </code>", sendOptionModel)
				if err != nil {
					log.Println(err)
					return true
				}
				time.Sleep(2 * time.Second)
				compose := tb.InlineButton{
					Unique: config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + channelID,
					Text:   config.LangConfig.GetString("MESSAGES.COMPOSE_MESSAGE"),
					URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + channelID,
				}
				groupKeys := [][]tb.InlineButton{
					[]tb.InlineButton{compose},
				}
				newReplyModel := new(tb.ReplyMarkup)
				newReplyModel.InlineKeyboard = groupKeys
				newSendOption := new(tb.SendOptions)
				newSendOption.ReplyMarkup = newReplyModel
				newSendOption.ParseMode = tb.ModeMarkdown
				pinMessage, err := bot.Send(m.Chat, lang.StartGroup, newSendOption)
				if err != nil {
					log.Println(err)
					return true
				}
				if err := bot.Pin(pinMessage); err != nil {
					log.Println(err)
					return true
				}
				if err := bot.Delete(m); err != nil {
					log.Println(err)
					return true
				}
			}
		}
		return true
	}
	return false
}

func (service *BotService) SendReply(app *config.App, bot *tb.Bot, m *tb.Message, request *Event) bool {
	if strings.Contains(m.Text, request.Command) {
		db := app.DB()
		defer db.Close()
		service.CheckIfBotIsAdmin(app, bot, m, db, request)
		lastState := GetUserLastState(db, app, bot, m, m.Sender.ID)
		if service.CheckUserRegisteredOrNot(db, app, bot, m, request, lastState, m.Text, m.Sender.ID, config.LangConfig.GetString("GENERAL.REPLY_VERIFY")) {
			return true
		}
		if m.Sender != nil {
			SaveUserLastState(db, app, bot, m.Text, m.Sender.ID, request.UserState)
		}
		ids := strings.TrimPrefix(m.Text, request.Command1)
		data := strings.Split(ids, "_")
		channelID := strings.TrimSpace(data[0])
		messageID := strings.TrimSpace(data[2])
		service.JoinFromGroup(db, app, bot, m, channelID)
		channelModel := new(models.Channel)
		messageModel := new(models.Message)
		if err := db.QueryRow("SELECT ch.id,ch.channelName,me.message, me.messageType FROM `channels` as ch inner join messages as me on ch.id=me.channelID and me.botMessageID=? where ch.channelID=?", messageID, channelID).Scan(&channelModel.ID, &channelModel.ChannelName, &messageModel.Message, &messageModel.MessageType); err != nil {
			log.Println(err)
			return true
		}
		_, err := service.checkUserHaveUserName(db, app, channelModel.ID, lastState.User.ID)
		if err != nil {
			SaveUserLastState(db, app, bot, "reply_"+strconv.FormatInt(lastState.User.ID, 10)+"_"+strconv.FormatInt(channelModel.ID, 10)+"_"+messageID, m.Sender.ID, config.LangConfig.GetString("STATE.ADD_PSEUDONYM"))
			bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USERNAME_MESSAGE"))
			return true
		}
		var maxLenOfString int
		if messageModel.MessageType == "TEXT" {
			if len(messageModel.Message) < 60 {
				maxLenOfString = len(messageModel.Message)
			} else {
				maxLenOfString = 60
			}
			_, err = bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.PLEASE_REPLY")+"'"+messageModel.Message[0:maxLenOfString]+"...' on "+channelModel.ChannelName)
			if err != nil {
				log.Println(err)
				return true
			}
		}
		file, err := bot.FileByID(messageModel.Message)
		if err != nil {
			log.Println(err)
			return true
		}
		photoModel := new(tb.Photo)
		switch messageModel.MessageType {
		case "PHOTO":
			photoModel.File = file
			photoModel.Caption = config.LangConfig.GetString("MESSAGES.PLEASE_REPLY_MEDIA") + "on " + channelModel.ChannelName
		}
		_, err = bot.Send(m.Sender, photoModel)
		if err != nil {
			log.Println(err)
			return true
		}
		return true
	}
	return false
}

func (service *BotService) SanedDM(app *config.App, bot *tb.Bot, m *tb.Message, request *Event) bool {
	if strings.Contains(m.Text, request.Command) {
		db := app.DB()
		defer db.Close()
		service.CheckIfBotIsAdmin(app, bot, m, db, request)
		ids := strings.TrimPrefix(m.Text, request.Command1)
		data := strings.Split(ids, "_")
		directSenderID, err := strconv.Atoi(data[1])
		if err != nil {
			log.Println(err)
			return true
		}
		if m.Sender.ID == directSenderID {
			bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.BAN_DIRECT"))
			if m.Sender != nil {
				SaveUserLastState(db, app, bot, config.LangConfig.GetString("STATE.NOT_DM_ACCESS"), m.Sender.ID, config.LangConfig.GetString("STATE.NOT_DM_ACCESS"))
			}
			return true
		}
		if m.Sender != nil {
			SaveUserLastState(db, app, bot, m.Text, m.Sender.ID, request.UserState)
		}
		channelID := strings.TrimSpace(data[0])
		service.JoinFromGroup(db, app, bot, m, channelID)
		lastState := GetUserLastState(db, app, bot, m, m.Sender.ID)
		if service.CheckUserRegisteredOrNot(db, app, bot, m, request, lastState, m.Text, m.Sender.ID, config.LangConfig.GetString("GENERAL.DIRECT_VERIFY")) {
			return true
		}
		options := new(tb.SendOptions)
		options.ParseMode = tb.ModeHTML
		channel := service.GetChannelByTelegramID(db, app, channelID)
		user := service.GetUserByTelegramID(db, app, m.Sender.ID)
		_, err = service.checkUserHaveUserName(db, app, channel.ID, user.ID)
		if err != nil {
			SaveUserLastState(db, app, bot, "dm_"+data[1]+"_"+strconv.FormatInt(channel.ID, 10)+"_"+data[2], m.Sender.ID, config.LangConfig.GetString("STATE.ADD_PSEUDONYM"))
			bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USERNAME_MESSAGE"))
			return true
		}
		userDtaModel := service.GetUserByTelegramID(db, app, directSenderID)
		usernamemodel, _ := service.checkUserHaveUserName(db, app, channel.ID, userDtaModel.ID)
		_, err = bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.PLEASE_SEND_YOUR_DIRECT")+"<b>"+usernamemodel.Username+"</b> "+config.LangConfig.GetString("GENERAL.FROM")+": <b>"+channel.ChannelName+"</b>", options)
		if err != nil {
			log.Println(err)
			return true
		}
		return true
	}
	return false
}

func (service *BotService) SanedAnswerDM(app *config.App, bot *tb.Bot, m *tb.Callback, request *Event) bool {
	if strings.Contains(m.Data, request.Command) {
		db := app.DB()
		defer db.Close()
		lastState := GetUserLastState(db, app, bot, m.Message, m.Sender.ID)
		if service.CheckUserRegisteredOrNot(db, app, bot, m.Message, request, lastState, m.Data, m.Sender.ID, config.LangConfig.GetString("GENERAL.DIRECT_VERIFY")) {
			return true
		}
		text := strings.TrimPrefix(m.Data, request.Command1)
		ids := strings.ReplaceAll(text, request.Command, "")
		data := strings.Split(ids, "_")
		directSenderID, err := strconv.Atoi(data[1])
		if err != nil {
			log.Println(err)
			return true
		}
		if m.Sender.ID == directSenderID {
			bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.BAN_DIRECT"))
			if m.Sender != nil {
				SaveUserLastState(db, app, bot, config.LangConfig.GetString("STATE.NOT_DM_ACCESS"), m.Sender.ID, config.LangConfig.GetString("STATE.NOT_DM_ACCESS"))
			}
			return true
		}
		if m.Sender != nil {
			SaveUserLastState(db, app, bot, m.Data, m.Sender.ID, request.UserState)
		}
		options := new(tb.SendOptions)
		options.ParseMode = tb.ModeHTML
		channelID := strings.TrimSpace(data[0])
		channel := service.GetChannelByTelegramID(db, app, channelID)
		user := service.GetUserByTelegramID(db, app, m.Sender.ID)
		_, err = service.checkUserHaveUserName(db, app, channel.ID, user.ID)
		if err != nil {
			SaveUserLastState(db, app, bot, "dm_"+data[1]+"_"+strconv.FormatInt(channel.ID, 10)+"_"+data[2], m.Sender.ID, config.LangConfig.GetString("STATE.ADD_PSEUDONYM"))
			bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.USERNAME_MESSAGE"))
			return true
		}
		userDtaModel := service.GetUserByTelegramID(db, app, directSenderID)
		usernamemodel, _ := service.checkUserHaveUserName(db, app, channel.ID, userDtaModel.ID)
		_, err = bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.PLEASE_SEND_YOUR_DIRECT")+"<b>"+usernamemodel.Username+"</b> "+config.LangConfig.GetString("GENERAL.FROM")+": <b>"+channel.ChannelName+"</b>", options)
		if err != nil {
			log.Println(err)
			return true
		}
		return true
	}
	return false
}

func (service *BotService) SaveAndSendMessage(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	activeChannel := service.GetUserCurrentActiveChannel(db, app, bot, m, m.Sender.ID)
	if activeChannel != nil {
		senderID := strconv.Itoa(m.Sender.ID)
		botMessageID := strconv.Itoa(m.ID)
		usernameModel, _ := service.checkUserHaveUserName(db, app, activeChannel.ID, lastState.User.ID)
		newReply := tb.InlineButton{
			Unique: config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_" + activeChannel.ChannelID + "_" + senderID + "_" + botMessageID,
			Text:   config.LangConfig.GetString("MESSAGES.REPLY"),
			URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_" + activeChannel.ChannelID + "_" + senderID + "_" + botMessageID,
		}
		newM := tb.InlineButton{
			Unique: config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + activeChannel.ChannelID,
			Text:   config.LangConfig.GetString("MESSAGES.NEW"),
			URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + activeChannel.ChannelID,
		}
		newDM := tb.InlineButton{
			Unique: config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_" + activeChannel.ChannelID + "_" + senderID + "_" + botMessageID,
			Text:   config.LangConfig.GetString("MESSAGES.DIRECT") + " [User " + usernameModel.Username + "]",
			URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_" + activeChannel.ChannelID + "_" + senderID + "_" + botMessageID,
		}
		inlineKeys := [][]tb.InlineButton{
			[]tb.InlineButton{newReply},
			[]tb.InlineButton{newDM},
			[]tb.InlineButton{newM},
		}
		activeChannelID, err := strconv.Atoi(activeChannel.ChannelID)
		if err == nil {
			user := new(tb.User)
			user.ID = activeChannelID
			options := new(tb.SendOptions)
			replyModel := new(tb.ReplyMarkup)
			replyModel.InlineKeyboard = inlineKeys
			options.ReplyMarkup = replyModel
			options.ParseMode = tb.ModeHTML
			var message *tb.Message
			var err error
			var messageType, saveMessage string
			switch {
			case m.Photo != nil:
				messageType = "PHOTO"
				saveMessage = m.Photo.MediaFile().FileID
				m.Photo.Caption = "[User " + usernameModel.Username + "]"
				message, err = bot.Send(user, m.Photo, options)
			case m.Video != nil:
				saveMessage = m.Video.MediaFile().FileID
				messageType = "VIDEO"
				m.Video.Caption = "[User " + usernameModel.Username + "]"
				message, err = bot.Send(user, m.Video, options)
			case m.Audio != nil:
				saveMessage = m.Audio.MediaFile().FileID
				messageType = "AUDIO"
				m.Audio.Caption = "[User " + usernameModel.Username + "]"
				message, err = bot.Send(user, m.Audio, options)
			default:
				messageType = "TEXT"
				saveMessage = m.Text
				message, err = bot.Send(user, "[User "+usernameModel.Username+"] "+m.Text, options)
			}
			if err == nil {
				channelMessageID := strconv.Itoa(message.ID)
				channelID := strconv.FormatInt(activeChannel.ID, 10)
				insertedMessage, err := db.Query("INSERT INTO `messages` (`messageType`,`message`,`userID`,`channelID`,`channelMessageID`,`botMessageID`,`type`,`createdAt`) VALUES(?,?,?,?,?,?,?,?)", messageType, saveMessage, senderID, channelID, channelMessageID, botMessageID, "NEW", app.CurrentTime)
				if err != nil {
					log.Println(err)
					return true
				}
				defer insertedMessage.Close()
				options := new(tb.SendOptions)
				markup := new(tb.ReplyMarkup)
				anotherMessage := tb.InlineButton{
					Unique: config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + activeChannel.ChannelID,
					Text:   config.LangConfig.GetString("MESSAGES.ANOTHER_NEW"),
				}
				var inlineKeys [][]tb.InlineButton
				if activeChannel.ChannelURL != "" {
					redirectBTN := tb.InlineButton{
						Text: config.LangConfig.GetString("MESSAGES.BACK_TO") + strings.Title(activeChannel.ChannelType),
						URL:  activeChannel.ChannelURL,
					}
					inlineKeys = [][]tb.InlineButton{
						[]tb.InlineButton{anotherMessage},
						[]tb.InlineButton{redirectBTN},
					}
				} else {
					inlineKeys = [][]tb.InlineButton{
						[]tb.InlineButton{anotherMessage},
					}
				}
				markup.InlineKeyboard = inlineKeys
				options.ReplyMarkup = markup
				options.ParseMode = tb.ModeMarkdown
				bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.MESSAGE_HAS_BEEN_SENT")+activeChannel.ChannelType+", *"+activeChannel.ChannelName+"*.", options)
				SaveUserLastState(db, app, bot, "", m.Sender.ID, "message_sent")
			}
		}
	}
	return true
}

func (service *BotService) SendAndSaveReplyMessage(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	if lastState.Data != "" {
		ids := strings.TrimPrefix(lastState.Data, request.Command1)
		if ids != "" {
			data := strings.Split(ids, "_")
			if len(data) == 3 {
				activeChannel := service.GetUserCurrentActiveChannel(db, app, bot, m, m.Sender.ID)
				channelID := strings.TrimSpace(data[0])
				userID := strings.TrimSpace(data[1])
				botMessageID := strings.TrimSpace(data[2])
				senderID := strconv.Itoa(m.Sender.ID)
				newBotMessageID := strconv.Itoa(m.ID)
				messageModel := new(models.Message)
				usernameModel, _ := service.checkUserHaveUserName(db, app, activeChannel.ID, lastState.User.ID)
				if err := db.QueryRow("SELECT me.id,me.channelMessageID from `messages` as me inner join `channels` as ch on me.channelID=ch.id and ch.channelID=? where me.`botMessageID`=? and me.`userID`=?", channelID, botMessageID, userID).Scan(&messageModel.ID, &messageModel.ChannelMessageID); err == nil {
					channelIntValue, err := strconv.Atoi(channelID)
					if err == nil {
						newReply := tb.InlineButton{
							Unique: config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
							Text:   config.LangConfig.GetString("MESSAGES.REPLY"),
							URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.REPLY_TO_MESSAGE") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
						}
						newM := tb.InlineButton{
							Unique: config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + channelID,
							Text:   config.LangConfig.GetString("MESSAGES.NEW"),
							URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.COMPOSE_MESSAGE") + "_" + channelID,
						}
						newDM := tb.InlineButton{
							Unique: config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
							Text:   config.LangConfig.GetString("MESSAGES.DIRECT") + " [User " + usernameModel.Username + "]",
							URL:    app.TgDomain + app.BotUsername + "?start=" + config.LangConfig.GetString("STATE.REPLY_BY_DM") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
						}
						inlineKeys := [][]tb.InlineButton{
							[]tb.InlineButton{newReply},
							[]tb.InlineButton{newDM},
							[]tb.InlineButton{newM},
						}
						ChannelMessageDataID, err := strconv.Atoi(messageModel.ChannelMessageID)
						if err == nil {
							sendMessageModel := new(tb.Message)
							sendMessageModel.ID = ChannelMessageDataID
							newReplyModel := new(tb.ReplyMarkup)
							newReplyModel.InlineKeyboard = inlineKeys
							newSendOption := new(tb.SendOptions)
							newSendOption.ReplyTo = sendMessageModel
							newSendOption.ReplyMarkup = newReplyModel
							newSendOption.ParseMode = tb.ModeHTML
							user := new(tb.User)
							user.ID = channelIntValue
							var sendMessage *tb.Message
							var err error
							var messageType, saveMessage string
							switch {
							case m.Photo != nil:
								messageType = "PHOTO"
								saveMessage = m.Photo.MediaFile().FileID
								m.Photo.Caption = "[User " + usernameModel.Username + "]"
								sendMessage, err = bot.Send(user, m.Photo, newSendOption)
							case m.Video != nil:
								saveMessage = m.Video.MediaFile().FileID
								messageType = "VIDEO"
								m.Video.Caption = "[User " + usernameModel.Username + "]"
								sendMessage, err = bot.Send(user, m.Video, newSendOption)
							case m.Audio != nil:
								saveMessage = m.Audio.MediaFile().FileID
								messageType = "AUDIO"
								m.Audio.Caption = "[User " + usernameModel.Username + "]"
								sendMessage, err = bot.Send(user, m.Audio, newSendOption)
							default:
								messageType = "TEXT"
								saveMessage = m.Text
								sendMessage, err = bot.Send(user, "[User "+usernameModel.Username+"] "+m.Text, newSendOption)
							}
							if err == nil {
								newChannelMessageID := strconv.Itoa(sendMessage.ID)
								parentID := strconv.FormatInt(messageModel.ID, 10)
								newChannelModel := new(models.Channel)
								if err := db.QueryRow("SELECT id,channelName,channelType from `channels` where channelID=?", channelID).Scan(&newChannelModel.ID, &newChannelModel.ChannelName, &newChannelModel.ChannelType); err == nil {
									newChannelModelID := strconv.FormatInt(newChannelModel.ID, 10)
									insertedMessage, err := db.Query("INSERT INTO `messages` (`messageType`,`message`,`userID`,`channelID`,`channelMessageID`,`botMessageID`,`parentID`,`type`,`createdAt`) VALUES(?,?,?,?,?,?,?,?,?)", messageType, saveMessage, senderID, newChannelModelID, newChannelMessageID, newBotMessageID, parentID, "REPLY", app.CurrentTime)
									if err != nil {
										log.Println(err)
										return true
									}
									defer insertedMessage.Close()
									options := new(tb.SendOptions)
									if activeChannel.ChannelURL != "" {
										markup := new(tb.ReplyMarkup)
										redirectBTN := tb.InlineButton{
											Text: config.LangConfig.GetString("MESSAGES.BACK_TO") + strings.Title(activeChannel.ChannelType),
											URL:  activeChannel.ChannelURL,
										}
										inlineKeys := [][]tb.InlineButton{
											[]tb.InlineButton{redirectBTN},
										}
										markup.InlineKeyboard = inlineKeys
										options.ReplyMarkup = markup
									}
									options.ParseMode = tb.ModeMarkdown
									bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.REPLY_MESSAGE_HAS_BEEN_SENT")+newChannelModel.ChannelType+", *"+newChannelModel.ChannelName+"*.", options)
									SaveUserLastState(db, app, bot, "", m.Sender.ID, "reply_message_sent")
								}
							}
						}
					}
				}
			}
		}
	}
	return true
}

func (service *BotService) SendAndSaveDirectMessage(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	if lastState.Data != "" {
		ids := strings.TrimPrefix(lastState.Data, request.Command1)
		if ids != "" {
			data := strings.Split(ids, "_")
			if len(data) == 3 {
				channelID := strings.TrimSpace(data[0])
				userID := strings.TrimSpace(data[1])
				botMessageID := strings.TrimSpace(data[2])
				senderID := strconv.Itoa(m.Sender.ID)
				newBotMessageID := strconv.Itoa(m.ID)
				userIDInInt, err := strconv.Atoi(userID)
				if err == nil {
					messageModel := new(models.Message)
					channelModel := new(models.Channel)
					if err := db.QueryRow("SELECT me.id,me.channelMessageID,ch.id,ch.channelName,ch.channelURL,ch.channelType from `messages` as me inner join `channels` as ch on me.channelID=ch.id and ch.channelID=? where me.`botMessageID`=? and me.`userID`=?", channelID, botMessageID, userID).Scan(&messageModel.ID, &messageModel.ChannelMessageID, &channelModel.ID, &channelModel.ChannelName, &channelModel.ChannelURL, &channelModel.ChannelType); err == nil {
						_, err := strconv.Atoi(messageModel.ChannelMessageID)
						if err == nil {
							userIntID, err := strconv.Atoi(userID)
							if err != nil {
								log.Println(err)
								return true
							}
							userDataModel := service.GetUserByTelegramID(db, app, userIntID)
							newUsernameModel, _ := service.checkUserHaveUserName(db, app, channelModel.ID, userDataModel.ID)
							options := new(tb.SendOptions)
							markup := new(tb.ReplyMarkup)
							SendAnotherDM := tb.InlineButton{
								Unique: config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_" + channelID + "_" + userID + "_" + newBotMessageID,
								Text:   config.LangConfig.GetString("MESSAGES.ANOTHER_DIRECT_REPLY") + " [User " + newUsernameModel.Username + "]",
							}
							dmHistory := tb.InlineButton{
								Text: config.LangConfig.GetString("MESSAGES.DM_HISTORY") + " [User " + newUsernameModel.Username + "]",
								URL:  app.APIURL + "/user/" + senderID + "/receiver/" + userID + "/channel/" + strconv.FormatInt(channelModel.ID, 10) + "/direct-messages",
							}
							var AnotherDMKeys [][]tb.InlineButton
							if channelModel.ChannelURL != "" {
								redirectBTN := tb.InlineButton{
									Text: config.LangConfig.GetString("MESSAGES.BACK_TO") + strings.Title(channelModel.ChannelType),
									URL:  channelModel.ChannelURL,
								}
								AnotherDMKeys = [][]tb.InlineButton{
									[]tb.InlineButton{SendAnotherDM},
									[]tb.InlineButton{dmHistory},
									[]tb.InlineButton{redirectBTN},
								}

							} else {
								AnotherDMKeys = [][]tb.InlineButton{
									[]tb.InlineButton{SendAnotherDM},
									[]tb.InlineButton{dmHistory},
								}
							}
							markup.InlineKeyboard = AnotherDMKeys
							options.ReplyMarkup = markup
							options.ParseMode = tb.ModeHTML
							bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.DIRECT_HAS_BEEN_SENT")+"<b>"+newUsernameModel.Username+"</b>", options)
							newReplyModel := new(tb.ReplyMarkup)
							senderUserDataModel := service.GetUserByTelegramID(db, app, m.Sender.ID)
							usernameModel, _ := service.checkUserHaveUserName(db, app, channelModel.ID, senderUserDataModel.ID)
							newReply := tb.InlineButton{
								Unique: config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
								Text:   config.LangConfig.GetString("MESSAGES.DIRECT_REPLY") + " [User " + usernameModel.Username + "]",
							}
							inlineKeys := [][]tb.InlineButton{
								[]tb.InlineButton{newReply},
							}
							newReplyModel.InlineKeyboard = inlineKeys
							newSendOption := new(tb.SendOptions)
							newSendOption.ReplyMarkup = newReplyModel
							newSendOption.ParseMode = tb.ModeHTML
							user := new(tb.User)
							user.ID = userIDInInt
							senderDataModel := service.GetUserByTelegramID(db, app, m.Sender.ID)
							usernameDataModel, _ := service.checkUserHaveUserName(db, app, channelModel.ID, senderDataModel.ID)
							sendMessage, err := bot.Send(user, config.LangConfig.GetString("GENERAL.FROM")+": "+channelModel.ChannelName+"\nBy: [User "+usernameDataModel.Username+"]\n------------------------------\n"+config.LangConfig.GetString("GENERAL.MESSAGE")+": "+m.Text, newSendOption)
							if err == nil {
								newChannelMessageID := strconv.Itoa(sendMessage.ID)
								parentID := strconv.FormatInt(messageModel.ID, 10)
								newChannelModel := new(models.Channel)
								if err := db.QueryRow("SELECT id from `channels` where channelID=?", channelID).Scan(&newChannelModel.ID); err == nil {
									newChannelModelID := strconv.FormatInt(newChannelModel.ID, 10)
									insertedMessage, err := db.Query("INSERT INTO `messages` (`message`,`userID`,`channelID`,`channelMessageID`,`botMessageID`,`parentID`,`receiver`,`type`,`createdAt`) VALUES(?,?,?,?,?,?,?,?,?)", m.Text, senderID, newChannelModelID, newChannelMessageID, newBotMessageID, parentID, userIDInInt, "DM", app.CurrentTime)
									if err != nil {
										log.Println(err)
										return true
									}
									defer insertedMessage.Close()
									SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.DIRECT_MESSAGE_SENT"))
								}
							}
						}
					}
				}
			}
		}
	}
	return true
}

func (service *BotService) SendAnswerAndSaveDirectMessage(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState) bool {
	if lastState.Data != "" {
		ids := strings.ReplaceAll(lastState.Data, config.LangConfig.GetString("STATE.ANSWER_TO_DM")+"_", "")
		if ids != "" {
			data := strings.Split(ids, "_")
			if len(data) == 3 {
				channelID := strings.TrimSpace(data[0])
				userID := strings.TrimSpace(data[1])
				senderID := strconv.Itoa(m.Sender.ID)
				newBotMessageID := strconv.Itoa(m.ID)
				userIDInInt, err := strconv.Atoi(userID)
				if err == nil {
					channelModel := new(models.Channel)
					if err := db.QueryRow("SELECT id,channelURL,channelType from `channels` where channelID=?", channelID).Scan(&channelModel.ID, &channelModel.ChannelURL, &channelModel.ChannelType); err == nil {
						options := new(tb.SendOptions)
						markup := new(tb.ReplyMarkup)
						userDataModel := service.GetUserByTelegramID(db, app, userIDInInt)
						usernameModel, _ := service.checkUserHaveUserName(db, app, channelModel.ID, userDataModel.ID)
						SendAnotherDM := tb.InlineButton{
							Unique: config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_" + channelID + "_" + userID + "_" + newBotMessageID,
							Text:   config.LangConfig.GetString("MESSAGES.ANOTHER_DIRECT_REPLY") + " [User " + usernameModel.Username + "]",
						}
						dmHistory := tb.InlineButton{
							Text: config.LangConfig.GetString("MESSAGES.DM_HISTORY") + " [User " + usernameModel.Username + "]",
							URL:  app.APIURL + "/user/" + senderID + "/receiver/" + userID + "/channel/" + strconv.FormatInt(channelModel.ID, 10) + "/direct-messages",
						}
						var AnotherDMKeys [][]tb.InlineButton
						if channelModel.ChannelURL != "" {
							redirectBTN := tb.InlineButton{
								Text: config.LangConfig.GetString("MESSAGES.BACK_TO") + strings.Title(channelModel.ChannelType),
								URL:  channelModel.ChannelURL,
							}
							AnotherDMKeys = [][]tb.InlineButton{
								[]tb.InlineButton{SendAnotherDM},
								[]tb.InlineButton{dmHistory},
								[]tb.InlineButton{redirectBTN},
							}

						} else {
							AnotherDMKeys = [][]tb.InlineButton{
								[]tb.InlineButton{SendAnotherDM},
								[]tb.InlineButton{dmHistory},
							}
						}
						markup.InlineKeyboard = AnotherDMKeys
						options.ReplyMarkup = markup
						options.ParseMode = tb.ModeHTML
						bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.DIRECT_HAS_BEEN_SENT")+" <b>"+usernameModel.Username+"</b>", options)
						newChannelModel := new(models.Channel)
						if err := db.QueryRow("SELECT id,channelName,channelType from `channels` where `channelID`=?", channelID).Scan(&newChannelModel.ID, &newChannelModel.ChannelName, &newChannelModel.ChannelType); err == nil {
							senderUserDataModel := service.GetUserByTelegramID(db, app, m.Sender.ID)
							newUsernameModel, _ := service.checkUserHaveUserName(db, app, newChannelModel.ID, senderUserDataModel.ID)
							newReply := tb.InlineButton{
								Unique: config.LangConfig.GetString("STATE.ANSWER_TO_DM") + "_" + channelID + "_" + senderID + "_" + newBotMessageID,
								Text:   config.LangConfig.GetString("MESSAGES.DIRECT_REPLY") + " [User " + newUsernameModel.Username + "]",
							}
							inlineKeys := [][]tb.InlineButton{
								[]tb.InlineButton{newReply},
							}
							newReplyModel := new(tb.ReplyMarkup)
							newReplyModel.InlineKeyboard = inlineKeys
							newSendOption := new(tb.SendOptions)
							newSendOption.ReplyMarkup = newReplyModel
							newSendOption.ParseMode = tb.ModeHTML
							user := new(tb.User)
							user.ID = userIDInInt
							senderDataModel := service.GetUserByTelegramID(db, app, m.Sender.ID)
							newUsernameDataModel, _ := service.checkUserHaveUserName(db, app, newChannelModel.ID, senderDataModel.ID)
							sendMessage, err := bot.Send(user, config.LangConfig.GetString("GENERAL.FROM")+": "+newChannelModel.ChannelName+"\nBy: [User "+newUsernameDataModel.Username+"]\n------------------------------\n"+config.LangConfig.GetString("GENERAL.MESSAGE")+": "+m.Text, newSendOption)
							if err == nil {
								newChannelMessageID := strconv.Itoa(sendMessage.ID)
								newChannelModelID := strconv.FormatInt(newChannelModel.ID, 10)
								insertedMessage, err := db.Query("INSERT INTO `messages` (`message`,`userID`,`channelID`,`channelMessageID`,`botMessageID`,`receiver`,`type`,`createdAt`) VALUES(?,?,?,?,?,?,?,?)", m.Text, senderID, newChannelModelID, newChannelMessageID, newBotMessageID, userIDInInt, "DM", app.CurrentTime)
								if err != nil {
									log.Println(err)
									return true
								}
								defer insertedMessage.Close()
								SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.DIRECT_MESSAGE_SENT"))
							}
						}
					}
				}
			}
		}
	}
	return true
}

func (service *BotService) JoinToOtherCompanyChannels(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, request *Event, lastState *models.UserLastState, text string, userID int) bool {
	companyName := strings.TrimPrefix(text, request.Command)
	rows, err := db.Query("SELECT ch.channelType,ch.channelName,ch.publicURL from `channels` as ch inner join `companies_channels` as uc on ch.id=uc.channelID inner join companies as co on co.id=uc.companyID and co.companyName=?", companyName)
	if err != nil {
		SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.No_CHANNEL_FOR_THE_COMPANY"))
		bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.THERE_IS_NO_CHANNEL_FOR_COMPANY"))
		return true
	}
	defer rows.Close()
	var inlineButtonsEven []tb.InlineButton
	var inlineButtonsOdd []tb.InlineButton
	var index *int
	for rows.Next() {
		channelModel := new(models.Channel)
		if err := rows.Scan(&channelModel.ChannelType, &channelModel.ChannelName, &channelModel.PublicURL); err != nil {
			log.Println(err)
			return true
		}
		inlineButton := tb.InlineButton{
			Text: channelModel.ChannelType + " " + channelModel.ChannelName,
			URL:  channelModel.PublicURL,
		}
		if *index%2 == 0 {
			inlineButtonsEven = append(inlineButtonsEven, inlineButton)
		} else {
			inlineButtonsEven = append(inlineButtonsOdd, inlineButton)
		}
		*index++
	}
	SaveUserLastState(db, app, bot, "", m.Sender.ID, config.LangConfig.GetString("STATE.COMPANY_CHANNEL_SENT"))
	inlineKeyboards := [][]tb.InlineButton{
		inlineButtonsEven,
		inlineButtonsOdd,
	}
	options := new(tb.SendOptions)
	reply := new(tb.ReplyMarkup)
	reply.InlineKeyboard = inlineKeyboards
	reply.ReplyKeyboardRemove = true
	options.ReplyMarkup = reply
	bot.Send(m.Sender, config.LangConfig.GetString("MESSAGES.THE_COMPANY_CHANNELS")+companyName+config.LangConfig.GetString("MESSAGES.GO_TO_COMPANY_CHANNEL_BY_CLICK"), options)
	return true
}

func (service *BotService) GetUserCurrentActiveChannel(db *sql.DB, app *config.App, bot *tb.Bot, m *tb.Message, userID int) *models.Channel {
	userModel := new(models.User)
	channelModel := new(models.Channel)
	companyModel := new(models.Company)
	channelSetting := new(models.ChannelSetting)
	var joinVerify, newMessageVerify, replyVerify, directVerify int
	if err := db.QueryRow("SELECT ch.uniqueID,ch.id,ch.channelID,ch.channelName,ch.manualChannelName,ch.channelURL,ch.channelType,us.id,us.userID,cc.companyID,IFNULL(cse.joinVerify,0) as joinVerify,IFNULL(cse.newMessageVerify,0) as newMessageVerify,IFNULL(cse.replyVerify,0) as replyVerify,IFNULL(cse.directVerify,0) as directVerify from `channels` as ch inner join `users_current_active_channel` as uc on ch.id=uc.channelID and uc.status='ACTIVE' inner join `users` as us on uc.userID=us.id and us.userID=? and us.`status`='ACTIVE' inner join companies_channels as cc on cc.channelID=ch.id left join channels_settings as cse on cse.channelID=ch.id", userID).Scan(&channelModel.UniqueID, &channelModel.ID, &channelModel.ChannelID, &channelModel.ChannelName, &channelModel.ManualChannelName, &channelModel.ChannelURL, &channelModel.ChannelType, &userModel.ID, &userModel.UserID, &companyModel.ID, &joinVerify, &newMessageVerify, &replyVerify, &directVerify); err != nil {
		log.Println(err)
		return channelModel
	}
	if joinVerify == 0 {
		channelSetting.JoinVerify = false
	} else {
		channelSetting.JoinVerify = true
	}
	if newMessageVerify == 0 {
		channelSetting.NewMessageVerify = false
	} else {
		channelSetting.NewMessageVerify = true
	}
	if replyVerify == 0 {
		channelSetting.ReplyVerify = false
	} else {
		channelSetting.ReplyVerify = true
	}
	if directVerify == 0 {
		channelSetting.DirectVerify = false
	} else {
		channelSetting.DirectVerify = true
	}
	channelModel.User = userModel
	channelModel.Company = companyModel
	channelModel.Setting = channelSetting
	return channelModel
}

func (service *BotService) GetChannelByTelegramID(db *sql.DB, app *config.App, channelID string) *models.Channel {
	channelModel := new(models.Channel)
	if err := db.QueryRow("SELECT id,`channelName` from `channels` where `channelID`=? ", channelID).Scan(&channelModel.ID, &channelModel.ChannelName); err != nil {
		log.Println(err)
		return channelModel
	}
	return channelModel
}

func SaveUserLastState(db *sql.DB, app *config.App, bot *tb.Bot, data string, userDataID int, state string) {
	userID := strconv.Itoa(userDataID)
	insertedState, err := db.Query("INSERT INTO `users_last_state` (`userID`,`state`,`data`,`createdAt`) VALUES('" + userID + "','" + state + "','" + strings.TrimSpace(data) + "','" + app.CurrentTime + "')")
	if err != nil {
		log.Println(err)
		return
	}
	defer insertedState.Close()
}
