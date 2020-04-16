package main

import (
	"DisGoNips/internal/turnips"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Variables used for command line parameters
var (
	Token string
)

var db *sql.DB

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()

	pgUser := os.Getenv("POSTGRES_USER")
	pgHost := os.Getenv("POSTGRES_HOST")
	pgPort := os.Getenv("POSTGRES_PORT")
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")

	source := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, pgDB)

	dbConnection, err := sql.Open("postgres", source)
	if err != nil {
		fmt.Println("Cannot connect to database:")
		fmt.Println(err)
		os.Exit(137)
	}

	db = dbConnection
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	botName := s.State.User.Username
	if m.Author.ID == s.State.User.ID {
		return
	}

	tokenizedContent, err := m.ContentWithMoreMentionsReplaced(s)
	botMentionToken := fmt.Sprintf("@%s", botName)
	if strings.HasPrefix(tokenizedContent, botMentionToken) {
		input := strings.TrimSpace(strings.Replace(tokenizedContent, botMentionToken, "", 1))
		q := turnips.New(db)
		ctx := context.Background()

		existingUserCount, _ := q.CountUsersByDiscordId(ctx, m.Author.ID)

		user := getOrCreateUser(s, m, existingUserCount, q, ctx)

		var response string
		reactionEmoji := "❌"
		const CmdHistory = "history"
		const CmdTimeZone = "timezone"
		if turnipPrice, err := strconv.Atoi(input); err == nil {
			reactionEmoji, response = persistTurnipPrice(q, ctx, user, turnipPrice, reactionEmoji, response)
		} else if strings.Contains(input, CmdHistory) {
			reactionEmoji, response = fetchUserPriceHistory(q, ctx, user, reactionEmoji, response)

		} else if strings.HasPrefix(input, CmdTimeZone) {
			timezoneInput := strings.TrimSpace(strings.Replace(input, CmdTimeZone, "", 1))
			_, err := time.LoadLocation(timezoneInput)
			if err != nil {
				reactionEmoji = "⛔"
				response = "Set a valid timezone from the `TZ database name` column https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"

			} else {
				reactionEmoji = "✅"
			}

			_, err = q.UpdateTimeZone(ctx, turnips.UpdateTimeZoneParams{
				DiscordID: user.DiscordID,
				TimeZone:  timezoneInput,
			})
		} else if strings.HasPrefix(input, "help") {
			response = fmt.Sprintf("`%s` - register a price for your the current time (defult timezone America/Chicago). Only one is allowed morning/afternoon each day\n"+
				"`%s` - get the your price history for the week\n"+
				"`%s` - set a timezone for yourself from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones\n",
				fmt.Sprintf("%s 119", botMentionToken),
				fmt.Sprintf("%s %s", botMentionToken, CmdHistory),
				fmt.Sprintf("%s %s America/New_York", botMentionToken, CmdTimeZone),
			)

		} else {
			response = "Wut?"
		}

		s.MessageReactionAdd(m.ChannelID, m.Message.ID, reactionEmoji)
		s.ChannelMessageSend(m.ChannelID, response)
	}

	if err != nil {
		fmt.Println(err)
	}
}

func fetchUserPriceHistory(q *turnips.Queries, ctx context.Context, user turnips.User, reactionEmoji string, response string) (string, string) {
	prices, err := q.GetWeeksPriceHistoryByUser(ctx, user.DiscordID)
	if err != nil {
		reactionEmoji = "⛔"
		response = fmt.Sprint(err)
	}

	var priceList []int32
	for _, price := range prices {
		priceList = append(priceList, price.Price)
	}

	response = fmt.Sprint(priceList)
	reactionEmoji = "✅"
	return reactionEmoji, response
}

func persistTurnipPrice(q *turnips.Queries, ctx context.Context, user turnips.User, turnipPrice int, reactionEmoji string, response string) (string, string) {
	usersTimeZone, err := time.LoadLocation(user.TimeZone)
	if err != nil {
		reactionEmoji = "⛔"
		response = "Set a valid timezone from the `TZ database name` column https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"
		return reactionEmoji, response
	}

	localTime := time.Now().In(usersTimeZone)
	var meridiem turnips.Meridiem
	switch fmt.Sprint(localTime.Format("pm")) {
	case "am":
		meridiem = turnips.MeridiemAm
	case "pm":
		meridiem = turnips.MeridiemPm
	}

	_, err = q.CreatePrice(ctx, turnips.CreatePriceParams{
		DiscordID: user.DiscordID,
		Price:     int32(turnipPrice),
		Meridiem:  meridiem,
		DayOfWeek: int32(localTime.Weekday()),
		DayOfYear: int32(localTime.YearDay()),
		Year:      int32(localTime.Year()),
	})

	if err != nil {
		reactionEmoji = "⛔"
		response = "You already created a price for this period"
	} else {
		reactionEmoji, response = turnipPriceColorfulResponse(reactionEmoji, turnipPrice, response)
	}
	return reactionEmoji, response
}

func getOrCreateUser(s *discordgo.Session, m *discordgo.MessageCreate, existingUserCount int64, q *turnips.Queries, ctx context.Context) turnips.User {
	var user turnips.User
	if existingUserCount > 0 {
		user, _ = q.GetUsers(ctx, m.Author.ID)
		fmt.Println("Found User", user)
		s.MessageReactionAdd(m.ChannelID, m.Message.ID, "👤")
	} else {
		user, _ = q.CreateUser(ctx, turnips.CreateUserParams{
			DiscordID: m.Author.ID,
			Username:  m.Author.Username,
		})
		s.MessageReactionAdd(m.ChannelID, m.Message.ID, "🆕")
		fmt.Println("Created User", user)
	}
	return user
}

func turnipPriceColorfulResponse(reactionEmoji string, turnipPrice int, response string) (string, string) {
	reactionEmoji = "✅"
	if turnipPrice == 69 {
		response = "nice."
	} else if turnipPrice > 0 && turnipPrice <= 100 {
		response = fmt.Sprintf("Oh, your turnips are selling for %d right now? Sucks to be poor!", turnipPrice)
	} else if turnipPrice > 0 && turnipPrice <= 149 {
		response = fmt.Sprintf("Oh, your turnips are selling for %d right now? Meh.", turnipPrice)
	} else if turnipPrice > 0 && turnipPrice <= 150 {
		response = fmt.Sprintf("Oh, your turnips are selling for %d right now? Decent!", turnipPrice)
	} else if turnipPrice > 0 && turnipPrice < 200 {
		response = fmt.Sprintf("Oh shit, your turnips are selling for %d right now? Dope!", turnipPrice)
	} else if turnipPrice > 200 {
		response = fmt.Sprintf("@everyone get in here! Someone has turnips trading for %d bells", turnipPrice)
	} else {
		response = "Is that even a real number?"
		reactionEmoji = "❌"
	}
	return reactionEmoji, response
}
