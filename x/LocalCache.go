package x

import (
	"github.com/nyxless/nyx/x/cache"
)

func NewLocalCache(cache_sizes ...int) *cache.Cache {
	var cache_size int
	if len(cache_sizes) > 0 {
		cache_size = cache_sizes[0]
	}
	return cache.NewCache(cache_size)
}
