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
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"
)

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

type StockPriceResult struct {
	// name of the scraper that get the results
	Scraper string
	// url of the html page
	URL string
	// stock identifier
	Isin string
	// timestamps
	timeStart, timeEnd time.Time

	Price string
	Date  string
	Err   error
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

	fmt.Fprintf(w, "Go! quick %v", headOrTails)
	//fmt.Printf("Go! quick %v", headOrTails)
	return
}

func main() {

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
