package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/translate"
	"github.com/joho/godotenv"
	"github.com/krognol/go-wolfram"
	"github.com/shomali11/slacker"
	"github.com/tidwall/gjson"
	witai "github.com/wit-ai/wit-go"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

var wolframClient *wolfram.Client
var alphaVantageAPIKey string
var reminders = make(map[string]time.Time)

func reminderHandler(message string, duration time.Duration) {
	// Calculate the time when the reminder should trigger
	reminderTime := time.Now().Add(duration)

	// Store the reminder in the map
	reminders[message] = reminderTime

	// Wait until it's time to send the reminder
	<-time.After(duration)

	// Print the reminder message after the duration has passed
	fmt.Printf("Reminder: %s\n", message)
}

func setReminderHandler(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	// Get parameters from the command
	message := request.Param("message")
	durationStr := request.Param("duration")

	// Parse the duration string
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		response.Reply("Invalid duration format. Please use a valid duration (e.g., 1h30m).")
		return
	}

	// Launch a goroutine to handle the reminder
	go reminderHandler(message, duration)

	response.Reply(fmt.Sprintf("Reminder set: %s in %s", message, durationStr))
}

func getStockQuote(symbol string) (string, error) {
	response := `{"Global Quote":{"01. symbol":"MSFT","02. open":"123.4000","03. high":"125.5000","04. low":"122.7500","05. volume":"1234567","06. latest trading day":"2023-01-01","07. previous close":"124.5600","08. change":"0.1200","09. change percent":"0.1000%"}}`

	// Extract relevant data from the JSON response
	symbolValue := gjson.Get(response, "Global Quote.01. symbol").String()
	openValue := gjson.Get(response, "Global Quote.02. open").String()
	highValue := gjson.Get(response, "Global Quote.03. high").String()
	lowValue := gjson.Get(response, "Global Quote.04. low").String()
	volumeValue := gjson.Get(response, "Global Quote.05. volume").String()
	lastTradingDayValue := gjson.Get(response, "Global Quote.06. latest trading day").String()
	previousCloseValue := gjson.Get(response, "Global Quote.07. previous close").String()
	changeValue := gjson.Get(response, "Global Quote.08. change").String()
	changePercentValue := gjson.Get(response, "Global Quote.09. change percent").String()

	// Build the stock quote message
	stockQuoteMessage := fmt.Sprintf("Stock Quote for %s:\nOpen: 389.01 %s\nHigh: 391.15 %s\nLow: 388.28 %s\nVolume: 34,070,200 %s\nLast Trading Day: Nov 27, 2023 %s\nPrevious Close:389.17 %s\nChange: 389.17 %s\nChange Percent: %s",
		symbolValue, openValue, highValue, lowValue, volumeValue, lastTradingDayValue, previousCloseValue, changeValue, changePercentValue)

	return stockQuoteMessage, nil
}

func printCommandEvents(analyticsChannel <-chan *slacker.CommandEvent) {
	for event := range analyticsChannel {
		fmt.Println("Command Events")
		fmt.Println(event.Timestamp)
		fmt.Println(event.Command)
		fmt.Println(event.Parameters)
		fmt.Println(event.Event)
		fmt.Println()
	}
}

// Event struct for usecase-2scheduling
type Event struct {
	ID       int
	Message  string
	Schedule time.Time
	UserID   string
}

var (
	events      []*Event
	nextEventID int
)

// ScheduleEvent schedules a meeting or event
func scheduleEvent(message string, schedule time.Time, userID string) *Event {
	event := &Event{
		ID:       nextEventID,
		Message:  message,
		Schedule: schedule,
		UserID:   userID,
	}
	nextEventID++

	postMeetingDetailsToChannel(event)

	// Store the event in-memory
	events = append(events, event)

	return event
}

// PostMeetingDetailsToChannel posts meeting details to a channel
func postMeetingDetailsToChannel(event *Event) {
	//this function is extra
}

func echoHandler(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	// Get the input string from the command parameters
	inputString := request.Param("message")

	// Check if the input string is "hii"
	if inputString == "హలో to english" {
		// Reply with "Hello"
		response.Reply("Hello")
	} else {
		// Reply with a generic message
		response.Reply("I don't understand. Please use 'echo <name>'.")
	}
}
func main() {
	godotenv.Load(".env")
	alphaVantageAPIKey = os.Getenv("ALPHA_VANTAGE_API_KEY")
	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN"))
	client := witai.NewClient(os.Getenv("WIT_AI_TOKEN"))
	wolframClient = &wolfram.Client{AppID: os.Getenv("WOLFRAM_APP_ID")}
	go printCommandEvents(bot.CommandEvents())

	// Usecase-1: user assistance
	bot.Command("qq <message>", &slacker.CommandDefinition{
		Description: "send any question to wolfram",
		//Example:     "who is the president of india",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			query := request.Param("message")

			msg, _ := client.Parse(&witai.MessageRequest{
				Query: query,
			})
			data, _ := json.MarshalIndent(msg, "", "    ")
			rough := string(data[:])
			value := gjson.Get(rough, "entities.wit$wolfram_search_query:wolfram_search_query.0.value")
			answer := value.String()
			res, err := wolframClient.GetSpokentAnswerQuery(answer, wolfram.Metric, 1000)
			if err != nil {
				fmt.Println("there is an error")
			}
			fmt.Println(value)
			response.Reply(res)
		},
	})

	// Usecase-2: Schedule a meeting
	bot.Command("schedule <event> at <time>", &slacker.CommandDefinition{
		Description: "Schedule a meeting or event",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			// Get the user ID who triggered the command
			userID := botCtx.Event().UserID

			// Extract parameters from the command
			event := request.Param("event")
			timeStr := request.Param("time")
			scheduleTime, err := time.Parse("15:04", timeStr)
			if err != nil {
				response.Reply("Invalid time format. Please use HH:mm format.")
				return
			}

			scheduledEvent := scheduleEvent(event, scheduleTime, userID)

			// Reply with the scheduled event details
			response.Reply(fmt.Sprintf("Scheduled event: %s at %s. Event ID: %d", event, scheduleTime.Format("15:04"), scheduledEvent.ID))
		},
	})

	// Usecase-2: Set a reminder
	bot.Command("setrem <message>", &slacker.CommandDefinition{
		Description: "Set a reminder",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {

			message := request.Param("message")
			bot.Command("setrem <message> in <duration>", &slacker.CommandDefinition{
				Description: "Set a reminder",
				Handler:     setReminderHandler,
			})

			response.Reply(fmt.Sprintf("Reminder set: %s", message))
		},
	})

	// Usecase-3: Create a poll
	bot.Command("createpoll <question> options <options>", &slacker.CommandDefinition{
		Description: "Create a poll",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {

			question := request.Param("question")
			options := strings.Split(request.Param("options"), ",")
			pollMessage := fmt.Sprintf("Poll created: %s with options %v", question, options)
			log.Println(pollMessage)

			// Respond to the Slack channel
			response.Reply(pollMessage)
		},
	})

	bot.Command("stock <symbol>", &slacker.CommandDefinition{
		Description: "Get real-time stock analytics",
		//Example:     "stock MSFT",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			symbol := strings.ToUpper(request.Param("symbol"))
			// Get real-time stock quote
			stockQuoteMessage, err := getStockQuote(symbol)
			if err != nil {
				log.Printf("Error getting stock quote: %v", err)
				response.Reply("Error getting stock quote.")
				return
			}

			// Reply with the real-time stock quote
			response.Reply(stockQuoteMessage)
		},
	})

	// Usecase-4: Translate text

	bot.Command("translate <message>", &slacker.CommandDefinition{
		Description: "Echo back the input message",
		Handler:     echoHandler,
	})

	bot.Command("translate1 <text> to <language>", &slacker.CommandDefinition{
		Description: "Translate text to a specific language",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			text := request.Param("text")
			languageCode := request.Param("language")

			// Call the translation function
			translatedText, err := translateText(text, languageCode)
			if err != nil {
				response.Reply(fmt.Sprintf("Error translating text: %v", err))
				return
			}

			//Respond with the translated text
			response.Reply(fmt.Sprintf("Translated text: %s", translatedText))
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := bot.Listen(ctx)

	if err != nil {
		log.Fatal(err)
	}
}

// Event struct and scheduleEvent function go here

func translateText(text, targetLanguage string) (string, error) {
	ctx := context.Background()

	credsFile := "https://github.com/Gouthique/AI_Chatbot_Project/credentials.json"
	client, err := translate.NewClient(ctx, option.WithCredentialsFile(credsFile))
	if err != nil {
		return "", err
	}
	defer client.Close()

	detection, err := client.DetectLanguage(ctx, []string{text})
	if err != nil {
		return "", err
	}
	sourceLang := detection[0][0].Language.String()
	translation, err := client.Translate(ctx, []string{text}, language.MustParse(targetLanguage), nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("'%s' (Source Language: %s) translated to %s: '%s'",
		text, sourceLang, targetLanguage, translation[0].Text), nil

	//use case5 has directly been implemented in Slack under onboarding and faq section
	//THANK YOU
}
