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
	"log"
	"math/rand"
	"sync"
	"time"
)

/*
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
*/

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

type dispatchItem struct {
	workers int
	jobs    []*Job
}

type dispatcher map[string]*dispatchItem

func (d dispatcher) Print() {

	for k, v := range d {
		fmt.Printf("%s: (workers=%d)\n", k, v.workers)
		for j, s := range v.jobs {
			fmt.Printf("  job[%d] %+v\n", j, s)
		}
	}

}

func (d dispatcher) shuffle() {

	shuffleJobs := func(src []*Job) []*Job {
		dest := make([]*Job, len(src))
		perm := rand.Perm(len(src))
		for i, v := range perm {
			dest[v] = src[i]
		}
		return dest
	}

	for _, item := range d {
		item.jobs = shuffleJobs(item.jobs)
	}

}

func newSimpleDispatcher(stocks Stocks, scrapers Scrapers) dispatcher {
	d := dispatcher(map[string]*dispatchItem{})

	for _, stock := range stocks {
		if stock.Disabled {
			continue
		}
		for _, source := range stock.Sources {
			if source.Disabled {
				continue
			}
			// check scraper
			scraper := scrapers[source.ScraperName]
			if scraper == nil || scraper.Disabled {
				continue
			}

			// stock, source and scrapers are not disabled
			//stock.enabled = true

			// create the job
			j := &Job{
				stockid: stock.ID,
				url:     source.URL,
			}

			// create/update the scraper's dispatchItem
			item, ok := d[scraper.Name]
			if !ok {
				d[scraper.Name] = &dispatchItem{workers: scraper.Workers, jobs: []*Job{j}}
				continue
			}
			item.jobs = append(item.jobs, j)
		}
	}
	d.shuffle()

	return d

}

func genJobRequestChan(ctxs map[string]context.Context, jobs []*Job, resChans map[string]chan *JobResult, scraperName string) chan *JobRequest {
	out := make(chan *JobRequest)
	go func() {
		for _, job := range jobs {
			stockid := job.stockid
			req := &JobRequest{
				ctx:     ctxs[stockid],
				resChan: resChans[stockid],
				job:     job,
			}
			log.Printf("ENQUEUE %s - %v\n", scraperName, job)
			out <- req
		}
		close(out)
	}()
	return out
}

func Dispatch(ctx context.Context, stocks Stocks, scrapers Scrapers) <-chan *JobResult {

	dispatcher := newSimpleDispatcher(stocks, scrapers)
	dispatcher.Print()

	// for each enabled stockid gives the number of enabled sources
	sources := map[string]int{}

	for _, item := range dispatcher {
		for _, job := range item.jobs {
			count := sources[job.stockid]
			sources[job.stockid] = count + 1
		}
	}

	for stockid, count := range sources {
		fmt.Printf("sources[%s] = %d\n", stockid, count)
	}

	// create a context with cancel and a result chan for each enabled stock
	ctxs := map[string]context.Context{}
	cancels := map[string]context.CancelFunc{}
	resChans := map[string]chan *JobResult{}
	for stockid, count := range sources {
		ctx0, cancel0 := context.WithCancel(ctx)
		ctxs[stockid] = ctx0
		cancels[stockid] = cancel0
		resChans[stockid] = make(chan *JobResult, count)
		//resChans[stockid] = make(chan *JobResult)
	}

	// create a request chan for each enabled scraper
	// and enqueues the jobs
	reqChan := map[string]chan *JobRequest{}
	for scraperName, item := range dispatcher {
		reqChan[scraperName] = genJobRequestChan(ctxs, item.jobs, resChans, scraperName)
	}

	out := make(chan *JobResult)

	var wg sync.WaitGroup

	// raccoglie le risposte per ogni stock enabled

	wg.Add(len(sources))
	for stockid, count := range sources {
		go func(stockid string, count int) {
			todo := true

			for ; count > 0; count-- {

				select {
				case res := <-resChans[stockid]:

					// if not already done, send the result if ok,
					// or if it is the last result.
					if todo && (res.Err == nil || count == 1) {
						todo = false
						cancels[stockid]()
						out <- res
					}
				case <-ctxs[stockid].Done():
					cancels[stockid]()
				}

			}
			wg.Done()
		}(stockid, count)
	}
	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		log.Println("CLOSING OUT")
		close(out)

	}()

	// crea le istanze dei workers che lavorano i jobs
	for name, item := range dispatcher {
		for j := 0; j < item.workers; j++ {
			worker := newWorker(scrapers[name], j+1)

			go func(w *Worker, input <-chan *JobRequest) {
				// per ogni job request ottenuto dal chan
				for req := range input {
					req.resChan <- w.doJob(req.ctx, req.job)
				}

			}(worker, reqChan[name])
		}
	}

	return out

}

func main() {
	log.SetFlags(log.Lmicroseconds)
	t1 := time.Now()

	numScrapers := 4
	numStocks := 3

	// init test stock server
	testStockServer := initTestStockServer()
	// init stocks
	stocks := initTestStocks(numStocks, numScrapers, testStockServer.URL)

	//fmt.Printf("Stocks = %+v\n", stocks)
	for k, s := range stocks {
		fmt.Printf("stock %s : [", k)
		for _, v := range s.Sources {
			fmt.Printf("%s, ", v.ScraperName)
		}
		fmt.Println("]")
	}
	fmt.Println()

	scrapers := initScrapers(numScrapers)

	ctx := context.Background()

	out := Dispatch(ctx, stocks, scrapers)
	for r := range out {
		log.Printf("RESULT  %v\n", r)
	}

	//spi := stock0.Sources[1]
	//scraper := scrapers[spi.ScraperName]
	//res := scraper.GetStockPrice(ctx, stock0.ID, spi.URL)

	//res := stocks["id9"].GetStockPrice(ctx, scrapers)

	//fmt.Printf("res = %+v\n", res)
	//fmt.Printf("Elapsed = %+v\n", res.TimeEnd.Sub(res.TimeStart))

	t2 := time.Now()
	fmt.Printf("Total Elapsed = %+v\n", t2.Sub(t1))
}
