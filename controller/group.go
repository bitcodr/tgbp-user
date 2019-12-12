package controller

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	"github.com/amiraliio/tgbp/config"
	"github.com/amiraliio/tgbp/model"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
)

func RegisterGroup(bot *tb.Bot, m *tb.Message) {
	channelID := strconv.FormatInt(m.Chat.ID, 10)
	db, err := config.DB()
	if err != nil {
		log.Println(err)
	}
	defer db.Close()
	results, err := db.Query("SELECT id FROM `channels` where channelID=" + channelID)
	if err != nil {
		log.Println(err)
	}
	if !results.Next() {
		//start transaction
		transaction, err := db.Begin()
		if err != nil {
			log.Println(err)
		}
		//insert channel
		channelInserted, err := transaction.Exec("INSERT INTO `channels` (`channelType`,`channelID`,`channelName`,`createdAt`,`updatedAt`) VALUES('group','" + channelID + "','" + m.Chat.Title + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		insertedChannelID, err := channelInserted.LastInsertId()
		if err == nil {
			//company name
			companyFlag := strconv.FormatInt(m.Chat.ID, 10)
			//check if company is not exist
			companyExists, err := db.Query("SELECT id FROM `companies` where `companyName`='" + companyFlag + "'")
			if err != nil {
				transaction.Rollback()
				log.Println(err)
			}
			if !companyExists.Next() {
				//insert company
				companyInserted, err := transaction.Exec("INSERT INTO `companies` (`companyName`,`createdAt`,`updatedAt`) VALUES('" + companyFlag + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
				if err != nil {
					transaction.Rollback()
					log.Println(err)
				}
				insertedCompanyID, err := companyInserted.LastInsertId()
				if err == nil {
					companyModelID := strconv.FormatInt(insertedCompanyID, 10)
					channelModelID := strconv.FormatInt(insertedChannelID, 10)
					//insert company channel pivot
					_, err := transaction.Exec("INSERT INTO `companies_channels` (`companyID`,`channelID`,`createdAt`) VALUES('" + companyModelID + "','" + channelModelID + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
					if err != nil {
						transaction.Rollback()
						log.Println(err)
					}
				}
			} else {
				companyModel := new(model.Company)
				if err := companyExists.Scan(&companyModel.ID); err != nil {
					transaction.Rollback()
					log.Println(err)
				}
				companyModelID := strconv.FormatInt(companyModel.ID, 10)
				channelModelID := strconv.FormatInt(insertedChannelID, 10)
				//insert company channel pivot
				_, err := transaction.Exec("INSERT INTO `companies_channels` (`companyID`,`channelID`,`createdAt`) VALUES('" + companyModelID + "','" + channelModelID + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
				if err != nil {
					transaction.Rollback()
					log.Println(err)
				}
			}
			transaction.Commit()
			successMessage, _ := bot.Send(m.Chat, "You're group registered successfully")
			time.Sleep(3 * time.Second)
			if err := bot.Delete(successMessage); err != nil {
				log.Println(err)
			}
			compose := tb.InlineButton{
				Unique: "compose_message_in_group_" + channelID,
				Text:   "Compose",
				URL:    "https://t.me/" + viper.GetString("APP.BOTUSERNAME") + "?start=compose_message_in_group_" + channelID,
			}
			groupKeys := [][]tb.InlineButton{
				[]tb.InlineButton{compose},
			}
			newReplyModel := new(tb.ReplyMarkup)
			newReplyModel.InlineKeyboard = groupKeys
			newSendOption := new(tb.SendOptions)
			newSendOption.ReplyMarkup = newReplyModel
			_, err = bot.Send(m.Chat, "Please use inline keyboard to start anonymous comminication", newSendOption)
			if err != nil {
				log.Println(err)
			}
		}
	} else {
		compose := tb.InlineButton{
			Unique: "compose_message_in_group_" + channelID,
			Text:   "Compose",
			URL:    "https://t.me/" + viper.GetString("APP.BOTUSERNAME") + "?start=compose_message_in_group_" + channelID,
		}
		groupKeys := [][]tb.InlineButton{
			[]tb.InlineButton{compose},
		}
		newReplyModel := new(tb.ReplyMarkup)
		newReplyModel.InlineKeyboard = groupKeys
		newSendOption := new(tb.SendOptions)
		newSendOption.ReplyMarkup = newReplyModel
		_, err = bot.Send(m.Chat, "Please use inline keyboard to start anonymous comminication", newSendOption)
		if err != nil {
			log.Println(err)
		}
	}
}

func NewMessageGroupHandler(bot *tb.Bot, m *tb.Message) {
	bot.Send(m.Sender, "Please send your message:")
}

func JoinFromGroup(bot *tb.Bot, m *tb.Message, channelID string) {
	db, err := config.DB()
	if err != nil {
		log.Println(err)
	}
	defer db.Close()
	userID := strconv.Itoa(m.Sender.ID)
	//check if user is not created
	results, err := db.Query("SELECT id FROM `users` where `status`= 'ACTIVE' and `userID`='" + userID + "'")
	if err != nil {
		log.Println(err)
	}
	//start transaction
	transaction, err := db.Begin()
	if err != nil {
		log.Println(err)
	}
	if !results.Next() {
		//insert user
		var isBotValue string
		if m.Sender.IsBot {
			isBotValue = "1"
		} else {
			isBotValue = "0"
		}
		userInsert, err := transaction.Exec("INSERT INTO `users` (`userID`,`username`,`firstName`,`lastName`,`lang`,`isBot`,`createdAt`,`updatedAt`) VALUES('" + userID + "','" + m.Sender.Username + "','" + m.Sender.FirstName + "','" + m.Sender.LastName + "','" + m.Sender.LanguageCode + "','" + isBotValue + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		insertedUserID, err := userInsert.LastInsertId()
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		checkAndInsertUserGroup(bot, m, insertedUserID, channelID, db, transaction)
	} else {
		userModel := new(model.User)
		if err := results.Scan(&userModel.ID); err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		checkAndInsertUserGroup(bot, m, userModel.ID, channelID, db, transaction)
	}
}

func checkAndInsertUserGroup(bot *tb.Bot, m *tb.Message, queryUserID int64, channelID string, db *sql.DB, transaction *sql.Tx) {
	userModelID := strconv.FormatInt(queryUserID, 10)
	//check if channel for user is exists
	checkUserChannel, err := db.Query("SELECT ch.id as id FROM `channels` as ch inner join `users_channels` as uc on uc.channelID = ch.id and uc.userID='" + userModelID + "' and ch.channelID='" + channelID + "'")
	if err != nil {
		transaction.Rollback()
		log.Println(err)
	}
	if !checkUserChannel.Next() {
		getChannelID, err := db.Query("SELECT `id` FROM `channels` where `channelID`='" + channelID + "'")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		if getChannelID.Next() {
			channelModel := new(model.Channel)
			if err := getChannelID.Scan(&channelModel.ID); err != nil {
				transaction.Rollback()
				log.Println(err)
			}
			channelModelID := strconv.FormatInt(channelModel.ID, 10)
			_, err := transaction.Exec("INSERT INTO `users_channels` (`status`,`userID`,`channelID`,`createdAt`,`updatedAt`) VALUES('ACTIVE','" + userModelID + "','" + channelModelID + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
			if err != nil {
				transaction.Rollback()
				log.Println(err)
			}
		}
	}
	channelDetail, err := db.Query("SELECT ch.`id` as id, ch.channelName as channelName, co.companyName as companyName  FROM `channels` as ch inner join `companies_channels` as cc on ch.id = cc.channelID inner join `companies` as co on cc.companyID = co.id where ch.`channelID`='" + channelID + "'")
	if err != nil {
		transaction.Rollback()
		log.Println(err)
	}
	if channelDetail.Next() {
		channelModelData := new(model.Channel)
		companyModel := new(model.Company)
		if err := channelDetail.Scan(&channelModelData.ID, &channelModelData.ChannelName, &companyModel.CompanyName); err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		channelModelID := strconv.FormatInt(channelModelData.ID, 10)
		//check the user is active or not
		_, err := db.Query("SELECT `id`,`status` from `users_channels` where `userID`='" + userModelID + "' and `channelID`='" + channelModelID + "' and `status`='ACTIVE'")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		//inactive user last active channels
		_, err = transaction.Exec("update `users_current_active_channel` set `status`='INACTIVE' where `userID`='" + userModelID + "'")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		//set user active channel
		_, err = transaction.Exec("INSERT INTO `users_current_active_channel` (`userID`,`channelID`,`createdAt`,`updatedAt`) VALUES('" + userModelID + "','" + channelModelID + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "','" + time.Now().UTC().Format("2006-01-02 03:04:05") + "')")
		if err != nil {
			transaction.Rollback()
			log.Println(err)
		}
		transaction.Commit()
	}
}