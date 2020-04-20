package main

import (
	"DiscordGoTurnips/internal/turnips/generated-code"
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

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

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
		existingAccount, err := q.CountAccountsByDiscordId(ctx, m.Author.ID)
		if err != nil {
			log.Println(err)
			response = "Nice work! You broke the one thing that made people happy."
			reactionEmoji = "🔥"
			flushEmojiAndResponseToDiscord(s, m, reactionEmoji, response)
			return
		}

		existingNickname, err := q.CountNicknameByDiscordId(ctx, turnips.CountNicknameByDiscordIdParams{
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

		account := getOrCreateAccount(s, m, existingAccount, existingNickname, q, ctx)

		const CmdHistory = "history"
		const CmdTimeZone = "timezone"
		const CmdUpdate = "update"
		if turnipPrice, err := strconv.Atoi(input); err == nil {
			reactionEmoji, response = persistTurnipPrice(q, ctx, account, turnipPrice, reactionEmoji, response)
		} else if strings.Contains(input, CmdHistory) {
			historyInput := strings.TrimSpace(strings.Replace(input, CmdHistory, "", 1))
			if historyInput == "" {
				reactionEmoji, response = fetchAccountPriceHistory(q, ctx, account, reactionEmoji, response)
			} else if historyInput == "all" {
				reactionEmoji, response = fetchServersPriceHistory(q, ctx, m)
			} else if historyInput == "chart" {
				reactionEmoji, response = linkServersCurrentPrices(m)
			} else if historyInput == "chart previous" {
				reactionEmoji, response = linkServersPreviousPrices(m)
			} else {
				reactionEmoji = "⛔"
				response = "That is not a valid history request"
			}

		} else if strings.Contains(input, CmdUpdate) {
			updateInput := strings.TrimSpace(strings.Replace(input, CmdUpdate, "", 1))
			if updateTurnipPrice, err := strconv.Atoi(updateInput); err == nil {
				reactionEmoji, response = updateExistingTurnipPrice(q, ctx, account, updateTurnipPrice, reactionEmoji, response)
			} else {
				reactionEmoji = "⛔"
				response = "That is not a valid price"
			}

		} else if strings.HasPrefix(input, CmdTimeZone) {
			reactionEmoji, response = updateAccountTimeZone(input, CmdTimeZone, reactionEmoji, response, q, ctx, account)
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
		"`%s` - set yout local timezone\n",
		fmt.Sprintf("%s 119", botMentionToken),
		fmt.Sprintf("%s update 110", botMentionToken),
		fmt.Sprintf("%s %s", botMentionToken, CmdHistory),
		fmt.Sprintf("%s %s America/New_York", botMentionToken, CmdTimeZone),
	)

	reactionEmoji = "💁"
	return reactionEmoji, response
}

func updateAccountTimeZone(input string, CmdTimeZone string, reactionEmoji string, response string, q *turnips.Queries, ctx context.Context, a turnips.Account) (string, string) {
	timezoneInput := strings.TrimSpace(strings.Replace(input, CmdTimeZone, "", 1))
	_, err := time.LoadLocation(timezoneInput)
	if err != nil {
		reactionEmoji = "⛔"
		response = "Set a valid timezone from the `TZ database name` column https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"

	} else {
		reactionEmoji = "✅"
	}

	_, err = q.UpdateTimeZone(ctx, turnips.UpdateTimeZoneParams{
		DiscordID: a.DiscordID,
		TimeZone:  timezoneInput,
	})
	return reactionEmoji, response
}

func fetchAccountPriceHistory(q *turnips.Queries, ctx context.Context, a turnips.Account, reactionEmoji string, response string) (string, string) {
	prices, err := q.GetWeeksPriceHistoryByAccount(ctx, a.DiscordID)
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

func fetchServersPriceHistory(q *turnips.Queries, ctx context.Context, m *discordgo.MessageCreate) (string, string) {
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
		if _, ok := priceMap[value.Nickname]; ok {
			//do something here
			priceMap[value.Nickname] = append(priceMap[value.Nickname], value.Price)
		} else {
			priceMap[value.Nickname] = []int32{value.Price}
		}
	}

	for nickname, prices := range priceMap {
		response += fmt.Sprintf("%s: %v\n", nickname, prices)
	}

	reactionEmoji = "✅"
	return reactionEmoji, response
}

type dailyPrice struct {
	DayOfWeek      int
	MorningPrice   int32
	AfternoonPrice int32
}

func linkServersCurrentPrices(m *discordgo.MessageCreate) (string, string) {
	var response string
	var reactionEmoji string
	q := turnips.New(db)
	ctx := context.Background()
	prices, err := q.GetWeeksPriceHistoryByServer(ctx, m.GuildID)
	log.Printf(fmt.Sprintf("found %d prices for %s", len(prices), m.GuildID))
	if err != nil {
		log.Println("error fetching prices: ", err)
	}

	return buildPriceLinks(prices, response, reactionEmoji)
}

func linkServersPreviousPrices(m *discordgo.MessageCreate) (string, string) {
	var response string
	var reactionEmoji string
	q := turnips.New(db)
	ctx := context.Background()
	prices, err := q.GetLastWeeksPriceHistoryByServer(ctx, m.GuildID)
	log.Printf(fmt.Sprintf("found %d prices for %s", len(prices), m.GuildID))
	if err != nil {
		log.Println("error fetching prices: ", err)
	}

	data := make([]turnips.GetWeeksPriceHistoryByServerRow, 0)
	for _, v := range prices {
		p := turnips.GetWeeksPriceHistoryByServerRow(v)
		data = append(data, p)
	}

	return buildPriceLinks(data, response, reactionEmoji)
}

func buildPriceLinks(prices []turnips.GetWeeksPriceHistoryByServerRow, response string, reactionEmoji string) (string, string) {
	priceMap := make(map[string]map[string]dailyPrice)

	for _, value := range prices {
		wp := getEmptyWeeklyPrices()
		if _, ok := priceMap[value.Nickname]; ok {
			updateMorningOrAfterNoonPrice(value, priceMap)
		} else {
			priceMap[value.Nickname] = wp
			updateMorningOrAfterNoonPrice(value, priceMap)
		}
	}

	turnipLink := make(map[string]string)
	for nickname, prices := range priceMap {
		for _, d := range dayRange(Monday, Saturday) {
			if _, ok := turnipLink[nickname]; !ok {
				turnipLink[nickname] = ""
			}

			if prices[fmt.Sprint(d)].MorningPrice != 0 {
				turnipLink[nickname] += fmt.Sprintf("-%d", prices[fmt.Sprint(d)].MorningPrice)
			} else {
				turnipLink[nickname] += "-"
			}
			if prices[fmt.Sprint(d)].AfternoonPrice != 0 {
				turnipLink[nickname] += fmt.Sprintf("-%d", prices[fmt.Sprint(d)].AfternoonPrice)
			} else {
				turnipLink[nickname] += "-"
			}
		}
		response += fmt.Sprintf("%s: https://ac-turnip.com/#%s\n", nickname, turnipLink[nickname])
	}

	reactionEmoji = "🔗"
	return reactionEmoji, response
}

func dayRange(min, max Weekday) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = int(min) + i
	}
	return a
}

func getEmptyWeeklyPrices() map[string]dailyPrice {
	w := newWeeklyPrices()

	for _, d := range dayRange(Monday, Saturday) {
		dp := dailyPrice{
			DayOfWeek:      d,
			MorningPrice:   0,
			AfternoonPrice: 0,
		}
		w[fmt.Sprintf("%d", d)] = dp
	}
	return w
}

func newWeeklyPrices() map[string]dailyPrice {
	w := make(map[string]dailyPrice)
	return w
}

func updateMorningOrAfterNoonPrice(value turnips.GetWeeksPriceHistoryByServerRow, priceMap map[string]map[string]dailyPrice) {
	if value.AmPm == turnips.AmPmAm {
		tempPrice := priceMap[value.Nickname][fmt.Sprint(value.DayOfWeek)]
		tempPrice.MorningPrice = value.Price
		priceMap[value.Nickname][fmt.Sprint(value.DayOfWeek)] = tempPrice
	} else {
		tempPrice := priceMap[value.Nickname][fmt.Sprint(value.DayOfWeek)]
		tempPrice.AfternoonPrice = value.Price
		priceMap[value.Nickname][fmt.Sprint(value.DayOfWeek)] = tempPrice
	}
}

func persistTurnipPrice(q *turnips.Queries, ctx context.Context, a turnips.Account, turnipPrice int, reactionEmoji string, response string) (string, string) {

	err, reactionEmoji, response, turnipPriceObj := buildPriceObjFromInput(a, turnipPrice)
	if err != nil {
		return reactionEmoji, response
	}

	priceParams := turnips.CreatePriceParams{
		DiscordID: a.DiscordID,
		Price:     turnipPriceObj.Price,
		AmPm:      turnipPriceObj.AmPm,
		DayOfWeek: turnipPriceObj.DayOfWeek,
		DayOfYear: turnipPriceObj.DayOfYear,
		Year:      turnipPriceObj.Year,
		Week:      turnipPriceObj.Week,
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

func updateExistingTurnipPrice(q *turnips.Queries, ctx context.Context, a turnips.Account, turnipPrice int, reactionEmoji string, response string) (string, string) {

	err, reactionEmoji, response, turnipPriceObj := buildPriceObjFromInput(a, turnipPrice)
	if err != nil {
		return reactionEmoji, response
	}

	priceParams := turnips.UpdatePriceParams{
		DiscordID: a.DiscordID,
		Price:     turnipPriceObj.Price,
		AmPm:      turnipPriceObj.AmPm,
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

func buildPriceObjFromInput(a turnips.Account, turnipPrice int) (error, string, string, turnips.TurnipPrice) {
	accountTimeZone, err := time.LoadLocation(a.TimeZone)
	var reactionEmoji string
	var response string
	if err != nil {
		reactionEmoji := "⛔"
		response := "Set a valid timezone from the `TZ database name` column https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"
		return err, reactionEmoji, response, turnips.TurnipPrice{}
	}

	localTime := time.Now().In(accountTimeZone)
	var meridiem turnips.AmPm
	switch fmt.Sprint(localTime.Format("pm")) {
	case "am":
		meridiem = turnips.AmPmAm
	case "pm":
		meridiem = turnips.AmPmPm
	}

	_, week := localTime.ISOWeek()
	priceThing := turnips.TurnipPrice{
		DiscordID: a.DiscordID,
		Price:     int32(turnipPrice),
		AmPm:      meridiem,
		DayOfWeek: int32(localTime.Weekday()),
		DayOfYear: int32(localTime.YearDay()),
		Year:      int32(localTime.Year()),
		Week:      int32(week),
	}
	priceThing.AmPm = meridiem

	return err, reactionEmoji, response, priceThing
}

func getOrCreateAccount(s *discordgo.Session, m *discordgo.MessageCreate, existingAccount int64, existingNickname int64, q *turnips.Queries, ctx context.Context) turnips.Account {
	var account turnips.Account
	var nickname turnips.Nickname
	if existingAccount > 0 {
		account, _ = q.GetAccount(ctx, m.Author.ID)
		reactToMessage(s, m, "👤")
	} else {
		account, _ = q.CreateAccount(ctx, m.Author.ID)
		reactToMessage(s, m, "🆕")
	}

	var name string
	if m.Member.Nick != "" {
		name = m.Member.Nick
	} else {
		name = m.Author.Username
	}

	if existingNickname > 0 {
		nickname, _ = q.GetNickname(ctx, turnips.GetNicknameParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
		})
		if nickname.Nickname != name {
			var err error
			nickname, err = q.UpdateNickname(ctx, turnips.UpdateNicknameParams{
				DiscordID: m.Author.ID,
				Nickname:  name,
				ServerID:  m.GuildID,
			})
			if err != nil {
				log.Println("Failed to update nickname")
			} else {
				reactToMessage(s, m, "🔁")
			}
		}

	} else {
		nickname, _ = q.CreateNickname(ctx, turnips.CreateNicknameParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
			Nickname:  name,
		})

		reactToMessage(s, m, "🆕")
	}
	return account
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
