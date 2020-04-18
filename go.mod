module DiscordGoTurnips

go 1.14

// Comment below is needed for heroku-buildpack-go https://github.com/heroku/heroku-buildpack-go/issues/301

// +heroku goVersion go1.14

require (
	github.com/bwmarrin/discordgo v0.20.3
	github.com/lib/pq v1.3.0
	github.com/rubenv/sql-migrate v0.0.0-20200402132117-435005d389bc
)
