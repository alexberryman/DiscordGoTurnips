package main

import (
	"DiscordGoTurnips/internal/turnips"
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Variables used for command line parameters
var (
	Token       string
	DatabaseUrl string
)

var db *sql.DB

func init() {
	Token = os.Getenv("DISCORD_TOKEN")
	if Token == "" {
		log.Fatal("DISCORD_TOKEN must be set")
	}

	DatabaseUrl = os.Getenv("DATABASE_URL")
	if DatabaseUrl == "" {
		log.Fatal("DatabaseUrl must be set")
	}

	dbConnection, err := sql.Open("postgres", DatabaseUrl)
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	db = dbConnection
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("error creating Discord session,", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
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
	if err != nil {
		log.Println("Failed to replace mentions:", err)
		return
	}

	botMentionToken := fmt.Sprintf("@%s", botName)
	if strings.HasPrefix(tokenizedContent, botMentionToken) {
		input := strings.TrimSpace(strings.Replace(tokenizedContent, botMentionToken, "", 1))
		q := turnips.New(db)
		ctx := context.Background()
		var response string
		reactionEmoji := "❌"
		existingUserCount, err := q.CountUsersByDiscordId(ctx, m.Author.ID)
		if err != nil {
			log.Println(err)
			response = "Nice work! You broke the one thing that made people happy."
			reactionEmoji = "🔥"
			flushEmojiAndResponseToDiscord(s, m, reactionEmoji, response)
			return
		}

		existingServerContextCount, err := q.CountServerContextByDiscordId(ctx, turnips.CountServerContextByDiscordIdParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
		})
		if err != nil {
			log.Println(err)
			response = "Nice work! You broke the one thing that made people happy."
			reactionEmoji = "🔥"
			flushEmojiAndResponseToDiscord(s, m, reactionEmoji, response)
			return
		}

		user := getOrCreateUser(s, m, existingUserCount, existingServerContextCount, q, ctx)

		const CmdHistory = "history"
		const CmdTimeZone = "timezone"
		const CmdUpdate = "update"
		if turnipPrice, err := strconv.Atoi(input); err == nil {
			reactionEmoji, response = persistTurnipPrice(q, ctx, user, turnipPrice, reactionEmoji, response)
		} else if strings.Contains(input, CmdHistory) {
			historyInput := strings.TrimSpace(strings.Replace(input, CmdHistory, "", 1))
			if historyInput == "" {
				reactionEmoji, response = fetchUserPriceHistory(q, ctx, user, reactionEmoji, response)
			} else if historyInput == "all" {
				reactionEmoji, response = fetchServersPriceHistory(q, ctx, m, s)
			} else {
				reactionEmoji = "⛔"
				response = "That is not a valid history request"
			}

		} else if strings.Contains(input, CmdUpdate) {
			updateInput := strings.TrimSpace(strings.Replace(input, CmdUpdate, "", 1))
			if updateTurnipPrice, err := strconv.Atoi(updateInput); err == nil {
				reactionEmoji, response = updateExistingTurnipPrice(q, ctx, user, updateTurnipPrice, reactionEmoji, response)
			} else {
				reactionEmoji = "⛔"
				response = "That is not a valid price"
			}

		} else if strings.HasPrefix(input, CmdTimeZone) {
			reactionEmoji, response = updateUsersTimeZone(input, CmdTimeZone, reactionEmoji, response, q, ctx, user)
		} else if strings.HasPrefix(input, "help") {
			reactionEmoji, response = fetchHelpResponse(response, botMentionToken, CmdHistory, CmdTimeZone, reactionEmoji)

		} else {
			response = "Wut?"
		}

		flushEmojiAndResponseToDiscord(s, m, reactionEmoji, response)
	}
}

func flushEmojiAndResponseToDiscord(s *discordgo.Session, m *discordgo.MessageCreate, reactionEmoji string, response string) {
	reactToMessage(s, m, reactionEmoji)
	respondAsNewMessage(s, m, response)
}

func respondAsNewMessage(s *discordgo.Session, m *discordgo.MessageCreate, response string) {
	_, err := s.ChannelMessageSend(m.ChannelID, response)
	if err != nil {
		log.Println("Error responding:", err)
	}
}

func reactToMessage(s *discordgo.Session, m *discordgo.MessageCreate, reactionEmoji string) {
	err := s.MessageReactionAdd(m.ChannelID, m.Message.ID, reactionEmoji)
	if err != nil {
		log.Println("Error adding and emoji:", err)
	}
}

func fetchHelpResponse(response string, botMentionToken string, CmdHistory string, CmdTimeZone string, reactionEmoji string) (string, string) {
	response = fmt.Sprintf("`%s` - register a price for your the current time (defult timezone America/Chicago). Only one is allowed morning/afternoon each day\n"+
		"`%s` - update existing reported price\n"+
		"`%s` - get the your price history for the week\n"+
		"`%s` - set a timezone for yourself from https://en.wikipedia.org/wiki/List_of_tz_database_time_zones\n",
		fmt.Sprintf("%s 119", botMentionToken),
		fmt.Sprintf("%s update 110", botMentionToken),
		fmt.Sprintf("%s %s", botMentionToken, CmdHistory),
		fmt.Sprintf("%s %s America/New_York", botMentionToken, CmdTimeZone),
	)

	reactionEmoji = "💁"
	return reactionEmoji, response
}

func updateUsersTimeZone(input string, CmdTimeZone string, reactionEmoji string, response string, q *turnips.Queries, ctx context.Context, user turnips.User) (string, string) {
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
	return reactionEmoji, response
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

func fetchServersPriceHistory(q *turnips.Queries, ctx context.Context, m *discordgo.MessageCreate, s *discordgo.Session) (string, string) {
	var reactionEmoji string
	var response string

	prices, err := q.GetWeeksPriceHistoryByServer(ctx, m.GuildID)

	if err != nil {
		reactionEmoji = "⛔"
		response = fmt.Sprint(err)
	}
	response = ""
	priceMap := make(map[string][]int32)

	for _, value := range prices {
		if _, ok := priceMap[value.Username]; ok {
			//do something here
			priceMap[value.Username] = append(priceMap[value.Username], value.Price)
		} else {
			priceMap[value.Username] = []int32{value.Price}
		}
	}

	for username, prices := range priceMap {
		response += fmt.Sprintf("%s: %v\n", username, prices)
	}

	reactionEmoji = "✅"
	return reactionEmoji, response
}

func persistTurnipPrice(q *turnips.Queries, ctx context.Context, user turnips.User, turnipPrice int, reactionEmoji string, response string) (string, string) {

	err, reactionEmoji, response, turnipPriceObj := buildPriceObjFromInput(user, turnipPrice)
	if err != nil {
		return reactionEmoji, response
	}

	priceParams := turnips.CreatePriceParams{
		DiscordID: user.DiscordID,
		Price:     turnipPriceObj.Price,
		Meridiem:  turnipPriceObj.Meridiem,
		DayOfWeek: turnipPriceObj.DayOfWeek,
		DayOfYear: turnipPriceObj.DayOfYear,
		Year:      turnipPriceObj.Year,
	}

	_, err = q.CreatePrice(ctx, priceParams)

	if err != nil {
		reactionEmoji = "⛔"
		response = "You already created a price for this period"
	} else {
		reactionEmoji, response = turnipPriceColorfulResponse(reactionEmoji, turnipPrice, response)
	}
	return reactionEmoji, response
}

func updateExistingTurnipPrice(q *turnips.Queries, ctx context.Context, user turnips.User, turnipPrice int, reactionEmoji string, response string) (string, string) {

	err, reactionEmoji, response, turnipPriceObj := buildPriceObjFromInput(user, turnipPrice)
	if err != nil {
		return reactionEmoji, response
	}

	priceParams := turnips.UpdatePriceParams{
		DiscordID: user.DiscordID,
		Price:     turnipPriceObj.Price,
		Meridiem:  turnipPriceObj.Meridiem,
		DayOfWeek: turnipPriceObj.DayOfWeek,
		DayOfYear: turnipPriceObj.DayOfYear,
		Year:      turnipPriceObj.Year,
	}

	_, err = q.UpdatePrice(ctx, priceParams)

	if err != nil {
		reactionEmoji = "⛔"
		response = "I didn't find an existing price."
	} else {
		reactionEmoji, response = turnipPriceColorfulResponse(reactionEmoji, turnipPrice, response)
	}
	return reactionEmoji, response
}

func buildPriceObjFromInput(user turnips.User, turnipPrice int) (error, string, string, turnips.Price) {
	usersTimeZone, err := time.LoadLocation(user.TimeZone)
	var reactionEmoji string
	var response string
	if err != nil {
		reactionEmoji := "⛔"
		response := "Set a valid timezone from the `TZ database name` column https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"
		return err, reactionEmoji, response, turnips.Price{}
	}

	localTime := time.Now().In(usersTimeZone)
	var meridiem turnips.Meridiem
	switch fmt.Sprint(localTime.Format("pm")) {
	case "am":
		meridiem = turnips.MeridiemAm
	case "pm":
		meridiem = turnips.MeridiemPm
	}

	priceThing := turnips.Price{
		DiscordID: user.DiscordID,
		Price:     int32(turnipPrice),
		Meridiem:  meridiem,
		DayOfWeek: int32(localTime.Weekday()),
		DayOfYear: int32(localTime.YearDay()),
		Year:      int32(localTime.Year()),
	}
	priceThing.Meridiem = meridiem

	return err, reactionEmoji, response, priceThing
}

func getOrCreateUser(s *discordgo.Session, m *discordgo.MessageCreate, existingUserCount int64, existingServerContextCount int64, q *turnips.Queries, ctx context.Context) turnips.User {
	var user turnips.User
	var serverContext turnips.ServerContext
	if existingUserCount > 0 {
		user, _ = q.GetUsers(ctx, m.Author.ID)
		reactToMessage(s, m, "👤")
	} else {
		user, _ = q.CreateUser(ctx, m.Author.ID)
		reactToMessage(s, m, "🆕")
	}
	if existingServerContextCount > 0 {
		serverContext, _ = q.GetServerContext(ctx, turnips.GetServerContextParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
		})
		if serverContext.Username != m.Member.Nick {
			var err error
			serverContext, err = q.UpdateUsername(ctx, turnips.UpdateUsernameParams{
				DiscordID: m.Author.ID,
				Username:  m.Member.Nick,
				ServerID:  m.GuildID,
			})
			if err != nil {
				log.Println("Failed to update username")
			} else {
				reactToMessage(s, m, "🔁")
			}
		}

	} else {
		serverContext, _ = q.CreateServerContext(ctx, turnips.CreateServerContextParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
			Username:  m.Author.Username,
		})

		reactToMessage(s, m, "🆕")
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
