package cache

import "github.com/tschuyebuhl/scraper/data"

type Cache interface {
	Get(key string) (*data.PageData, bool)
	Put(value *data.PageData)
	Delete(key string)
	Nuke(sure bool)
}
