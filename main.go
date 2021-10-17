package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TheRicci/taskcreator/tools"
	"github.com/bwmarrin/discordgo"
	thousands "github.com/floscodes/golang-thousands"
	"github.com/joho/godotenv"
	cmc "github.com/miguelmota/go-coinmarketcap/pro/v1"
)

type Infos struct {
	Price     interface{}
	Pricebtc  float64
	Rank      float64
	Vol24     float64
	Change24  interface{}
	Change1hr float64
	Changebtc float64
}

var infos Infos

var arrow, arrow2, arrow3 string

func init() {
	_ = godotenv.Load(".env")
}

func main() {

	dg, err := discordgo.New("Bot " + os.Getenv("BOTKEY"))
	err = dg.Open()

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	tools.NewTask(1*time.Second, 360*time.Second).Start(func() {
		client := cmc.NewClient(&cmc.Config{
			ProAPIKey: os.Getenv("CMCKEY"),
		})

		quotes, err := client.Cryptocurrency.LatestQuotes(&cmc.QuoteOptions{
			Symbol:  "MIOTA",
			Convert: "USD",
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		//println(quotes)

		for _, quote := range quotes {

			infos.Change1hr = quote.Quote["USD"].PercentChange1H
			infos.Rank = quote.CMCRank

			if infos.Change1hr > 0 {
				arrow2 = "➚"
			} else {
				arrow2 = "➘"
			}

		}
	}, false)

	tools.NewTask(1*time.Second, 5*time.Second).Start(func() {

		resp, err := http.Get("https://api.coingecko.com/api/v3/simple/price?ids=iota&vs_currencies=usd&include_24hr_vol=true&include_24hr_change=true")

		if err != nil {
			log.Printf("Request Failed: %s", err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Reading body failed: %s", err)
			return
		} // Log the request body

		var m map[string]interface{}

		err = json.Unmarshal(body, &m)

		if err != nil {
			log.Printf("Unmarshal Failed: %s", err)
			return
		}

		resp.Body.Close()
		infos.Change24 = m["iota"].(map[string]interface{})["usd_24h_change"]
		infos.Vol24, _ = strconv.ParseFloat(fmt.Sprintf("%v", m["iota"].(map[string]interface{})["usd_24h_vol"]), 64)

		resp, err = http.Get("https://api.coingecko.com/api/v3/simple/price?ids=iota&vs_currencies=btc&include_24hr_change=true")

		if err != nil {
			log.Printf("Request Failed: %s", err)
			return
		}

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Reading body failed: %s", err)
			return
		} // Log the request body

		var m2 map[string]interface{}

		err = json.Unmarshal(body, &m2)

		if err != nil {
			log.Printf("Unmarshal failed: %s", err)
			return
		}

		resp.Body.Close()
		infos.Pricebtc, _ = strconv.ParseFloat(fmt.Sprintf("%v", m2["iota"].(map[string]interface{})["btc"]), 64)

		if infos.Changebtc, _ = strconv.ParseFloat(fmt.Sprintf("%v", m2["iota"].(map[string]interface{})["btc_24h_change"]), 64); infos.Changebtc > 0 {
			arrow3 = "➚"
		} else {
			arrow3 = "➘"
		}

		var a int = 0
		var StatusData discordgo.UpdateStatusData

		vol, _ := thousands.Separate(int(infos.Vol24), "en")

		if change24, _ := strconv.ParseFloat(fmt.Sprintf("%v", infos.Change24), 64); change24 > 0 {
			arrow = "➚"
			StatusData = discordgo.UpdateStatusData{
				IdleSince: &a,
				Activities: []*discordgo.Activity{
					&discordgo.Activity{
						Name: fmt.Sprintf("$ 24h: %.3f%%%s, 1h: %.3f%%%s, btc: %f%s, 24vol: $%s ", change24, arrow, infos.Change1hr, arrow2, infos.Pricebtc, arrow3, vol),
						Type: discordgo.ActivityTypeWatching,
						URL:  "",
					},
				},
				AFK:    false,
				Status: "online",
			}

		} else {
			arrow = "➘"
			StatusData = discordgo.UpdateStatusData{
				IdleSince: &a,
				Activities: []*discordgo.Activity{
					&discordgo.Activity{
						Name: fmt.Sprintf("$ 24h: %.3f%%%s, 1h: %.3f%%%s, btc: %f%s, 24vol: $%s ", change24, arrow, infos.Change1hr, arrow2, infos.Pricebtc, arrow3, vol),
						Type: discordgo.ActivityTypeWatching,
						URL:  "",
					},
				},
				AFK:    false,
				Status: "dnd",
			}
		}

		err = dg.UpdateStatusComplex(StatusData)
		if err != nil {
			log.Printf("Status update failed: %s", err)
			return
		}

	}, false)

	tools.NewTask(1*time.Second, 6*time.Second).Start(func() {

		resp, err := http.Get("https://api.binance.com/api/v1/ticker/price?symbol=IOTAUSDT")

		if err != nil {
			log.Printf("Request Failed: %s", err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Reading body failed: %s", err)
			return
		} // Log the request body

		var m2 map[string]interface{}

		err = json.Unmarshal(body, &m2)

		if err != nil {
			log.Printf("Unmarshal failed: %s", err)
			return
		}
		infos.Price = m2["price"]

		price, _ := strconv.ParseFloat(fmt.Sprintf("%s", infos.Price), 64)
		err = dg.GuildMemberNickname(os.Getenv("GUILDID"), "@me", fmt.Sprintf("$%.4f%s #%v", price, arrow, infos.Rank))

		if err != nil {
			log.Printf("Nickname update failed: %s", err)
			return
		}

	}, false)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}
