package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var gBot *tgbotapi.BotAPI
var gToken string
var gChatId int64

var gUsersInChat Users

type User struct {
	id int64
	name string
	coins uint16
}

type Users []*User

type Activity struct {
	code, name string
	coins      uint16
}
type Activities []*Activity

var gUsefulActivities = Activities{
	// Self-Development
	{"yoga", "Yoga (15 minutes)", 1},
	{"meditation", "Meditation (15 minutes)", 1},
	{"language", "Learning a foreign language (15 minutes)", 1},
	{"swimming", "Swimming (15 minutes)", 1},
	{"walk", "Walk (15 minutes)", 1},
	{"chores", "Chores", 1},

	// Work
	{"work_learning", "Studying work materials (15 minutes)", 1},
	{"portfolio_work", "Working on a portfolio project (15 minutes)", 1},
	{"resume_edit", "Resume editing (15 minutes)", 1},

	// Creativity
	{"creative", "Creative creation (15 minutes)", 1},
	{"reading", "Reading fiction literature (15 minutes)", 1},
}

var gRewards = Activities{
	// Entertainment
	{"watch_series", "Watching a series (1 episode)", 10},
	{"watch_movie", "Watching a movie (1 item)", 30},
	{"social_nets", "Browsing social networks (30 minutes)", 10},

	// Food
	{"eat_sweets", "300 kcal of sweets", 60},
}

const (
	EMOJI_COIN         = "\U0001FA99"   // (coin)
	EMOJI_SMILE        = "\U0001F642"   // 🙂
	EMOJI_SUNGLASSES   = "\U0001F60E"   // 😎
	EMOJI_WOW          = "\U0001F604"   // 😄
	EMOJI_DONT_KNOW    = "\U0001F937"   // 🤷
	EMOJI_SAD          = "\U0001F63F"   // 😿
	EMOJI_BICEPS       = "\U0001F4AA"   // 💪
	EMOJI_BUTTON_START = "\U000025B6  " // ▶
	EMOJI_BUTTON_END   = "  \U000025C0" // ◀

	BUTTON_TEXT_PRINT_INTRO       = EMOJI_BUTTON_START + "View Introduction" + EMOJI_BUTTON_END
	BUTTON_TEXT_SKIP_INTRO        = EMOJI_BUTTON_START + "Skip Introduction" + EMOJI_BUTTON_END
	BUTTON_TEXT_BALANCE           = EMOJI_BUTTON_START + "Current Balance" + EMOJI_BUTTON_END
	BUTTON_TEXT_USEFUL_ACTIVITIES = EMOJI_BUTTON_START + "Useful Activities" + EMOJI_BUTTON_END
	BUTTON_TEXT_REWARDS           = EMOJI_BUTTON_START + "Rewards" + EMOJI_BUTTON_END
	BUTTON_TEXT_PRINT_MENU        = EMOJI_BUTTON_START + "MAIN MENU" + EMOJI_BUTTON_END

	BUTTON_CODE_PRINT_INTRO       = "print_intro"
	BUTTON_CODE_SKIP_INTRO        = "skip_intro"
	BUTTON_CODE_BALANCE           = "show_balance"
	BUTTON_CODE_USEFUL_ACTIVITIES = "show_useful_activities"
	BUTTON_CODE_REWARDS           = "show_rewards"
	BUTTON_CODE_PRINT_MENU        = "print_menu"

	TOKEN_NAME_IN_OS             = "improve_your_skills_bot"
	UPDATE_CONFIG_TIMEOUT  = 60
	MAX_USER_COINS        uint16 = 500
)

func init() {
	var err error

	err = godotenv.Load()
	if err != nil {
	  log.Fatal("Error loading .env file")
	}

	if gToken = os.Getenv("TELEGRAM_BOT_TOKEN"); gToken == ""  {
		fmt.Println(gToken)
		log.Fatal("TELEGRAM_BOT_TOKEN is not found in the environment")
	}

	if gBot, err = tgbotapi.NewBotAPI(gToken); err != nil {
		log.Panic(err)
	}

	gBot.Debug = true
}

func main() {
	log.Printf("Authorized on account %s", gBot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = UPDATE_CONFIG_TIMEOUT

	updates := gBot.GetUpdatesChan(updateConfig)

	for update := range updates {
		//CallbackQuery - кнопки которые нажимаем
		if isCallbackQuery(&update){
			updateProcessing(&update)
		} else if isStartMessage(&update) { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			gChatId = update.Message.Chat.ID

			askToPrintIntro()

			//update.Message.Text - то что нам прислал user
			// msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			// msg.ReplyToMessageID = update.Message.MessageID
			// gBot.Send(msg)
		}
	}
}

func updateProcessing(update *tgbotapi.Update) {
	user, found := getUserFromUpdate(update)

	if !found {
		if user, found = storeUserFromUpdate(update); !found {
			sendStringMessage("Unable to identify the user")
			return
		}
	}

	//поймем введенную команду (кнопку которую нажал)
	choiseCode := update.CallbackQuery.Data
	log.Printf("[%T] %s", time.Now(), choiseCode)

	switch choiseCode {
	case BUTTON_CODE_BALANCE:
		showBalance(update, user)
	case BUTTON_CODE_USEFUL_ACTIVITIES:
		showUsefulActivities()
	case BUTTON_CODE_REWARDS:
		showRewards()
	case BUTTON_CODE_PRINT_INTRO:
		printIntro(update)
		//выводим пунты меню
		showMenu()
	case BUTTON_CODE_PRINT_MENU:
		showMenu()
	case BUTTON_CODE_SKIP_INTRO:
		showMenu()
	default:
		if usefulActivity, found := findActivity(gUsefulActivities, choiseCode); found {
			processUsefulActivity(usefulActivity, user)

			delay(2)
			showUsefulActivities()
			return
		}

		if reward, found := findActivity(gRewards, choiseCode); found {
			processReward(reward, user)

			delay(2)
			showUsefulActivities()
			return
		}
		
		log.Printf(`[%T] !!!!!!!!! ERROR: Unknown code "%s"`, time.Now(), choiseCode)
		msg := fmt.Sprintf("%s, I'm sorry, I don't recognize code '%s' %s Please report this error to my creator.", user.name, choiseCode, EMOJI_SAD)
		sendStringMessage(msg)
	}
}

func showRewards() {
	showActivities(gRewards, "Purchase a reward or return to the main menu:", false)
}

func processReward(activity *Activity, user *User) {
	errorMsg := ""
	if activity.coins == 0 {
		errorMsg = fmt.Sprintf(`the reward "%s" doesn't have a specified cost`, activity.name)
	} else if user.coins < activity.coins {
		errorMsg = fmt.Sprintf(`you currently have %d %s. You cannot afford "%s" for %d %s`, user.coins, EMOJI_COIN, activity.name, activity.coins, EMOJI_COIN)
	}

	resultMessage := ""
	if errorMsg != "" {
		resultMessage = fmt.Sprintf("%s, I'm sorry, but %s %s Your balance remains unchanged, the reward is unavailable %s", user.name, errorMsg, EMOJI_SAD, EMOJI_DONT_KNOW)
	} else {
		user.coins -= activity.coins
		resultMessage = fmt.Sprintf(`%s, the reward "%s" has been paid for, get started! %d %s has been deducted from your account. Now you have %d %s`, user.name, activity.name, activity.coins, EMOJI_COIN, user.coins, EMOJI_COIN)
	}
	sendStringMessage(resultMessage)
}

func findActivity(activities Activities, code string) (activity *Activity, found bool) {
	for _, activity := range activities {
		if code == activity.code {
			return activity, true
		}
	}
	return
}

func showUsefulActivities() {
	showActivities(gUsefulActivities, "Track a useful activity or return to the main menu:", true)
}

func showActivities(activities Activities, message string, isUseful bool) {
	activitiesButtonsRows := make([]([]tgbotapi.InlineKeyboardButton), 0, len(activities) + 1)

	for _, activity := range activities {
		activityDescription := ""
		if isUseful {
			activityDescription = fmt.Sprintf("+ %d %s: %s", activity.coins, EMOJI_COIN, activity.name)
		} else {
			activityDescription = fmt.Sprintf("- %d %s: %s", activity.coins, EMOJI_COIN, activity.name)
		}
		activitiesButtonsRows = append(activitiesButtonsRows, getKeyboardRow(activityDescription, activity.code))
	}

	activitiesButtonsRows = append(activitiesButtonsRows, getKeyboardRow(BUTTON_TEXT_PRINT_MENU, BUTTON_CODE_PRINT_MENU))

	msg := tgbotapi.NewMessage(gChatId, message)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(activitiesButtonsRows...)

	gBot.Send(msg)
}

func processUsefulActivity(activity *Activity, user *User) {
	errorMsg := ""
	if activity.coins == 0 {
		errorMsg = fmt.Sprintf(`the activity "%s" doesn't have a specified cost`, activity.name)
	} else if user.coins+activity.coins > MAX_USER_COINS {
		errorMsg = fmt.Sprintf("you cannot have more than %d %s", MAX_USER_COINS, EMOJI_COIN)
	}

	resultMessage := ""
	if errorMsg != "" {
		resultMessage = fmt.Sprintf("%s, I'm sorry, but %s %s Your balance remains unchanged.", user.name, errorMsg, EMOJI_SAD)
	} else {
		user.coins += activity.coins
		resultMessage = fmt.Sprintf(`%s, the "%s" activity is completed! %d %s has been added to your account. Keep it up! %s%s Now you have %d %s`,
			user.name, activity.name, activity.coins, EMOJI_COIN, EMOJI_BICEPS, EMOJI_SUNGLASSES, user.coins, EMOJI_COIN)
	}
	sendStringMessage(resultMessage)
}

func storeUserFromUpdate(update *tgbotapi.Update) (user *User, found bool) {
	if callbackQueryFromIsMissing(update) {
		return
	}

	from := update.CallbackQuery.From
	user = &User{
		id: from.ID,
		name: strings.TrimSpace(from.FirstName + " " + from.LastName),
		coins: 0,
	}
	gUsersInChat = append(gUsersInChat, user)
	return user, true
}

func getUserFromUpdate(update *tgbotapi.Update) (user *User, found bool) {
	if callbackQueryFromIsMissing(update) {
		return
	}

	userId := update.CallbackQuery.From.ID

	for _, userInChat := range gUsersInChat {
		if userId == userInChat.id {
			return userInChat, true
		}
	}

	return
}

func callbackQueryFromIsMissing(update *tgbotapi.Update) bool {
	return update.CallbackQuery == nil || update.CallbackQuery.From == nil
}

func showBalance(update *tgbotapi.Update, user *User) {
	msg := fmt.Sprintf("%s, your wallet is currently empty %s \nTrack a useful activity to earn coins", user.name, EMOJI_DONT_KNOW)
	if coins := user.coins; coins > 0 {
		msg = fmt.Sprintf("%s, you have %d %s", user.name, coins, EMOJI_COIN)
	}
	sendStringMessage(msg)
	showMenu()
}

func showMenu() {
	msg := tgbotapi.NewMessage(gChatId, "Choose one of the options:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		getKeyboardRow(BUTTON_TEXT_BALANCE, BUTTON_CODE_BALANCE),
		getKeyboardRow(BUTTON_TEXT_USEFUL_ACTIVITIES, BUTTON_CODE_USEFUL_ACTIVITIES),
		getKeyboardRow(BUTTON_TEXT_REWARDS, BUTTON_CODE_REWARDS),
	)

	gBot.Send(msg)
}

func printIntro(update *tgbotapi.Update) {
	sendMessageWithDelay(2, "Hello! "+EMOJI_SUNGLASSES)
	sendMessageWithDelay(7, "There are numerous beneficial actions that, by performing regularly, we improve the quality of our life. However, often it's more fun, easier, or tastier to do something harmful. Isn't that so?")
	sendMessageWithDelay(7, "With greater likelihood, we'll prefer to get lost in YouTube Shorts instead of an English lesson, buy M&M's instead of vegetables, or lie in bed instead of doing yoga.")
	sendMessageWithDelay(1, EMOJI_SAD)
	sendMessageWithDelay(10, "Everyone has played at least one game where you need to level up a character, making them stronger, smarter, or more beautiful. It's enjoyable because each action brings results. In real life, though, systematic actions over time start to become noticeable. Let's change that, shall we?")
	sendMessageWithDelay(1, EMOJI_SMILE)
	sendMessageWithDelay(14, `Before you are two tables: "Useful Activities" and "Rewards". The first table lists simple short activities, and for completing each of them, you'll earn the specified amount of coins. In the second table, you'll see a list of activities you can only do after paying for them with coins earned in the previous step.`)
	sendMessageWithDelay(1, EMOJI_COIN)
	sendMessageWithDelay(10, `For example, you spend half an hour doing yoga, for which you get 2 coins. After that, you have 2 hours of programming study, for which you get 8 coins. Now you can watch 1 episode of "Interns" and break even. It's that simple!`)
	sendMessageWithDelay(6, `Mark completed useful activities to not lose your coins. And don't forget to "purchase" the reward before actually doing it.`)
}

func sendMessageWithDelay(delayInSec uint8, message string) {
	sendStringMessage(message)
	delay(delayInSec)
}

func delay(seconds uint8) {
	time.Sleep(time.Second * time.Duration(seconds))
}

func sendStringMessage(msg string) {
	gBot.Send(tgbotapi.NewMessage(gChatId, msg))
}

func isStartMessage(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Text == "/start"
}

func isCallbackQuery(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && update.CallbackQuery.Data != ""
}

func askToPrintIntro() {
	msg := tgbotapi.NewMessage(gChatId, "In the introductory messages, you can find the purpose of this bot and the rules of the game. What do you think?")
	//будем отвечать
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		// будут ответы да и нет
		getKeyboardRow(BUTTON_TEXT_PRINT_INTRO, BUTTON_CODE_PRINT_INTRO),
		getKeyboardRow(BUTTON_TEXT_SKIP_INTRO, BUTTON_CODE_SKIP_INTRO),
	)

	gBot.Send(msg)
}

func getKeyboardRow(buttonText, buttonCode string) []tgbotapi.InlineKeyboardButton {
	//создаем кнопки ответа
	//NewInlineKeyboardButtonData - в параметрах имя и код - ответ от пользователя
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCode))
}