package main

import (
	"github.com/tschuyebuhl/scraper/cache"
	"github.com/tschuyebuhl/scraper/scraper"
	"log/slog"
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

	resultsChan := make(chan map[string]int)
	var wg sync.WaitGroup
	c := cache.NewInMemoryCache()

	taskChan := make(chan string, 1000)

	// Worker pool
	const workerCount = 200
	sem := make(chan struct{}, workerCount)

	// Initialize the worker pool
	for i := 0; i < workerCount; i++ {
		go func() {
			for url := range taskChan {
				err := scraper.Scrape(url, c, resultsChan, &wg, 0, sem,
					taskChan)
				if err != nil {
					slog.Error("error scraping", "error", err)
				}
			}
		}()
	}

	// Seed initial URLs
	for _, url := range urls {
		wg.Add(1)
		taskChan <- url
	}

	go func() {
		wg.Wait()
		close(taskChan)
		close(resultsChan)
	}()

	for res := range resultsChan {
		UpdateMetrics("res", res)
		slog.Debug("results: %v", res)
	}

	// Wait for termination signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
}
