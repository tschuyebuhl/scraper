package main

import (
	"github.com/tschuyebuhl/scraper/cache"
	"github.com/tschuyebuhl/scraper/data"
	"github.com/tschuyebuhl/scraper/scraper"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	Serve()

	urls := []string{
		"https://news.google.com",
	}

	resultsChan := make(chan data.PageData, 100)
	sem := make(chan struct{}, 20)

	c := cache.NewInMemoryCache()
	client := &http.Client{}

	for _, url := range urls {
		go func(url string) {
			err := scraper.Scrape(url, c, resultsChan, 0, client, sem)
			if err != nil {
				slog.Error("error scraping", "error", err)
			}
		}(url)
	}

	for res := range resultsChan {
		slog.Info("received results, updating metrics")
		UpdateMetrics(res.URL, res.WordFrequency)
	}

	slog.Info("finished scraping")

	// Wait for termination signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
}
