package cache

import (
	"github.com/nyxless/nyx/x/log"
)

type LocalCache interface {
	Set(key, value []byte, expireSeconds int) (err error)
	Get(key []byte) (value []byte, err error)
	GetFn(key []byte, fn func([]byte) error) (err error)
	GetOrSet(key, value []byte, expireSeconds int) (retValue []byte, err error)
	GetOrSetFn(key []byte, fn func() (res []byte, update bool, err error), expireSeconds int, callbackFn ...func([]byte) error) (retValue []byte, hit bool, err error)
	GetOrRefreshFn(key []byte, fn func() (res []byte, update bool, err error), refreshInterval int, callbackFn ...func([]byte) error) (retValue []byte, hit bool, err error)
	SetAndGet(key, value []byte, expireSeconds int) (retValue []byte, found bool, err error)
	GetWithBuf(key, buf []byte) (value []byte, err error)
	Del(key []byte) (affected bool)
	HitRate() float64
	Clear()
	WithLogger(*log.Logger)
}

func NewLocalCache(cache_size int) LocalCache {
	return NewFreeCache(cache_size)
}
