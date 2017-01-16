/*
References:
- [Context and Cancellation of goroutines](http://dahernan.github.io/2015/02/04/context-and-cancellation-of-goroutines/)
- [Cancelation, Context, and Plumbing](https://talks.golang.org/2014/gotham-context.slide#1)
- [Go Concurrency Patterns: Context](https://blog.golang.org/context)
- [Go Concurrency Patterns: Pipelines and cancellation](https://blog.golang.org/pipelines)
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

type ParseDocFunc func(doc *goquery.Document) (price string, date string, err error)

type Scraper struct {
	Name     string
	Disabled bool
	ParseDoc ParseDocFunc
}

func (scraper *Scraper) GetStockPrice(ctx context.Context, stockId, url string) *Result {
	result := &Result{
		ScraperName: scraper.Name,
		URL:         url,
		StockId:     stockId,
		TimeStart:   time.Now(),
	}
	// use defer to set timeEnd
	defer func() { result.TimeEnd = time.Now() }()

	// error in case of disabled scraper
	if scraper.Disabled {
		result.Err = errors.New("Disabled scraper.")
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
func borsaitaliana(doc *goquery.Document) (price string, date string, err error) {

	doc.Find("div.l-box > div.l-box > span > strong").Each(func(i int, s *goquery.Selection) {
		switch i {
		case 0:
			price = s.Text()
		case 3:
			date = s.Text()
		}
	})
	if price == "" {
		err = errors.New("Price not found")
	}
	return
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	// http://dahernan.github.io/2015/02/04/context-and-cancellation-of-goroutines/

	headOrTails := rand.Intn(2)

	if headOrTails == 0 {
		time.Sleep(5 * time.Second)
		fmt.Fprintf(w, "Go! slow %v", headOrTails)
		//fmt.Printf("Go! slow %v", headOrTails)
		return
	}

	time.Sleep(1 * time.Second)
	fmt.Fprintf(w, "Go! quick %v", headOrTails)
	//fmt.Printf("Go! quick %v", headOrTails)
	return
}

func main2() {

	rand.Seed(time.Now().UTC().UnixNano())

	ts := httptest.NewServer(http.HandlerFunc(TestHandler))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)

	}
	text, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)

	}

	fmt.Printf("%s", text)

}

func main() {
	bi := Scraper{
		Name:     "borsaitaliana",
		ParseDoc: borsaitaliana,
	}

	ctx := context.TODO()
	stockId := "btp"
	url := "http://www.borsaitaliana.it/borsa/obbligazioni/mot/btp/scheda/IT0004009673.html?lang=it"
	res := bi.GetStockPrice(ctx, stockId, url)
	fmt.Printf("res = %+v\n", res)
	fmt.Printf("Elapsed = %+v\n", res.TimeEnd.Sub(res.TimeStart))
}
