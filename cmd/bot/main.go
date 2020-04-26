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

const CmdGraph = "graph"
const CmdTimeZone = "timezone"
const CmdUpdate = "update"

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
	var r response

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

		r.Emoji = "❌"
		existingAccount, err := q.CountAccountsByDiscordId(ctx, m.Author.ID)
		if err != nil {
			log.Println(err)
			r.Text = "Nice work! You broke the one thing that made people happy."
			r.Emoji = "🔥"
			flushEmojiAndResponseToDiscord(s, m, r)
			return
		}

		existingNickname, err := q.CountNicknameByDiscordId(ctx, turnips.CountNicknameByDiscordIdParams{
			DiscordID: m.Author.ID,
			ServerID:  m.GuildID,
		})
		if err != nil {
			log.Println(err)
			r.Text = "Nice work! You broke the one thing that made people happy."
			r.Emoji = "🔥"
			flushEmojiAndResponseToDiscord(s, m, r)
			return
		}

		account := getOrCreateAccount(s, m, existingAccount, existingNickname, q, ctx)

		if turnipPrice, err := strconv.Atoi(input); err == nil {
			persistTurnipPrice(ctx, m, s, account, turnipPrice)
		} else if strings.Contains(input, CmdGraph) {
			historyInput := strings.TrimSpace(strings.Replace(input, CmdGraph, "", 1))
			if historyInput == "" {
				linkUsersCurrentPrices(s, m, AcTurnipsChartLink)
			} else if historyInput == "all" {
				linkServersCurrentPrices(s, m, AcTurnipsChartLink)
			} else if offset, err := strconv.Atoi(historyInput); err == nil {
				linkAccountsPreviousPrices(m, s, offset*(-1), AcTurnipsChartLink)
			} else if strings.HasPrefix(historyInput, "all") {
				historicalServerInput := strings.TrimSpace(strings.Replace(historyInput, "all", "", 1))
				if offset, err := strconv.Atoi(historicalServerInput); err == nil {
					linkServersPreviousPrices(m, s, offset*(-1), AcTurnipsChartLink)
				} else {
					r.Text = "That isn't a valid week offset. Use -1, -2, -3 etc..."
					r.Emoji = "⏰"
					flushEmojiAndResponseToDiscord(s, m, r)
					return
				}
			} else {
				r.Emoji = "⛔"
				r.Text = "That is not a valid history request"
				flushEmojiAndResponseToDiscord(s, m, r)
				return
			}

		} else if strings.Contains(input, CmdUpdate) {
			updateInput := strings.TrimSpace(strings.Replace(input, CmdUpdate, "", 1))
			if updateTurnipPrice, err := strconv.Atoi(updateInput); err == nil {
				updateExistingTurnipPrice(ctx, s, m, account, updateTurnipPrice)
			} else {
				r.Emoji = "⛔"
				r.Text = "That is not a valid price"
				flushEmojiAndResponseToDiscord(s, m, r)
				return
			}

		} else if strings.HasPrefix(input, CmdTimeZone) {
			updateAccountTimeZone(ctx, input, CmdTimeZone, s, m, q, account)
		} else if strings.HasPrefix(input, "help") {
			helpResponse(s, m, botMentionToken, CmdGraph, CmdTimeZone)
		} else {
			r.Text = "Wut?"
			flushEmojiAndResponseToDiscord(s, m, r)
			return
		}
	}
}

func flushEmojiAndResponseToDiscord(s *discordgo.Session, m *discordgo.MessageCreate, r response) {
	reactToMessage(s, m, r.Emoji)
	respondAsNewMessage(s, m, r.Text)
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
