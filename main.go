package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	Login    string
	Password string
	LoggedIn bool
}

const layout = "2006-01-02"

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	managerCredentials := make(map[int64]*Manager)

	var manager *Manager

	var start, end *time.Time

	// todo: add source
	vehicles := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Enterprise #1", "e 1"),
		tgbotapi.NewInlineKeyboardButtonData("Vehicle #2", "v 2"),

		tgbotapi.NewInlineKeyboardButtonData("Vehicle #1", "v 1"),
		tgbotapi.NewInlineKeyboardButtonData("Enterprise #2", "e 2"),

		tgbotapi.NewInlineKeyboardButtonData("Vehicle #3", "v 3"),
		tgbotapi.NewInlineKeyboardButtonData("Enterprise #3", "e 3"),

		tgbotapi.NewInlineKeyboardButtonData("Vehicle #4", "v 4"),
		tgbotapi.NewInlineKeyboardButtonData("Vehicle #5", "v 5"),
		tgbotapi.NewInlineKeyboardButtonData("Vehicle #6", "v 6"),
		tgbotapi.NewInlineKeyboardButtonData("Vehicle #7", "v 7"),
	}

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			data := update.CallbackQuery.Data

			switch {
			case strings.HasPrefix(data, "login"):
				m, ok := managerCredentials[chatID]
				if !ok {
					msg := tgbotapi.NewMessage(chatID, "Please enter your login and password (format: login password)")
					bot.Send(msg)

					for up := range updates {
						if up.Message == nil {
							continue
						}

						splitted := strings.Split(up.Message.Text, " ")
						if len(splitted) != 2 {
							msg := tgbotapi.NewMessage(chatID, "Please enter your login and password (format: login password)")
							bot.Send(msg)
							continue
						}

						manager = &Manager{
							Login:    splitted[0],
							Password: splitted[1],
							LoggedIn: true,
						}

						managerCredentials[chatID] = manager

						msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Hello, %s!", manager.Login))
						bot.Send(msg)

						break
					}
					continue
				}

				var msg tgbotapi.MessageConfig
				if !m.LoggedIn {
					msg = tgbotapi.NewMessage(chatID, "Welcome back!")
					m.LoggedIn = true
				} else {
					msg = tgbotapi.NewMessage(chatID, fmt.Sprintf("You've already been logged!"))
				}

				manager = m

				bot.Send(msg)

			case strings.HasPrefix(data, "date_today"):
				// Generate a report for today's date
				//generateReport(bot, chatID, "Today's Report", time.Now(), time.Now())

				today := time.Now()
				end = &today

				dayBefore := end.AddDate(0, 0, -1)
				start = &dayBefore
			case strings.HasPrefix(data, "date_this_week"):
				today := time.Now()
				end = &today

				weekBefore := end.AddDate(0, 0, -int(end.Weekday()))
				start = &weekBefore
			case strings.HasPrefix(data, "date_this_month"):
				today := time.Now()
				end = &today

				monthBefore := today.AddDate(0, -1, 0)
				start = &monthBefore
			case strings.HasPrefix(data, "date_custom"):
				msg := tgbotapi.NewMessage(chatID, "Please enter the custom date range in the format 'YYYY-MM-DD YYYY-MM-DD':")
				bot.Send(msg)

			case strings.HasPrefix(data, "next_"):
				page, _ := strconv.Atoi(strings.TrimPrefix(data, "next_"))
				page++
				updatePagination(bot, chatID, vehicles,
					page, update.CallbackQuery.Message.MessageID)
			case strings.HasPrefix(data, "prev_"):
				page, _ := strconv.Atoi(strings.TrimPrefix(data, "prev_"))
				page--
				updatePagination(bot, chatID, vehicles,
					page, update.CallbackQuery.Message.MessageID)

			default:
				switch {
				case strings.HasPrefix(data, "v"):
					//vehicleID := getID(data)
				case strings.HasPrefix(data, "e"):
					enterpriseID := getID(data)

					if start == nil {
						zeroTime := time.Unix(0, 0)
						start = &zeroTime
					}

					if end == nil {
						now := time.Now()
						end = &now
					}

					resp, err := http.Get(
						fmt.Sprintf("http://localhost:8080/api/enterprises/report?id=%v&start=%s&end=%s",
							enterpriseID, start.Format(layout), end.Format(layout)))

					if err != nil {
						sendToChat(chatID, "Can't get a report. Error has occurred", bot)
						continue
					}

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						sendToChat(chatID, "Can't get a report. Error has occurred", bot)
						continue
					}

					var result map[string]interface{}
					err = json.Unmarshal(body, &result)
					if err != nil {
						sendToChat(chatID, "Can't get a report. Error has occurred", bot)
						continue
					}

					v, ok := result["mileage"]
					if !ok {
						sendToChat(chatID, "Can't get a report. Error has occurred", bot)
						continue
					}

					msg := ""
					if fmt.Sprintf("%v", v) == "0" {
						msg = fmt.Sprintf("Enterprise #%d mileage during %s-%s wasn't calculate. Not enough data.",
							enterpriseID, start.Format(layout), end.Format(layout))
					} else {
						msg = fmt.Sprintf("Enterprise #%d mileage during %s-%s is %v km!",
							enterpriseID, start.Format(layout), end.Format(layout), v)
					}

					sendToChat(chatID, msg, bot)

					resp.Body.Close()
				}

			}
		}

		if update.Message != nil {
			chatID := update.Message.Chat.ID

			switch update.Message.Text {
			case "/login":
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Login", "login"),
					),
				)

				msg := tgbotapi.NewMessage(chatID, "Please click the 'Login' button to enter your credentials:")
				msg.ReplyMarkup = inlineKeyboard
				bot.Send(msg)
			case "/report":
				if !isAuth(manager, chatID, bot) {
					continue
				}

				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Today", "date_today"),
						tgbotapi.NewInlineKeyboardButtonData("This Week", "date_this_week"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("This Month", "date_this_month"),
						tgbotapi.NewInlineKeyboardButtonData("Custom Range", "date_custom"),
					),
				)

				msg := tgbotapi.NewMessage(chatID, "Please choose a date range for the report:")
				msg.ReplyMarkup = inlineKeyboard
				bot.Send(msg)

			default:
				if strings.HasPrefix(update.Message.Text, "date_custom:") {
					input := strings.TrimPrefix(update.Message.Text, "date_custom:")
					dateRange := strings.Split(input, " ")
					if len(dateRange) == 2 {
						startDate, err := time.Parse("2006-01-02", dateRange[0])
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Invalid start date format. Please use 'YYYY-MM-DD'."))
							continue
						}
						endDate, err := time.Parse("2006-01-02", dateRange[1])
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Invalid end date format. Please use 'YYYY-MM-DD'."))
							continue
						}
						if endDate.Before(startDate) {
							bot.Send(tgbotapi.NewMessage(chatID, "End date cannot be before start date."))
							continue
						}

						// Generate a report for the custom date range
						generateReport(bot, chatID, "Custom Date Range Report", startDate, endDate)
					} else {
						bot.Send(tgbotapi.NewMessage(chatID, "Invalid input format. Please use 'YYYY-MM-DD YYYY-MM-DD'."))
					}

					continue
				}

				start, end = parseDate(update.Message.Text)
				if start != nil && end != nil {
					showKeyboardReportsKind(chatID, bot, vehicles)
					continue
				}

				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("I got from you: %v", update.Message.Text)))
			}
		}
	}
}

func showKeyboardReportsKind(chatID int64, bot *tgbotapi.BotAPI,
	src []tgbotapi.InlineKeyboardButton) {
	//inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
	//	tgbotapi.NewInlineKeyboardRow(
	//		tgbotapi.NewInlineKeyboardButtonData("Vehicle #1", "v 1"),
	//		tgbotapi.NewInlineKeyboardButtonData("Vehicle #2", "v 2"),
	//		tgbotapi.NewInlineKeyboardButtonData("Vehicle #3", "v 3"),
	//	),
	//	tgbotapi.NewInlineKeyboardRow(
	//		tgbotapi.NewInlineKeyboardButtonData("Enterprise #1", "e 1"),
	//		tgbotapi.NewInlineKeyboardButtonData("Enterprise #2", "e 2"),
	//		tgbotapi.NewInlineKeyboardButtonData("Enterprise #3", "e 3"),
	//	),
	//)

	inlineKeyboard := createPaginationKeyboard(src, 1)

	msg := tgbotapi.NewMessage(chatID, "What kind of report do you want?")
	msg.ReplyMarkup = inlineKeyboard
	bot.Send(msg)
}

func createPaginationKeyboard(buttons []tgbotapi.InlineKeyboardButton, currentPage int) tgbotapi.InlineKeyboardMarkup {
	perPage := 3

	// Calculate the maximum number of pages based on the buttons count
	morePages := (len(buttons) + perPage - 1) / perPage

	// Create keyboard rows for the current page
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < perPage && i < len(buttons); i++ {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttons[i].Text, *buttons[i].CallbackData),
		})
	}

	// Add pagination buttons
	paginationRow := []tgbotapi.InlineKeyboardButton{}
	if currentPage > 1 {
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("Previous", fmt.Sprintf("prev_%d", currentPage)))
	}
	if morePages > 1 {
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("Next", fmt.Sprintf("next_%d", currentPage)))
	}
	rows = append(rows, paginationRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func updatePagination(bot *tgbotapi.BotAPI, chatID int64,
	src []tgbotapi.InlineKeyboardButton,
	page, messageID int) {

	i := len(src) - 1 - (page-1)*3 // todo: add constant of max on page

	keyboard := createPaginationKeyboard(src[i:], page)

	// Edit the existing message to update the keyboard
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
	bot.Send(editMsg)
}

func getID(s string) int64 {
	splitted := strings.Split(s, " ")
	if len(splitted) != 2 {
		return 0
	}

	id := splitted[1]
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0
	}

	return v
}

func isAuth(manager *Manager, chatID int64, bot *tgbotapi.BotAPI) bool {
	if manager == nil || !manager.LoggedIn {
		msg := tgbotapi.NewMessage(chatID, "Please login first.")
		bot.Send(msg)

		return false
	}

	return true
}

func generateReport(bot *tgbotapi.BotAPI, chatID int64, title string, startDate, endDate time.Time) {
	reportMessage := fmt.Sprintf("%s\nStart Date: %s\nEnd Date: %s", title, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	bot.Send(tgbotapi.NewMessage(chatID, reportMessage))
}

func parseDate(s string) (*time.Time, *time.Time) {
	splitted := strings.Split(s, " ")
	if len(splitted) != 2 {
		return nil, nil
	}

	t1, err := time.Parse(layout, splitted[0])
	if err != nil {
		return nil, nil
	}

	t2, err := time.Parse(layout, splitted[1])
	if err != nil {
		return nil, nil
	}

	return &t1, &t2
}

func sendToChat(chat int64,
	message string,
	bot *tgbotapi.BotAPI) {

	msg := tgbotapi.NewMessage(chat, message)
	bot.Send(msg)
}
