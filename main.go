/*
References:
- [Context and Cancellation of goroutines](http://dahernan.github.io/2015/02/04/context-and-cancellation-of-goroutines/)
- [Cancelation, Context, and Plumbing](https://talks.golang.org/2014/gotham-context.slide#1)
- [Go Concurrency Patterns: Context](https://blog.golang.org/context)
- [Go Concurrency Patterns: Pipelines and cancellation](https://blog.golang.org/pipelines)

- [Beautiful Go patterns for concurrent access to shared resources and coordinating responses](http://dieter.plaetinck.be/post/beautiful_go_patterns_for_concurrent_access_to_shared_resources_and_coordinating_responses/)
- [Go by Example: Worker Pools](https://gobyexample.com/worker-pools)

https://gist.github.com/tmiller/5550127
A very simple example of using a map of channels for pub/sub in go.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type StockPriceSource struct {
	ScraperName string
	URL         string
	Disabled    bool
}

type Stock struct {
	ID       string
	Isin     string
	Name     string
	Disabled bool
	Sources  []StockPriceSource
}

type Stocks map[string]*Stock

type ScraperRequest struct {
	stockId     string
	scraperName string
	url         string
	ctx         context.Context
	cancel      context.CancelFunc
}

func testsource(doc *goquery.Document) (price string, date string, err error) {

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

func initScrapers(numScrapers int) Scrapers {
	scrapers := map[string]*Scraper{}
	for j := 0; j < numScrapers; j++ {
		name := getScraperName(j)
		scrapers[name] = &Scraper{
			Name:     name,
			ParseDoc: testsource,
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

// First runs query on replicas and returns the first result.
func (s *Stock) GetStockPrice(ctx context.Context, scrapers Scrapers) *Result {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan *Result, len(s.Sources))

	search := func(spi StockPriceSource) {
		scr := scrapers[spi.ScraperName]
		c <- scr.GetStockPrice(ctx, s.ID, spi.URL)
	}

	for _, replica := range s.Sources {
		go search(replica)
	}

	select {
	case <-ctx.Done():
		return &Result{Err: ctx.Err()}
	case r := <-c:
		return r
	}
}

// Enabled return true if the stock is enabled and it has at least one source enabled.
func (stock *Stock) Enabled() bool {

	if stock != nil && !stock.Disabled {
		for _, spi := range stock.Sources {
			if !spi.Disabled {
				return true
			}
		}
	}
	return false

}

func channelizeRequest(ctx context.Context, stocks Stocks) <-chan *ScraperRequest {
	out := make(chan *ScraperRequest)
	go func() {
		for _, stock := range stocks {
			if stock.Enabled() {
				ctx4stock, cancel4stock := context.WithCancel(ctx)

				for _, spi := range stock.Sources {
					if spi.Disabled {
						continue
					}
					sr := &ScraperRequest{
						stockId:     stock.ID,
						scraperName: spi.ScraperName,
						url:         spi.URL,
						ctx:         ctx4stock,
						cancel:      cancel4stock,
					}
					out <- sr
				}
			}
		}
		close(out)
	}()
	return out
}

//func doJob(ctx context.Context, stocks Stocks, scrapers Scrapers) {

//// creates a chan for each scraper
//scraperChan := map[string]chan *ScraperRequest{}
//for k, _ := range scrapers {
//scraperChan[k] = make(chan *ScraperRequest)
//}

//go func() {
//for stockId, stock := range stocks {
//// creates a context for each stock
//stockCtx, cancel := context.WithCancel(ctx)

//for _, spi := range stock.Sources {
//if spi.Disabled {
//continue
//}

//sch, ok := scraperChan[spi.ScraperName]
//if !ok {
//panic(fmt.Errorf("Invalid scraper %q for stock %q", spi.ScraperName, stock.ID))
//}

//sr := &ScraperRequest{
//ctx:     stockCtx,
//stockId: stockId,
//url:     spi.URL,
//}
//sch <- sr

//}

//}
//}()

//}
// First runs query on replicas and returns the first result.
func (s *Stock) GetStockPrice2(ctx context.Context, scrapers Scrapers) *Result {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan *Result, len(s.Sources))

	search := func(spi StockPriceSource) {
		scr := scrapers[spi.ScraperName]
		c <- scr.GetStockPrice(ctx, s.ID, spi.URL)
	}

	for _, replica := range s.Sources {
		go search(replica)
	}

	select {
	case <-ctx.Done():
		return &Result{Err: ctx.Err()}
	case r := <-c:
		return r
	}
}
func main() {
	t1 := time.Now()

	numScrapers := 3
	numStocks := 10

	// init test stock server
	testStockServer := initTestStockServer()
	// init scrapers
	scrapers := initScrapers(numScrapers)
	// init stocks
	stocks := initTestStocks(numStocks, numScrapers, testStockServer.URL)

	ctx := context.Background()

	//spi := stock0.Sources[1]
	//scraper := scrapers[spi.ScraperName]
	//res := scraper.GetStockPrice(ctx, stock0.ID, spi.URL)

	res := stocks["id9"].GetStockPrice(ctx, scrapers)

	fmt.Printf("res = %+v\n", res)
	fmt.Printf("Elapsed = %+v\n", res.TimeEnd.Sub(res.TimeStart))

	t2 := time.Now()
	fmt.Printf("Total Elapsed = %+v\n", t2.Sub(t1))
}
