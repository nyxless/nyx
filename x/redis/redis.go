package redis

import (
	"github.com/redis/go-redis/v9"
)

type Options = redis.UniversalOptions

func NewRedisClient(options *Options) *RedisClient { // {{{
	return &RedisClient{
		UniversalClient: redis.NewUniversalClient(options),
	}
} // }}}

type RedisClient struct {
	redis.UniversalClient
}
