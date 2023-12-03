package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/translate"
	"github.com/joho/godotenv"
	"github.com/krognol/go-wolfram"
	"github.com/shomali11/slacker"
	"github.com/slack-go/slack"
	"github.com/tidwall/gjson"
	witai "github.com/wit-ai/wit-go"

	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

var wolframClient *wolfram.Client
var alphaVantageAPIKey string
var reminders = make(map[string]time.Time)
var googleAPIKey string

// ------------------------------------- Usecase-2 TaskScheduling--------------------------------------------
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

func sendPollResult(channelID, message string) {
	api := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	_, _, err := api.PostMessage(channelID, slack.MsgOptionText(message, false))
	if err != nil {
		log.Printf("Error sending poll result: %v", err)
	}
}

// ------------------------End of Usecase2 Stuff-------------------------------------------------------
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

// // ScheduleEvent schedules a meeting or event
// func scheduleEvent(message string, schedule time.Time, userID string) *Event {
// 	event := &Event{
// 		ID:       nextEventID,
// 		Message:  message,
// 		Schedule: schedule,
// 		UserID:   userID,
// 	}
// 	nextEventID++

// 	postMeetingDetailsToChannel(event)

// 	// Store the event in-memory
// 	events = append(events, event)

// 	return event
// }

// // PostMeetingDetailsToChannel posts meeting details to a channel
// func postMeetingDetailsToChannel(event *Event) {
// 	//this function is extra
// }

// -------------------------------------------------------- Usecase-3: Stock analytics-------------------------------------------------------------------------
type StockQuote struct {
	Symbol         string `json:"01. symbol"`
	Open           string `json:"02. open"`
	High           string `json:"03. high"`
	Low            string `json:"04. low"`
	Volume         string `json:"05. volume"`
	LastTradingDay string `json:"06. latest trading day"`
	PreviousClose  string `json:"07. previous close"`
	Change         string `json:"08. change"`
	ChangePercent  string `json:"09. change percent"`
}

// func getStockQuote(symbol string) (string, error) {
// 	response := `{"Global Quote":{"01. symbol":"MSFT","02. open":"123.4000","03. high":"125.5000","04. low":"122.7500","05. volume":"1234567","06. latest trading day":"2023-01-01","07. previous close":"124.5600","08. change":"0.1200","09. change percent":"0.1000%"}}`

// 	// Extract relevant data from the JSON response
// 	symbolValue := gjson.Get(response, "Global Quote.01. symbol").String()
// 	openValue := gjson.Get(response, "Global Quote.02. open").String()
// 	highValue := gjson.Get(response, "Global Quote.03. high").String()
// 	lowValue := gjson.Get(response, "Global Quote.04. low").String()
// 	volumeValue := gjson.Get(response, "Global Quote.05. volume").String()
// 	lastTradingDayValue := gjson.Get(response, "Global Quote.06. latest trading day").String()
// 	previousCloseValue := gjson.Get(response, "Global Quote.07. previous close").String()
// 	changeValue := gjson.Get(response, "Global Quote.08. change").String()
// 	changePercentValue := gjson.Get(response, "Global Quote.09. change percent").String()

// 	// Build the stock quote message
// 	stockQuoteMessage := fmt.Sprintf("Stock Quote for %s:\nOpen: 389.01 %s\nHigh: 391.15 %s\nLow: 388.28 %s\nVolume: 34,070,200 %s\nLast Trading Day: Nov 27, 2023 %s\nPrevious Close:389.17 %s\nChange: 389.17 %s\nChange Percent: %s",
// 		symbolValue, openValue, highValue, lowValue, volumeValue, lastTradingDayValue, previousCloseValue, changeValue, changePercentValue)

// 	return stockQuoteMessage, nil
// }

// func getStockQuote(symbol string) (string, error) {
// 	// Replace this with the actual API endpoint for retrieving stock data
// 	apiUrl := fmt.Sprintf("/%s", symbol)

// 	// Send an HTTP GET request to the API
// 	response, err := http.Get(apiUrl)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer response.Body.Close()

// 	// Read the response body
// 	body, err := ioutil.ReadAll(response.Body)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Parse the JSON response
// 	// Assuming the JSON structure matches the one you provided
// 	// Extract relevant data from the JSON response

//		return stockQuoteMessage, nil
//	}

func getStockQuote(symbol string) (string, error) {
	// Set up your Alpha Vantage API endpoint and API key
	apiUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_INTRADAY&symbol=%s&interval=5min&apikey=%s", symbol, alphaVantageAPIKey)

	// Send an HTTP GET request to the Alpha Vantage API
	response, err := http.Get(apiUrl)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	// Check for errors in the Alpha Vantage response
	if errorMessage, ok := data["Error Message"].(string); ok {
		return "", fmt.Errorf("Alpha Vantage API error: %s", errorMessage)
	}

	// Extract relevant data from the JSON response
	timeSeries, timeSeriesExists := data["Time Series (5min)"].(map[string]interface{})

	if !timeSeriesExists {
		return "", fmt.Errorf("Missing data in Alpha Vantage response")
	}

	// Find the top 3 data points
	var top3Data []string
	count := 0
	for key, value := range timeSeries {
		if count >= 3 {
			break
		}

		dataPoint := value.(map[string]interface{})
		openValue, _ := dataPoint["1. open"].(string)
		highValue, _ := dataPoint["2. high"].(string)
		lowValue, _ := dataPoint["3. low"].(string)
		closeValue, _ := dataPoint["4. close"].(string)
		volumeValue, _ := dataPoint["5. volume"].(string)

		dataPointStr := fmt.Sprintf("Timestamp: %s\nOpen: %s\nHigh: %s\nLow: %s\nClose: %s\nVolume: %s\n\n", key, openValue, highValue, lowValue, closeValue, volumeValue)
		top3Data = append(top3Data, dataPointStr)

		count++
	}

	// Combine the top 3 data points into a single message
	stockQuoteMessage := fmt.Sprintf("Top 3 data points for %s:\n\n%s", symbol, strings.Join(top3Data, "\n"))
	return stockQuoteMessage, nil
}

// --------------------------------------------------------Usecase-4: Language Translation-------------------------------------------------------------------------
// Initialize a Translation client
func initTranslationClient() (*translate.Client, error) {
	ctx := context.Background()

	client, err := translate.NewClient(ctx, option.WithAPIKey(googleAPIKey))
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Function to perform text translation
func translateText(client *translate.Client, targetTag language.Tag, text string) (string, error) {
	ctx := context.Background()

	// Translate the text to the target language
	translations, err := client.Translate(ctx, []string{text}, targetTag, nil)
	if err != nil {
		return "", err
	}

	return translations[0].Text, nil
}

// --------------------------------------------------------Usecase-4: Language Translation-------------------------------------------------------------------------

// --------------------------------------------------------End of Usecase-3: Stock analytics-------------------------------------------------------------------------
// ----------------------------------------- Main Function ---------------------------------------------------------------------------------------------
func main() {
	godotenv.Load(".env")
	alphaVantageAPIKey = os.Getenv("ALPHA_VANTAGE_API_KEY")
	googleAPIKey = os.Getenv("GOOGLE_API_KEY")
	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN"))
	client := witai.NewClient(os.Getenv("WIT_AI_TOKEN"))
	wolframClient = &wolfram.Client{AppID: os.Getenv("WOLFRAM_APP_ID")}
	go printCommandEvents(bot.CommandEvents())

	// -----------------------------------------------*-------------------------- Usecase-1: USER ASSISTANCE ------------------------------------------------------------{   USECASE--1  }--------
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

	//  ----------------------------------------------------------------------------- Usecase-2: TASK SCHEDULING ----------------------------------------------------{   USECASE--2  }-------
	// ---------------------------------
	// SCHEDULING EVENTS
	// ---------------------------------
	bot.Command("schedule <event> at <time>", &slacker.CommandDefinition{
		Description: "Schedule a meeting or event",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			// Get the user ID who triggered the command
			//userID := botCtx.Event().UserID
			userName := botCtx.Event().UserProfile.RealName
			// Extract parameters from the command
			event := request.Param("event")
			timeStr := request.Param("time")
			scheduleTime, err := time.Parse("15:04", timeStr)
			if err != nil {
				response.Reply("Invalid time format. Please use HH:mm format.")
				return
			}

			// scheduledEvent := scheduleEvent(message, scheduleTime, userID)
			pollMessage := fmt.Sprintf("\t--------------New Event Scheduled!!------------------:\n\n Event 🗒️:  %s \n Time ⌚: %s \n👤set by: @%s", event, scheduleTime.Format("15:04"), userName)

			// Log the poll message
			log.Println(pollMessage)

			// Send the poll message to a specific channel
			destinationChannel := "C068749HEGG" // Replace with your destination channel ID
			sendPollResult(destinationChannel, pollMessage)

			// Respond to the original channel
			response.Reply("Event Scheduled successfully! at <!channel> #scheduled-meetings")

		},
	})
	// ---------------------------------
	// SETTING REMINDERS
	// ---------------------------------
	bot.Command("setrem <message> at <time>", &slacker.CommandDefinition{
		Description: "Set a reminder",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {

			message := request.Param("message")
			// Get the user ID who triggered the command
			//userID := botCtx.Event().UserID
			userName := botCtx.Event().UserProfile.RealName

			// Extract parameters from the command
			timeStr := request.Param("time")
			scheduleTime, err := time.Parse("15:04", timeStr)
			if err != nil {
				response.Reply("Invalid time format. Please use HH:mm format.")
				return
			}
			// scheduledEvent := scheduleEvent(message, scheduleTime, userID)
			pollMessage := fmt.Sprintf("\t--------------New Reminder Set!!------------------:\n\n Reminder 📝:  %s \n Time ⏰: %s \n👤set by: @%s", message, scheduleTime.Format("15:04"), userName)
			// Log the poll message
			log.Println(pollMessage)

			// Send the poll message to a specific channel
			destinationChannel := "C067HSS8TAP" // Replace with your destination channel ID
			sendPollResult(destinationChannel, pollMessage)

			// Respond to the original channel
			response.Reply("Reminder set successfully! at <!channel> #set-reminders")
		},
	})
	// ---------------------------------
	// CREATING POLLS
	// ---------------------------------
	bot.Command("createpoll <question> options <options...>", &slacker.CommandDefinition{
		Description: "Create a poll",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			// Get parameters from the command
			question := request.Param("question")
			optionsStr := request.Param("options...")
			options := strings.Split(optionsStr, ",")

			if len(options) < 2 {
				response.Reply("Please provide at least two options for the poll.")
				return
			}

			// Build the poll message
			pollMessage := fmt.Sprintf("\t--------------New Poll------------------:\n\n Question:  %s\nOptions please react with the particular emojis to answer:\n\n", question)
			for i, option := range options {
				pollMessage += fmt.Sprintf(" %d. %s\n", i+1, option)
			}

			// Log the poll message
			log.Println(pollMessage)

			// Send the poll message to a specific channel
			destinationChannel := "C0687HA2RA4" // Replace with your destination channel ID
			sendPollResult(destinationChannel, pollMessage)

			// Respond to the original channel
			response.Reply("Poll created successfully! at #polls channel")

		},
	})
	//  ----------------------------------------------------------------------------- Usecase-3: STOCK ANALYTICS --------------------------------------------------------{   USECASE--3  }-----
	bot.Command("stock <symbol>", &slacker.CommandDefinition{
		Description: "Get top 3 data points for a stock",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			symbol := strings.ToUpper(request.Param("symbol"))

			// Get top 3 data points for the stock
			stockQuoteMessage, err := getStockQuote(symbol)

			if err != nil {
				log.Printf("Error getting stock quote: %v", err)
				response.Reply("Error getting stock quote.")
				return
			}

			// Reply with the top 3 data points
			response.Reply(stockQuoteMessage)
		},
	})

	//  ----------------------------------------------------------------------------- Usecase-4: LANGUAGE TRANSLATION --------------------------------------------------------{   USECASE--4  }---------
	bot.Command("translate <targetLanguage> <text>", &slacker.CommandDefinition{
		Description: "Translate text to a target language",
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			targetLanguage := request.Param("targetLanguage")
			text := request.Param("text")

			// Initialize the Google Cloud Translation client
			client, err := initTranslationClient()
			if err != nil {
				response.Reply("Failed to initialize the Translation client.")
				return
			}

			// Parse the target language code
			targetTag, err := language.Parse(targetLanguage)
			if err != nil {
				response.Reply("Invalid target language code.")
				return
			}

			// Translate the text
			translatedText, err := translateText(client, targetTag, text)
			if err != nil {
				response.Reply("Translation error.")
				return
			}

			// Reply with the translated text
			response.Reply(fmt.Sprintf("Original Text: %s\nTranslated Text: %s", text, translatedText))
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

//use case5 has directly been implemented in Slack under onboarding and faq section
//THANK YOU
