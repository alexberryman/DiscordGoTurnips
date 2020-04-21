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

		const CmdGraph = "graph"
		const CmdTimeZone = "timezone"
		const CmdUpdate = "update"
		if turnipPrice, err := strconv.Atoi(input); err == nil {
			reactionEmoji, response = persistTurnipPrice(q, ctx, account, turnipPrice, reactionEmoji, response)
		} else if strings.Contains(input, CmdGraph) {
			historyInput := strings.TrimSpace(strings.Replace(input, CmdGraph, "", 1))
			if historyInput == "" {
				reactionEmoji, response = linkUsersCurrentPrices(m)
			} else if historyInput == "all" {
				reactionEmoji, response = linkServersCurrentPrices(m)
			} else if offset, err := strconv.Atoi(historyInput); err == nil {
				reactionEmoji, response = linkAccountsPreviousPrices(m, offset*(-1))
			} else if historyInput == "all -1" {
				historicalServerInput := strings.TrimSpace(strings.Replace(historyInput, "all", "", 1))
				if offset, err := strconv.Atoi(historicalServerInput); err == nil {
					reactionEmoji, response = linkServersPreviousPrices(m, offset*(-1))
				}
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
			reactionEmoji, response = fetchHelpResponse(response, botMentionToken, CmdGraph, CmdTimeZone, reactionEmoji)

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
