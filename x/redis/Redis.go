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

// r.Text(r.Get(ctx, "key"))
func (r *RedisClient) Text(cmd *redis.Cmd) string { // {{{
	s, err := cmd.Text()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return ""
	}

	return s
} // }}}

func (r *RedisClient) Bool(cmd *redis.Cmd) bool { // {{{
	f, err := cmd.Bool()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return false
	}

	return f
} // }}}

func (r *RedisClient) Int(cmd *redis.Cmd) int { // {{{
	num, err := cmd.Int()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return 0
	}

	return num
} // }}}

func (r *RedisClient) Int64(cmd *redis.Cmd) int64 { // {{{
	num, err := cmd.Int64()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return 0
	}

	return num
} // }}}

func (r *RedisClient) Uint64(cmd *redis.Cmd) uint64 { // {{{
	num, err := cmd.Uint64()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return 0
	}

	return num
} // }}}

func (r *RedisClient) Float32(cmd *redis.Cmd) float32 { // {{{
	num, err := cmd.Float32()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return 0
	}

	return num
} // }}}

func (r *RedisClient) Float64(cmd *redis.Cmd) float64 { // {{{
	num, err := cmd.Float64()
	if err != nil {
		if err != redis.Nil { // key does not exists
			//log err
			//log.Error("redis err:", err)
		}

		return 0
	}

	return num
} // }}}

/*
ss, err := cmd.StringSlice()
ns, err := cmd.Int64Slice()
ns, err := cmd.Uint64Slice()
fs, err := cmd.Float32Slice()
fs, err := cmd.Float64Slice()
bs, err := cmd.BoolSlice()
*/
