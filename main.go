package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type urlCount struct {
	url   string
	count int
}

func main() {
	urls := []string{
		"http://go.dev",
		"http://golang.org",
		"http://google.com",
		"http://vc.ru",
		"http://www.reddit.com/r/golang/",
		"http://diesel.elcat.kg",
		"https://pikabu.ru/tag/Golang",
		"http://airvuz.com",
		"http://www.kg",
		"http://gopro.com",
		"http://python.org",
		"http://souncloud.com",
	}
	concurrentJobs := 4
	chURLs := make(chan string, concurrentJobs)
	chResults := make(chan *urlCount, concurrentJobs)

	for i := 0; i <= concurrentJobs; i++ {
		go processURLs(i, chURLs, chResults)
	}

	for _, url := range urls {
		chURLs <- url
	}
	close(chURLs)

	sum := 0
	for range urls {
		val := <-chResults
		fmt.Printf("Count for %s: %d\n", val.url, val.count)
		sum += val.count
	}
	close(chResults)
	fmt.Println("Total: ", sum)
}

func processURLs(id int, chURLs <-chan string, chResult chan<- *urlCount) {
	for val := range chURLs {
		response, err := makeHTTPRequest(val)
		if err != nil {
			netErr := new(net.DNSError)
			switch {
			case errors.As(err, &netErr):
				log.Printf("failed to find a host %s: %v\n", val, err)
				chResult <- newURLCount(val, 0, id)
				continue
			case errors.Is(err, context.DeadlineExceeded):
				log.Printf("failed to get response from host %s: %v\n", val, err)
				chResult <- newURLCount(val, 0, id)
				continue
			}

			log.Fatal("failed to make an http request: ", err)
		}

		chResult <- newURLCount(val, bytes.Count(response, []byte("go")), id)
	}
}

func newURLCount(url string, count, workerID int) *urlCount {
	return &urlCount{
		url:   url,
		count: count,
	}
}

func makeHTTPRequest(url string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create an http request: %w", err)
	}

	client := new(http.Client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Do(request.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to do an http request: %w", err)
	}

	defer response.Body.Close()

	return io.ReadAll(response.Body)
}
