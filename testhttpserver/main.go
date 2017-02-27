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
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {

	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <port>", os.Args[0])
	}
	sPort := os.Args[1]
	if _, err := strconv.Atoi(sPort); err != nil {
		log.Fatalf("Invalid port: %s (%s)", sPort, err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "--> :%s %s", sPort, req.URL.String())
		log.Printf("--> :%s %s", sPort, req.URL.String())
	})
	log.Printf("Listening on :%s", sPort)
	if err := http.ListenAndServe(":"+sPort, nil); err != nil {
		log.Fatalf("can't start server: %s", err)
	}
}
