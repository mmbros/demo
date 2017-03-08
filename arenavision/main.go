package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func avGet(URL string) (*http.Response, error) {

	//Host: github.com
	//User-Agent: Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:52.0) Gecko/20100101 Firefox/52.0
	//Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*[>;q=0.8
	//Accept-Language: en-US,en;q=0.5
	//Accept-Encoding: gzip, deflate, br
	//DNT: 1
	//Upgrade-Insecure-Requests: 1
	//Connection: keep-alive

	// Don’t use Go’s default HTTP client (in production)
	// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779#.q5iexu8v7
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:52.0) Gecko/20100101 Firefox/52.0")

	for name, val := range req.Header {
		fmt.Printf("%s: %s\n", name, val)
	}

	return client.Do(req)

}

func main() {

	resp, err := avGet("http://arenavision.in/schedule")
	//resp, err := avGet("http://mmbros.github.io")
	if err != nil {
		log.Fatal(err)
	}

	text, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	fmt.Print("-----------\n")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", text)

	fmt.Print("-----------\n")

	for name, val := range resp.Header {
		fmt.Printf("%s: %s\n", name, val)
	}

	//dump, err := httputil.DumpResponse(resp, true)
	//if err != nil {
	//log.Fatal(err)
	//}
	//fmt.Printf("%q", dump)

}
