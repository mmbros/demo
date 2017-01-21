package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseDocFunc is ...
type ParseDocFunc func(doc *goquery.Document) (price string, date string, err error)

type Scraper struct {
	Name     string
	Disabled bool
	Workers  int
	parseDoc ParseDocFunc
}

type Scrapers map[string]*Scraper

type Worker struct {
	name     string
	index    int
	parseDoc ParseDocFunc
}

type Job struct {
	stockid string
	url     string
}

func (j *Job) String() string {

	return path.Base(j.url)
}

type JobRequest struct {
	ctx     context.Context
	resChan chan *JobResult
	job     *Job
}

// JobResult cointains the informations returned by a stock price scraper.
type JobResult struct {
	worker    *Worker
	job       *Job
	TimeStart time.Time
	TimeEnd   time.Time

	StockPrice string
	StockDate  string
	Err        error
}

func newWorker(scraper *Scraper, index int) *Worker {
	if scraper == nil {
		return nil
	}
	w := &Worker{
		name:     scraper.Name,
		index:    index,
		parseDoc: scraper.parseDoc,
	}
	return w
}

func (w *Worker) String() string {
	if w == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s-%d", w.name, w.index)
}

func (res *JobResult) String() string {
	return fmt.Sprintf("%s - %s, err:%v", res.worker.String(), res.job.stockid, res.Err)
}

func (res *JobResult) String2() string {
	return fmt.Sprintf(`Result{
	worker:     %v,
	stockId:    %v,
	url:        %v,
	stockPrice: %v,
	stockDate:  %v,
	err:        %v,
	timeStart:  %v,
	timeEnd:    %v,
	elapsed:    %v,
}`, res.worker.String(), res.job.stockid, res.job.url,
		res.StockPrice, res.StockDate, res.Err,
		res.TimeStart, res.TimeEnd, res.TimeEnd.Sub(res.TimeStart),
	)
}

func (w *Worker) doJob(ctx context.Context, job *Job) *JobResult {
	// check worker
	if w == nil {
		panic("GetStockPrice: worker is nil")
	}
	if w.parseDoc == nil {
		panic("GetStockPrice: worker.parseDoc is nil")
	}
	log.Printf("JOB IN  %s - %s", w, job)

	// init the result
	result := &JobResult{
		worker:    w,
		job:       job,
		TimeStart: time.Now(),
	}
	// use defer to set timeEnd
	defer func() {
		result.TimeEnd = time.Now()

		log.Printf("JOB OUT %s - %s, err:%v ", w, job, result.Err)
	}()

	// get the response
	resp, err := GetUrl(ctx, job.url)
	if err != nil {
		result.Err = err
		return result
	}
	if resp.StatusCode != http.StatusOK {
		result.Err = fmt.Errorf(resp.Status)
		return result
	}

	// create goquery document
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		result.Err = err
		return result
	}
	// parse the response
	price, date, err := w.parseDoc(doc)
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
