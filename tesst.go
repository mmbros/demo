package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func parseTestSource(doc *goquery.Document) (price string, date string, err error) {

	doc.Find("ul > li > b").Each(func(i int, s *goquery.Selection) {
		switch i {
		case 0:
			price = s.Text()
		case 1:
			date = s.Text()
		}
	})
	if price == "" {
		err = errors.New("Price not found")
	}
	return
}

func handlerTestStockServer(w http.ResponseWriter, r *http.Request) {

	a, b := 100, 3000
	msec := a + rand.Intn(b-a)
	fmt.Printf("msec = %+v\n", msec)

	time.Sleep(time.Duration(msec) * time.Millisecond)

	price := msec
	date := time.Now()
	fmt.Fprintf(w, "<ul>\n  <li>Price: <b>%d</b></li>\n  <li>Date: <b>%s</b></li>\n</ul>", price, date)
}

func initTestStockServer() *httptest.Server {

	rand.Seed(time.Now().UTC().UnixNano())

	return httptest.NewServer(http.HandlerFunc(handlerTestStockServer))
}

func getScraperName(n int) string {
	name := fmt.Sprintf("scraper_%d", n)
	return name
}
func getStockName(n int) string {
	name := fmt.Sprintf("stock_%d", n)
	return name
}

func initScrapersConfig(numScrapers int) []*ScraperConfig {
	// init scraper config
	res := []*ScraperConfig{}
	for j := 0; j < numScrapers; j++ {
		name := getScraperName(j)
		res = append(res, &ScraperConfig{
			Name:     name,
			Disabled: false,
			Workers:  1,
		})
	}
	return res
}

func initScrapers(numScrapers int) Scrapers {
	scrapers := map[string]*Scraper{}
	for j := 0; j < numScrapers; j++ {
		name := getScraperName(j)
		scrapers[name] = &Scraper{
			Name:     name,
			ParseDoc: parseTestSource,
		}
	}
	return scrapers
}

func initTestStocks(numStocks, numScrapers int, url string) Stocks {
	stocks := Stocks{}

	newSpi := func(numScraper, numStock int) StockPriceSource {
		var spi StockPriceSource
		spi.ScraperName = getScraperName(numScraper)
		spi.URL = fmt.Sprintf("%s/%s/%s", url, spi.ScraperName, getStockName(numStock))
		return spi
	}

	for j := 0; j < numStocks; j++ {
		suffix := strconv.Itoa(j)
		stock := &Stock{
			Name:    "name" + suffix,
			Isin:    "isin" + suffix,
			ID:      "id" + suffix,
			Sources: []StockPriceSource{},
		}
		for n := 0; n < numScrapers; n++ {
			stock.Sources = append(stock.Sources, newSpi(n, j))
		}
		stocks[stock.ID] = stock
	}
	return stocks
}
