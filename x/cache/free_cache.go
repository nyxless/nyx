package cache

import (
	"github.com/coocood/freecache"
	"github.com/nyxless/nyx/x/log"
	"github.com/nyxless/nyx/x/timer"
	"golang.org/x/sync/singleflight"
	"time"
)

type freeCache struct {
	cache     *freecache.Cache
	sf        singleflight.Group
	timerTask *timer.TimerTask
}

func NewFreeCache(cache_size int) *freeCache { // {{{
	timerTask := timer.NewTimerTask()
	timerTask.Start()

	return &freeCache{cache: freecache.NewCache(cache_size), timerTask: timerTask}
} // }}}

func (fc *freeCache) Set(key, value []byte, expireSeconds int) (err error) {
	return fc.cache.Set(key, value, expireSeconds)
}

func (fc *freeCache) Get(key []byte) (value []byte, err error) {
	return fc.cache.Get(key)
}

func (fc *freeCache) GetFn(key []byte, fn func([]byte) error) (err error) {
	return fc.cache.GetFn(key, fn)
}

func (fc *freeCache) GetOrSet(key, value []byte, expireSeconds int) (retValue []byte, err error) {
	return fc.cache.GetOrSet(key, value, expireSeconds)
}

func (fc *freeCache) GetOrSetFn(key []byte, fn func() (res []byte, update bool, err error), expireSeconds int, callbackFns ...func([]byte) error) (res []byte, hit bool, err error) { // {{{
	if val, err := fc.cache.Get(key); err == nil {
		return val, true, nil
	}

	var callbackFn func([]byte) error
	if len(callbackFns) > 0 {
		callbackFn = callbackFns[0]
	}

	return fc.setFn(key, fn, expireSeconds, callbackFn)
} // }}}

func (fc *freeCache) GetOrRefreshFn(key []byte, fn func() (res []byte, update bool, err error), refreshInterval int, callbackFns ...func([]byte) error) (res []byte, hit bool, err error) { // {{{
	if val, err := fc.cache.Get(key); err == nil {
		return val, true, nil
	}

	var callbackFn func([]byte) error
	if len(callbackFns) > 0 {
		callbackFn = callbackFns[0]
	}

	taskFn := func() error {
		_, _, err := fc.setFn(key, fn, refreshInterval*2, callbackFn)
		if err != nil {
			return err
		}
		return nil
	}

	cacheCallbackFn := func(data []byte) error {
		fc.addTask(key, taskFn, refreshInterval)

		if callbackFn != nil {
			return callbackFn(data)
		}

		return nil
	}

	return fc.setFn(key, fn, refreshInterval*2, cacheCallbackFn)
} // }}}

func (fc *freeCache) setFn(key []byte, fn func() (res []byte, update bool, err error), expireSeconds int, callbackFn func([]byte) error) (res []byte, hit bool, err error) { // {{{
	result, err, shared := fc.sf.Do(string(key), func() (any, error) { // {{{
		val, update, err := fn()
		if err != nil {
			return nil, err
		}

		if update {
			err = fc.cache.Set(key, val, expireSeconds)
			if err != nil {
				return nil, err
			}

			if callbackFn != nil {
				err := callbackFn(val)
				if err != nil {
					return nil, err
				}
			}
		}

		return val, nil
	}) // }}}

	if err != nil {
		return nil, false, err
	}

	return result.([]byte), shared, nil
} // }}}

func (fc *freeCache) SetAndGet(key, value []byte, expireSeconds int) (retValue []byte, found bool, err error) {
	return fc.cache.SetAndGet(key, value, expireSeconds)
}

func (fc *freeCache) GetWithBuf(key, buf []byte) (value []byte, err error) {
	return fc.cache.GetWithBuf(key, buf)
}

func (fc *freeCache) Del(key []byte) (affected bool) {
	return fc.cache.Del(key)
}

func (fc *freeCache) HitRate() float64 {
	return fc.cache.HitRate()
}

func (fc *freeCache) Clear() {
	fc.timerTask.Stop()
	fc.cache.Clear()
}

func (fc *freeCache) WithLogger(logger *log.Logger) {
	fc.timerTask.WithLogger(logger)
}

func (fc *freeCache) addTask(key []byte, fn func() error, refreshInterval int) error {
	return fc.timerTask.AddTask(string(key), time.Duration(refreshInterval)*time.Second, fn)
}
