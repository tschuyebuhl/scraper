package cache

import "github.com/tschuyebuhl/scraper/data"

type Cache interface {
	Get(key string) (*data.PageData, bool)
	Put(key string, value *data.PageData)
	Delete(key string)
	Nuke(sure bool)
}
