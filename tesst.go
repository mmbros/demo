package main

import (
	"errors"
	"fmt"
	"log"
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

	a, b := 1000, 1000
	msec := a
	if a != b {
		msec += rand.Intn(b - a)
	}
	//fmt.Printf("msec = %+v\n", msec)

	time.Sleep(time.Duration(msec) * time.Millisecond)

	if 1+rand.Intn(10) <= 6 {
		log.Println("SERVER ERROR 500 - " + r.URL.Path)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	price := msec
	date := time.Now()
	fmt.Fprintf(w, "<ul>\n  <li>Price: <b>%d</b></li>\n  <li>Date: <b>%s</b></li>\n</ul>", price, date)
}

func initTestStockServer() *httptest.Server {

	rand.Seed(time.Now().UTC().UnixNano())

	return httptest.NewServer(http.HandlerFunc(handlerTestStockServer))
}

func getScraperName(n int) string {
	name := fmt.Sprintf("scraper_%d", n+1)
	return name
}
func getStockName(n int) string {
	name := fmt.Sprintf("stock_%d", n+1)
	return name
}

func initScrapers(numScrapers int) Scrapers {
	scrapers := map[string]*Scraper{}
	for j := 0; j < numScrapers; j++ {
		name := getScraperName(j)
		scrapers[name] = &Scraper{
			Name:     name,
			Disabled: false,
			Workers:  1,
			parseDoc: parseTestSource,
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
		suffix := strconv.Itoa(j + 1)
		stock := &Stock{
			Name:    "name" + suffix,
			Isin:    "isin" + suffix,
			ID:      "id_" + suffix,
			Sources: []StockPriceSource{},
		}
		for n := 0; n < numScrapers; n++ {
			ns := (n + j) % numScrapers
			stock.Sources = append(stock.Sources, newSpi(ns, j))
		}
		stocks[stock.ID] = stock
	}
	return stocks
}
