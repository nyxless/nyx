package x

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/redis"
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

func (this *RedisProxy) Get(conf_name string) (*redis.RedisClient, error) { //{{{
	var err error

	this.mutex.RLock()
	v, ok := this.c[conf_name]
	this.mutex.RUnlock()

	if ok {
		_, err := v.Ping(context.Background()).Result()
		if err != nil {
			v.Close()
			ok = false
		}
	}

	if !ok {
		v, err = this.add(conf_name)
	}

	return v, err
} // }}}

func (this *RedisProxy) add(conf_name string) (*redis.RedisClient, error) { //{{{
	this.mutex.Lock()
	defer this.mutex.Unlock()

	config := Conf.GetMap(conf_name)
	if 0 == len(config) {
		return nil, fmt.Errorf("Redis 资源不存在: %s", conf_name)
	}

	var hosts []string
	if addr, ok := config["addr"]; ok {
		if host, ok := addr.(string); ok {
			hosts = []string{host}
		} else {
			hosts = AsStringSlice(addr)
		}
	}

	options := &redis.Options{
		Addrs: hosts,
	}

	if master_name, ok := config["master_name"]; ok {
		options.MasterName = AsString(master_name)
	}

	if password, ok := config["password"]; ok {
		options.Password = AsString(password)
	}

	if db, ok := config["db"]; ok {
		options.DB = AsInt(db)
	}

	if max_retries, ok := config["max_retries"]; ok {
		options.MaxRetries = AsInt(max_retries)
	}

	if max_redirects, ok := config["max_redirects"]; ok {
		options.MaxRedirects = AsInt(max_redirects)
	}

	if read_only, ok := config["read_only"]; ok {
		options.ReadOnly = AsBool(read_only)
	}

	if route_by_latency, ok := config["route_by_latency"]; ok {
		options.RouteByLatency = AsBool(route_by_latency)
	}

	if route_randomly, ok := config["route_randomly"]; ok {
		options.RouteRandomly = AsBool(route_randomly)
	}

	if pool_size, ok := config["pool_size"]; ok {
		options.PoolSize = AsInt(pool_size)
	}

	if min_idle_conns, ok := config["min_idle_conns"]; ok {
		options.MinIdleConns = AsInt(min_idle_conns)
	}

	if max_idle_conns, ok := config["max_idle_conns"]; ok {
		options.MaxIdleConns = AsInt(max_idle_conns)
	}

	if conn_max_idle_time, ok := config["conn_max_idle_time"]; ok {
		options.ConnMaxIdleTime = time.Duration(AsInt(conn_max_idle_time))
	}

	if conn_max_lifetime, ok := config["conn_max_lifetime"]; ok {
		options.ConnMaxLifetime = time.Duration(AsInt(conn_max_lifetime))
	}

	if read_timeout, ok := config["read_timeout"]; ok {
		options.ReadTimeout = time.Duration(AsInt(read_timeout))
	}

	if write_timeout, ok := config["write_timeout"]; ok {
		options.WriteTimeout = time.Duration(AsInt(write_timeout))
	}

	if username, ok := config["username"]; ok {
		options.Username = AsString(username)
	}

	rc := redis.NewRedisClient(options)

	ctx := context.Background()
	// 测试连接
	_, err := rc.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("无法连接到 Redis: [%v] %v", hosts, err)
	}

	this.c[conf_name] = rc
	Printf("add redis : [ %v ]\n", hosts)

	return rc, nil
} // }}}
