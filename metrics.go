package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
)

var (
	wordFrequency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "web_scraper_word_frequency",
			Help: "The frequency of each word on scraped URLs",
		},
		[]string{"url", "word"},
	)
)

func UpdateMetrics(url string, wordCounts map[string]int) {
	for word, count := range wordCounts {
		wordFrequency.WithLabelValues(url, word).Set(float64(count))
	}
}

func Serve() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		slog.Info("serving on :2112")
		err := http.ListenAndServe(":2112", nil)
		if err != nil {
			slog.Error("error serving: %s", err)
		}
	}()
}
