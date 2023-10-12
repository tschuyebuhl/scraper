package cache

import (
	"github.com/tschuyebuhl/scraper/data"
	"log/slog"
	"sync"
)

type InMem struct {
	data map[string]*data.PageData
	mu   sync.RWMutex
}

func NewInMemoryCache() *InMem {
	return &InMem{
		data: make(map[string]*data.PageData),
	}
}

func (c *InMem) Delete(key string) {
	slog.Error("not implemented, ", "key", key)
}

func (c *InMem) Nuke(sure bool) {
	slog.Error("not implemented, ", "sure", sure)
}

func (c *InMem) Put(value *data.PageData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[value.URL] = value
}

func (c *InMem) Get(key string) (*data.PageData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}
