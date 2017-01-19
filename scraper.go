package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ParseDocFunc func(doc *goquery.Document) (price string, date string, err error)

type Scraper struct {
	Name     string
	Disabled bool
	ParseDoc ParseDocFunc
}

type Scrapers map[string]*Scraper

// Result cointains the informations returned by a stock price scraper.
type Result struct {
	// name of the scraper that get the results
	ScraperName string
	// url of the html page
	URL string
	// stock identifier
	StockId string
	// timestamps
	TimeStart, TimeEnd time.Time

	StockPrice string
	StockDate  string
	Err        error
}

func (res *Result) String() string {
	return fmt.Sprintf(`Result{
	scraper:    %v,
	stockId:    %v,
	url:        %v,
	stockPrice: %v,
	stockDate:  %v,
	err:        %v,
	timeStart:  %v,
	timeEnd:    %v,
	elapsed:    %v,
}`, res.ScraperName, res.StockId, res.URL,
		res.StockPrice, res.StockDate, res.Err,
		res.TimeStart, res.TimeEnd, res.TimeEnd.Sub(res.TimeStart),
	)
}

func (scraper *Scraper) GetStockPrice(ctx context.Context, stockId, url string) *Result {
	// check scraper
	if scraper == nil {
		panic("GetStockPrice: scraper is nil")
	}
	if scraper.ParseDoc == nil {
		panic("GetStockPrice: scraper.ParseDocFunc is nil")
	}

	// init the result
	result := &Result{
		ScraperName: scraper.Name,
		URL:         url,
		StockId:     stockId,
		TimeStart:   time.Now(),
	}
	// use defer to set timeEnd
	defer func() { result.TimeEnd = time.Now() }()

	// return error in case of disabled scraper
	if scraper.Disabled {
		result.Err = errors.New("GetStockPrice: scraper is disabled")
		return result
	}

	// get the response
	resp, err := GetUrl(ctx, url)
	if err != nil {
		result.Err = err
		return result
	}

	// create goquery document
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		result.Err = err
		return result
	}
	// parse the response
	price, date, err := scraper.ParseDoc(doc)
	if err != nil {
		result.Err = err
		return result
	}

	result.StockPrice = price
	result.StockDate = date

	return result
}

func GetUrl(ctx context.Context, url string) (*http.Response, error) {

	type result struct {
		resp *http.Response
		err  error
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// make the request
		tr := &http.Transport{}
		client := &http.Client{Transport: tr}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		c := make(chan result, 1)

		go func() {
			resp, err := client.Do(req)
			c <- result{resp: resp, err: err}
		}()

		select {
		case <-ctx.Done():
			tr.CancelRequest(req)
			<-c // Wait for client.Do
			return nil, ctx.Err()
		case r := <-c:
			return r.resp, r.err
		}
	}
}
