// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/revel/revel"
)

// RedisCache wraps the Redis client to meet the Cache interface.
type RedisCache struct {
	pool              *redis.Pool
	defaultExpiration time.Duration
}

// NewRedisCache returns a new RedisCache with given parameters
// until redigo supports sharding/clustering, only one host will be in hostList
func NewRedisCache(host string, password string, defaultExpiration time.Duration) RedisCache {
	var pool = &redis.Pool{
		MaxIdle:     revel.Config.IntDefault("cache.redis.maxidle", 5),
		MaxActive:   revel.Config.IntDefault("cache.redis.maxactive", 0),
		IdleTimeout: time.Duration(revel.Config.IntDefault("cache.redis.idletimeout", 240)) * time.Second,
		Dial: func() (redis.Conn, error) {
			protocol := revel.Config.StringDefault("cache.redis.protocol", "tcp")
			toc := time.Millisecond * time.Duration(revel.Config.IntDefault("cache.redis.timeout.connect", 10000))
			tor := time.Millisecond * time.Duration(revel.Config.IntDefault("cache.redis.timeout.read", 5000))
			tow := time.Millisecond * time.Duration(revel.Config.IntDefault("cache.redis.timeout.write", 5000))
			c, err := redis.DialTimeout(protocol, host, toc, tor, tow)
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err = c.Do("AUTH", password); err != nil {
					_ = c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err = c.Do("PING"); err != nil {
					_ = c.Close()
					return nil, err
				}
			}
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return RedisCache{pool, defaultExpiration}
}

func (c RedisCache) Set(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	return c.invoke(conn.Do, key, value, expires)
}

func (c RedisCache) Add(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()

	existed, err := exists(conn, key)
	if err != nil {
		return err
	} else if existed {
		return ErrNotStored
	}
	return c.invoke(conn.Do, key, value, expires)
}

func (c RedisCache) Replace(key string, value interface{}, expires time.Duration) error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()

	existed, err := exists(conn, key)
	if err != nil {
		return err
	} else if !existed {
		return ErrNotStored
	}

	err = c.invoke(conn.Do, key, value, expires)
	if value == nil {
		return ErrNotStored
	}
	return err
}

func (c RedisCache) Get(key string, ptrValue interface{}) error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	raw, err := conn.Do("GET", key)
	if err != nil {
		return err
	} else if raw == nil {
		return ErrCacheMiss
	}
	item, err := redis.Bytes(raw, err)
	if err != nil {
		return err
	}
	return Deserialize(item, ptrValue)
}

func generalizeStringSlice(strs []string) []interface{} {
	ret := make([]interface{}, len(strs))
	for i, str := range strs {
		ret[i] = str
	}
	return ret
}

func (c RedisCache) GetMulti(keys ...string) (Getter, error) {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()

	items, err := redis.Values(conn.Do("MGET", generalizeStringSlice(keys)...))
	if err != nil {
		return nil, err
	} else if items == nil {
		return nil, ErrCacheMiss
	}

	m := make(map[string][]byte)
	for i, key := range keys {
		m[key] = nil
		if i < len(items) && items[i] != nil {
			s, ok := items[i].([]byte)
			if ok {
				m[key] = s
			}
		}
	}
	return RedisItemMapGetter(m), nil
}

func exists(conn redis.Conn, key string) (bool, error) {
	return redis.Bool(conn.Do("EXISTS", key))
}

func (c RedisCache) Delete(key string) error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	existed, err := redis.Bool(conn.Do("DEL", key))
	if err == nil && !existed {
		err = ErrCacheMiss
	}
	return err
}

func (c RedisCache) Increment(key string, delta uint64) (uint64, error) {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that. Since we need to do increment
	// ourselves instead of natively via INCRBY (redis doesn't support wrapping), we get the value
	// and do the exists check this way to minimize calls to Redis
	val, err := conn.Do("GET", key)
	if err != nil {
		return 0, err
	} else if val == nil {
		return 0, ErrCacheMiss
	}
	currentVal, err := redis.Int64(val, nil)
	if err != nil {
		return 0, err
	}
	sum := currentVal + int64(delta)
	_, err = conn.Do("SET", key, sum)
	if err != nil {
		return 0, err
	}
	return uint64(sum), nil
}

func (c RedisCache) Decrement(key string, delta uint64) (newValue uint64, err error) {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that, hence the exists call
	existed, err := exists(conn, key)
	if err != nil {
		return 0, err
	} else if !existed {
		return 0, ErrCacheMiss
	}
	// Decrement contract says you can only go to 0
	// so we go fetch the value and if the delta is greater than the amount,
	// 0 out the value
	currentVal, err := redis.Int64(conn.Do("GET", key))
	if err != nil {
		return 0, err
	}
	if delta > uint64(currentVal) {
		var tempint int64
		tempint, err = redis.Int64(conn.Do("DECRBY", key, currentVal))
		return uint64(tempint), err
	}
	tempint, err := redis.Int64(conn.Do("DECRBY", key, delta))
	return uint64(tempint), err
}

func (c RedisCache) Flush() error {
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	_, err := conn.Do("FLUSHALL")
	return err
}

func (c RedisCache) invoke(f func(string, ...interface{}) (interface{}, error),
	key string, value interface{}, expires time.Duration) error {

	switch expires {
	case DefaultExpiryTime:
		expires = c.defaultExpiration
	case ForEverNeverExpiry:
		expires = time.Duration(0)
	}

	b, err := Serialize(value)
	if err != nil {
		return err
	}
	conn := c.pool.Get()
	defer func() {
		_ = conn.Close()
	}()
	if expires > 0 {
		_, err = f("SETEX", key, int32(expires/time.Second), b)
		return err
	}
	_, err = f("SET", key, b)
	return err
}

// RedisItemMapGetter implements a Getter on top of the returned item map.
type RedisItemMapGetter map[string][]byte

func (g RedisItemMapGetter) Get(key string, ptrValue interface{}) error {
	item, ok := g[key]
	if !ok {
		return ErrCacheMiss
	}
	return Deserialize(item, ptrValue)
}
