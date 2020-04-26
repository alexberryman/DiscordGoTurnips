package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
)

func helpResponse(s *discordgo.Session, m *discordgo.MessageCreate, botMentionToken string, CmdGraph string, CmdTimeZone string) {
	var response response
	response.Text = fmt.Sprintf("`%s` - register a price for your the current time (defult timezone America/Chicago). Only one is allowed morning/afternoon each day\n"+
		"`%s` - update existing reported price\n"+
		"`%s` - get the your price prediction graph for the week\n"+
		"`%s` - get the price prediction graphs for all users on the server for the week\n"+
		"`%s` - get the your price prediction graph for the last week (-2 for two weeks ago)\n"+
		"`%s` - get the price prediction graphs for all users on the server for the last week (-2 for two weeks ago)\n"+
		"`%s` - set yout local timezone <https://en.wikipedia.org/wiki/List_of_tz_database_time_zones>\n",
		fmt.Sprintf("%s 119", botMentionToken),
		fmt.Sprintf("%s update 110", botMentionToken),
		fmt.Sprintf("%s %s", botMentionToken, CmdGraph),
		fmt.Sprintf("%s %s all", botMentionToken, CmdGraph),
		fmt.Sprintf("%s %s -1", botMentionToken, CmdGraph),
		fmt.Sprintf("%s %s all -1", botMentionToken, CmdGraph),
		fmt.Sprintf("%s %s America/New_York", botMentionToken, CmdTimeZone),
	)

	response.Emoji = "💁"

	flushEmojiAndResponseToDiscord(s, m, response)
}
