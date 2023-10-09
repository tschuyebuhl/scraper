package cache

import (
	"github.com/tschuyebuhl/scraper/data"
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
	//TODO implement me
	panic("implement me")
}

func (c *InMem) Nuke(sure bool) {
	//TODO implement me
	panic("implement me")
}

func (c *InMem) Put(key string, value *data.PageData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *InMem) Get(key string) (*data.PageData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}
