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
	"fmt"
	"time"
)

// First runs query on replicas and returns the first result.
func (s *Stock) GetStockPrice(ctx context.Context, scrapers Scrapers) *JobResult {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan *JobResult, len(s.Sources))

	search := func(spi StockPriceSource) {
		scr := scrapers[spi.ScraperName]
		c <- scr.GetStockPrice(ctx, s.ID, spi.URL)
	}

	for _, replica := range s.Sources {
		go search(replica)
	}

	select {
	case <-ctx.Done():
		return &JobResult{Err: ctx.Err()}
	case r := <-c:
		return r
	}
}

/*
			   1. in input ha tutte le richieste (ogni source di ogni stock)
			   2. in output restituisce un chan in cui verranno inviati i risultati
			      (1 risultato per ciascun stock)
			   3. per ogni tipo di scraper utilizzato,
			        - crea una coda di job,  chan (map[string]chan <- request)
			        - crea N istanze di scraper che lavora la coda, in base al numero
					  di richieste concorrenti gestibili dallo scraper
			   4. in input c'è anche il context
			   5. per ogni stock, crea un nuovo context WithCancel -> ctx, cancel
			   6. ogni job_request ha le seguenti info:
			       - ctx dello stock
				   - stock_id
				   - url
				   - response_chan // dove inviare i risultati.
			   7. la job_response  ha i seguenti campi:
			       - scraper_name
				   - stock_id
				   - url
				   - time_start
				   - time_end
				   - err
				   - result {stock price, stock date}
			   8. ogni scraper prende dalla coda un job_request, la lavora,
			      e restituisce il risultato nel job_request.response_chan


	ordina i job, li raggruppa per scraper e per ogni scrpper
	var jobs map[string][]*job


	scraper_work := func(c chan *job_request, jobs []*job_request) {
		for _, job := range jobs {
			c <- job
		}
	}

	for scraper_name := range scrpaers {
		go scraper_work( scraper_chan[scraper_name], jobs[scraper_name] )

	}


*/

type dispatchLayoutItem struct {
	stockid string
	url     string
}

type dispatchLayout map[string][]dispatchLayoutItem

type JobRequest struct {
	ctx      context.Context
	stockid  string
	url      string
	respChan chan JobResult
}

func getSimpleDispatchLayout(stocks Stocks) dispatchLayout {
	dl := map[string][]dispatchLayoutItem{}

	for _, stock := range stocks {
		if stock.Disabled {
			continue
		}
		for _, src := range stock.Sources {
			if src.Disabled {
				continue
			}
			item := dispatchLayoutItem{
				stockid: stock.ID,
				url:     src.URL,
			}
			items, ok := dl[src.ScraperName]
			if !ok {
				items = []dispatchLayoutItem{}
			}
			dl[src.ScraperName] = append(items, item)
		}
	}
	return dl

}

func genReqChan(ctxs map[string]context.Context, items []dispatchLayoutItem, respChan chan JobResult) chan *JobRequest {
	out := make(chan *JobRequest)
	go func() {
		for _, item := range items {
			job := &JobRequest{
				ctx:      ctxs[item.stockid],
				stockid:  item.stockid,
				url:      item.url,
				respChan: respChan,
			}
			out <- job
		}
		close(out)
	}()
	return out
}

func Dispatch(ctx context.Context, stocks Stocks, scrapersConfig []*ScraperConfig) {

	dispatchLayout := getSimpleDispatchLayout(stocks)

	// delete disabled scrapers from layout
	for _, sc := range scrapersConfig {
		if sc.Disabled {
			fmt.Printf("deleting scraper %q\n", sc.Name)
			delete(dispatchLayout, sc.Name)
		}
	}
	// print dispatchLayout
	for k, v := range dispatchLayout {
		fmt.Printf("%s:\n", k)
		for j, s := range v {
			fmt.Printf("    [%d] %+v\n", j, s)
		}
	}

	// create the results chan
	respChan := make(chan JobResult)

	// create a context and cancel for each stock
	ctxs := map[string]context.Context{}
	cancels := map[string]context.CancelFunc{}
	for _, stock := range stocks {
		ctx0, cancel0 := context.WithCancel(ctx)
		ctxs[stock.ID] = ctx0
		cancels[stock.ID] = cancel0
	}

	// create a request chan for each enabled scraper
	// and enqueues the jobs
	reqChan := map[string]chan *JobRequest{}
	for scraperName, items := range dispatchLayout {
		reqChan[scraperName] = genReqChan(ctxs, items, respChan)
	}

	// crea le istanze degli scraper che lavorano le code di jobs

}

func main() {
	t1 := time.Now()

	numScrapers := 3
	numStocks := 10

	// init test stock server
	testStockServer := initTestStockServer()
	// init stocks
	stocks := initTestStocks(numStocks, numScrapers, testStockServer.URL)

	scraperCfg := initScrapersConfig(numScrapers)

	ctx := context.Background()

	Dispatch(ctx, stocks, scraperCfg)

	//spi := stock0.Sources[1]
	//scraper := scrapers[spi.ScraperName]
	//res := scraper.GetStockPrice(ctx, stock0.ID, spi.URL)

	//res := stocks["id9"].GetStockPrice(ctx, scrapers)

	//fmt.Printf("res = %+v\n", res)
	//fmt.Printf("Elapsed = %+v\n", res.TimeEnd.Sub(res.TimeStart))

	t2 := time.Now()
	fmt.Printf("Total Elapsed = %+v\n", t2.Sub(t1))
}
