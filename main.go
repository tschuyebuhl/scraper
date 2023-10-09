package main

import (
	"github.com/tschuyebuhl/scraper/cache"
	"github.com/tschuyebuhl/scraper/data"
	"github.com/tschuyebuhl/scraper/scraper"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	Serve()

	urls := []string{
		"https://www.wikipedia.org",
	}

	//resultsChan := make(chan map[string]int)
	resultsChan := make(chan data.PageData)
	var wg sync.WaitGroup
	c := cache.NewInMemoryCache()

	taskChan := make(chan data.Task, 1000)

	const workerCount = 20
	sem := make(chan struct{}, workerCount)

	client := &http.Client{}

	for _, url := range urls {
		wg.Add(1)
		taskChan <- data.Task{URL: url}
	}

	for i := 0; i < workerCount; i++ {
		go func() {
			for task := range taskChan {
				err := scraper.Scrape(task.URL, c, resultsChan, &wg, 0, sem,
					taskChan, client)
				if err != nil {
					slog.Error("error scraping", "error", err)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(taskChan)
		close(resultsChan)
	}()

	for res := range resultsChan {
		UpdateMetrics(res.URL, res.WordFrequency)
		slog.Debug("results: %v", res)
	}

	// Wait for termination signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
}
