package cache_test

import (
	"github.com/tschuyebuhl/scraper/data"
	"github.com/tschuyebuhl/scraper/scraper"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockCache struct {
	data  map[string]*data.PageData
	mutex sync.RWMutex
}

func (m *mockCache) Get(key string) (*data.PageData, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *mockCache) Put(key string, value *data.PageData) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = value
}

func (m *mockCache) Delete(key string) {

}

func (m *mockCache) Nuke(sure bool) {

}

type MockRequester struct{}

func (m *MockRequester) Get(url string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("<html>aaa</html>")),
	}, nil
}

func TestCacheBehaviors(t *testing.T) {
	resultsChan := make(chan data.PageData, 1)
	wg := &sync.WaitGroup{}
	sem := make(chan struct{}, 1)
	taskChan := make(chan string, 1)

	mc := &mockCache{
		data: map[string]*data.PageData{
			"https://example.com": {
				URL:           "https://example.com",
				WordFrequency: map[string]int{"test": 1},
				IsScraped:     false,
			},
		},
	}

	mockRequester := &MockRequester{}
	wg.Add(1)
	err := scraper.Scrape("https://example.com", mc, resultsChan, wg, 0, sem, taskChan, mockRequester)
	if err != nil {
		t.Error("Expected no error for already scraped URL, got", err)
	}

	select {
	case result := <-resultsChan:
		if result.WordFrequency["test"] != 1 {
			t.Errorf("Expected word frequency for 'test' to be 1, got %d", result.WordFrequency["test"])
		}
	case <-time.After(time.Second * 1):
		t.Errorf("resultchan cannot be empty")
	}

	mc = &mockCache{
		data: map[string]*data.PageData{
			"https://example.com": {
				URL:       "https://example.com",
				IsScraped: true,
			},
		},
	}
	wg.Add(1)
	err = scraper.Scrape("https://example.com", mc, resultsChan, wg, 0, sem, taskChan, mockRequester)
	if err != nil {
		t.Error("error for URL that's being scraped: ", err)
	}

	select {
	case <-resultsChan:
		t.Error("should not retrieve any results")
	case <-time.After(time.Second * 1):
	}
}
