package main

import (
	turnips "DiscordGoTurnips/internal/turnips/generated-code"
	"context"
	"database/sql"
	"fmt"
	"github.com/go-echarts/go-echarts/charts"
	_ "github.com/lib/pq"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
)

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

var DatabaseUrl string
var db *sql.DB
var port string

func init() {
	DatabaseUrl = os.Getenv("DATABASE_URL")
	if DatabaseUrl == "" {
		log.Fatal("DatabaseUrl must be set")
	}

	dbConnection, err := sql.Open("postgres", DatabaseUrl)
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	db = dbConnection

	port = os.Getenv("PORT")
	if port == "" {
		log.Fatal("port is not set")
	}
}

type dailyPrice struct {
	DayOfWeek      int
	MorningPrice   int32
	AfternoonPrice int32
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("inside handle for: r", r.URL.RequestURI())
	periods := []string{"Mon am", "Mon pm", "Tue am", "Tue pm", "Wed am", "Wed pm", "Thu am", "Thu pm", "Fri am", "Sat pm", "Mon am", "Sat pm"}
	serverID := html.EscapeString(strings.TrimLeft(r.URL.RequestURI(), "/"))

	q := turnips.New(db)
	ctx := context.Background()
	prices, err := q.GetWeeksPriceHistoryByServer(ctx, serverID)
	log.Printf(fmt.Sprintf("found %d prices for %s", len(prices), serverID))
	if err != nil {
		log.Println("error fetching prices: ", err)
		w.WriteHeader(500)
	}

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

	renderBar(w, periods, serverID, priceMap)
	renderSmoothLine(w, periods, serverID, priceMap)
	renderSharpLine(w, periods, serverID, priceMap)

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

func renderBar(w http.ResponseWriter, periods []string, serverID string, priceMap map[string]map[string]dailyPrice) {
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.TitleOpts{Title: "Turnip Prices"})
	bar.AddXAxis(periods)

	for nickname, prices := range priceMap {
		var p []int32
		for _, d := range dayRange(Monday, Saturday) {
			p = append(p, prices[fmt.Sprint(d)].MorningPrice)
			p = append(p, prices[fmt.Sprint(d)].AfternoonPrice)
		}
		bar.AddYAxis(nickname, p)
	}

	f, err := os.Create(fmt.Sprintf("%s.html", serverID))
	if err != nil {
		log.Println(err)
	}
	_ = bar.Render(w, f)
}

func renderSmoothLine(w http.ResponseWriter, periods []string, serverID string, priceMap map[string]map[string]dailyPrice) {
	line := charts.NewLine()
	line.SetGlobalOptions(charts.TitleOpts{Title: "Turnip Prices"})
	line.AddXAxis(periods)

	for nickname, prices := range priceMap {
		var p []int32
		for _, d := range dayRange(Monday, Saturday) {
			p = append(p, prices[fmt.Sprint(d)].MorningPrice)
			p = append(p, prices[fmt.Sprint(d)].AfternoonPrice)
		}
		line.AddYAxis(nickname, p, charts.LineOpts{Smooth: true})
	}

	f, err := os.Create(fmt.Sprintf("%s.html", serverID))
	if err != nil {
		log.Println(err)
	}
	_ = line.Render(w, f)
}

func renderSharpLine(w http.ResponseWriter, periods []string, serverID string, priceMap map[string]map[string]dailyPrice) {
	line := charts.NewLine()
	line.SetGlobalOptions(charts.TitleOpts{Title: "Turnip Prices"})
	line.AddXAxis(periods)

	for nickname, prices := range priceMap {
		var p []int32
		for _, d := range dayRange(Monday, Saturday) {
			p = append(p, prices[fmt.Sprint(d)].MorningPrice)
			p = append(p, prices[fmt.Sprint(d)].AfternoonPrice)
		}
		line.AddYAxis(nickname, p)
	}

	f, err := os.Create(fmt.Sprintf("%s.html", serverID))
	if err != nil {
		log.Println(err)
	}
	_ = line.Render(w, f)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s\n", r.RequestURI)
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	fmt.Fprintln(w, "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQEAYAAABPYyMiAAAABmJLR0T///////8JWPfcAAAACXBIWXMAAABIAAAASABGyWs+AAAAF0lEQVRIx2NgGAWjYBSMglEwCkbBSAcACBAAAeaR9cIAAAAASUVORK5CYII=\n")
}

func main() {
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/", handler)
	log.Println("Run server at " + os.Getenv("CUSTOM_DOMAIN") + ":" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
