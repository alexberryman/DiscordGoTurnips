package main

import (
	"DiscordGoTurnips/internal/turnips/generated-code"
	"context"
	"fmt"
	"time"
)

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
