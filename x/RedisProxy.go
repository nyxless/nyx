package x

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/redis"
	"strings"
	"sync"
	"time"
)

func NewRedisProxy() *RedisProxy {
	return &RedisProxy{c: map[string]*redis.RedisClient{}}
}

type RedisProxy struct {
	mutex sync.RWMutex
	c     map[string]*redis.RedisClient
}

func (this *RedisProxy) Get(conf MAP) (*redis.RedisClient, error) { //{{{
	var err error
	var c *redis.RedisClient
	var key string

	host, ok := conf["host"]
	if !ok {
		return nil, fmt.Errorf("Redis 配置有误")
	}

	if hoststr, ok := host.(string); ok {
		key = hoststr
	} else {
		key = Join(host, ",")
	}

	this.mutex.RLock()
	c, ok = this.c[key]
	this.mutex.RUnlock()

	if ok {
		_, err = c.Ping(context.Background()).Result()
		if err != nil {
			c.Close()
			ok = false
		}
	}

	if !ok {
		c, err = this.add(conf)
	}

	return c, err
} // }}}

func (this *RedisProxy) add(conf MAP) (*redis.RedisClient, error) { //{{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var hosts []string
	var key string

	if addr, ok := conf["host"]; ok {
		if host, ok := addr.(string); ok {
			hosts = []string{host}
			key = host
		} else {
			hosts = AsStringSlice(addr)
			key = strings.Join(hosts, ",")
		}
	}

	options := &redis.Options{
		Addrs:       hosts,
		DialTimeout: 3 * time.Second,
	}

	if timeout, ok := conf["timeout"]; ok {
		options.DialTimeout = time.Duration(AsInt(timeout)) * time.Second
	}

	if timeout, ok := conf["dial_timeout"]; ok {
		options.DialTimeout = time.Duration(AsInt(timeout)) * time.Second
	}

	if master_name, ok := conf["master_name"]; ok {
		options.MasterName = AsString(master_name)
	}

	if password, ok := conf["password"]; ok {
		options.Password = AsString(password)
	}

	if db, ok := conf["db"]; ok {
		options.DB = AsInt(db)
	}

	if max_retries, ok := conf["max_retries"]; ok {
		options.MaxRetries = AsInt(max_retries)
	}

	if max_redirects, ok := conf["max_redirects"]; ok {
		options.MaxRedirects = AsInt(max_redirects)
	}

	if read_only, ok := conf["read_only"]; ok {
		options.ReadOnly = AsBool(read_only)
	}

	if route_by_latency, ok := conf["route_by_latency"]; ok {
		options.RouteByLatency = AsBool(route_by_latency)
	}

	if route_randomly, ok := conf["route_randomly"]; ok {
		options.RouteRandomly = AsBool(route_randomly)
	}

	if pool_size, ok := conf["pool_size"]; ok {
		options.PoolSize = AsInt(pool_size)
	}

	if min_idle_conns, ok := conf["min_idle_conns"]; ok {
		options.MinIdleConns = AsInt(min_idle_conns)
	}

	if max_idle_conns, ok := conf["max_idle_conns"]; ok {
		options.MaxIdleConns = AsInt(max_idle_conns)
	}

	if conn_max_idle_time, ok := conf["conn_max_idle_time"]; ok {
		options.ConnMaxIdleTime = time.Duration(AsInt(conn_max_idle_time))
	}

	if conn_max_lifetime, ok := conf["conn_max_lifetime"]; ok {
		options.ConnMaxLifetime = time.Duration(AsInt(conn_max_lifetime))
	}

	if read_timeout, ok := conf["read_timeout"]; ok {
		options.ReadTimeout = time.Duration(AsInt(read_timeout)) * time.Second
	}

	if write_timeout, ok := conf["write_timeout"]; ok {
		options.WriteTimeout = time.Duration(AsInt(write_timeout)) * time.Second
	}

	if username, ok := conf["username"]; ok {
		options.Username = AsString(username)
	}

	c := redis.NewRedisClient(options)

	ctx := context.Background()
	// 测试连接
	_, err := c.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("无法连接到 Redis: [%v] %v", hosts, err)
	}

	this.c[key] = c
	Printf("add redis : [ %v ]\n", hosts)

	return c, nil
} // }}}
