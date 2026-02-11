package x

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/redis"
	"golang.org/x/sync/singleflight"
	"sync"
	"time"
)

func NewRedisProxy() *RedisProxy {
	return &RedisProxy{
		c:  make(map[string]*redis.RedisClient),
		sf: &singleflight.Group{},
	}
}

type RedisProxy struct {
	mutex sync.RWMutex
	c     map[string]*redis.RedisClient
	sf    *singleflight.Group
}

func (r *RedisProxy) Get(conf MAP) (*redis.RedisClient, error) { //{{{
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

	if client := r.getClient(key); client != nil {
		return client, nil
	}

	result, err, _ := r.sf.Do(key, func() (interface{}, error) {
		// 再次检查，防止在等待期间已经有其他goroutine创建了连接
		if client := r.getClient(key); client != nil {
			return client, nil
		}

		// 创建新连接
		return r.add(conf, key)
	})

	if err != nil {
		return nil, err
	}

	return result.(*redis.RedisClient), nil
} // }}}

func (r *RedisProxy) getClient(key string) *redis.RedisClient { // {{{
	r.mutex.RLock()
	client, ok := r.c[key]
	r.mutex.RUnlock()

	if !ok || client == nil {
		return nil
	}

	return client
} // }}}

func (r *RedisProxy) add(conf MAP, key string) (*redis.RedisClient, error) { //{{{
	var hosts []string

	if addr, ok := conf["host"]; ok {
		if host, ok := addr.(string); ok {
			hosts = []string{host}
		} else {
			hosts = AsStringSlice(addr)
		}
	}

	options := &redis.Options{
		Addrs:       hosts,
		DialTimeout: 3 * time.Second,
	}

	var to_cluster bool
	if v, ok := conf["to_cluster"]; ok {
		to_cluster = AsBool(v)
	}

	options.ToCluster = to_cluster || len(hosts) > 1

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

	client := redis.NewRedisClient(options)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	_, err := client.Ping(ctx).Result()
	cancel()

	if err != nil {
		client.Close()
		return nil, fmt.Errorf("无法连接到 Redis: [%v] %v", hosts, err)
	}

	r.mutex.Lock()
	r.c[key] = client
	r.mutex.Unlock()

	Printf("add RedisProxy : [ %v ]\n", hosts)

	return client, nil
} // }}}

func (r *RedisProxy) Close() { // {{{
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, client := range r.c {
		client.Close()
	}
	r.c = make(map[string]*redis.RedisClient)
} // }}}
