package cache

import (
	"github.com/coocood/freecache"
)

var (
	DefaultCacheSize = 512 * 1024 * 1024 //100M
)

type Cache struct {
	*freecache.Cache
}

func NewCache(cache_size int) *Cache { // {{{
	return &Cache{freecache.NewCache(DefaultCacheSize)}
} // }}}
