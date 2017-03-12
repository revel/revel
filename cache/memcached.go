// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/revel/revel"
)

// MemcachedCache wraps the Memcached client to meet the Cache interface.
type MemcachedCache struct {
	*memcache.Client
	defaultExpiration time.Duration
}

func NewMemcachedCache(hostList []string, defaultExpiration time.Duration) MemcachedCache {
	return MemcachedCache{memcache.New(hostList...), defaultExpiration}
}

func (c MemcachedCache) Set(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Set, key, value, expires)
}

func (c MemcachedCache) Add(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Add, key, value, expires)
}

func (c MemcachedCache) Replace(key string, value interface{}, expires time.Duration) error {
	return c.invoke((*memcache.Client).Replace, key, value, expires)
}

func (c MemcachedCache) Get(key string, ptrValue interface{}) error {
	item, err := c.Client.Get(key)
	if err != nil {
		return convertMemcacheError(err)
	}
	return Deserialize(item.Value, ptrValue)
}

func (c MemcachedCache) GetMulti(keys ...string) (Getter, error) {
	items, err := c.Client.GetMulti(keys)
	if err != nil {
		return nil, convertMemcacheError(err)
	}
	return ItemMapGetter(items), nil
}

func (c MemcachedCache) Delete(key string) error {
	return convertMemcacheError(c.Client.Delete(key))
}

func (c MemcachedCache) Increment(key string, delta uint64) (newValue uint64, err error) {
	newValue, err = c.Client.Increment(key, delta)
	return newValue, convertMemcacheError(err)
}

func (c MemcachedCache) Decrement(key string, delta uint64) (newValue uint64, err error) {
	newValue, err = c.Client.Decrement(key, delta)
	return newValue, convertMemcacheError(err)
}

func (c MemcachedCache) Flush() error {
	err := errors.New("revel/cache: can not flush memcached")
	revel.ERROR.Println(err)
	return err
}

func (c MemcachedCache) invoke(f func(*memcache.Client, *memcache.Item) error,
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
	return convertMemcacheError(f(c.Client, &memcache.Item{
		Key:        key,
		Value:      b,
		Expiration: int32(expires / time.Second),
	}))
}

// ItemMapGetter implements a Getter on top of the returned item map.
type ItemMapGetter map[string]*memcache.Item

func (g ItemMapGetter) Get(key string, ptrValue interface{}) error {
	item, ok := g[key]
	if !ok {
		return ErrCacheMiss
	}

	return Deserialize(item.Value, ptrValue)
}

func convertMemcacheError(err error) error {
	switch err {
	case nil:
		return nil
	case memcache.ErrCacheMiss:
		return ErrCacheMiss
	case memcache.ErrNotStored:
		return ErrNotStored
	}

	revel.ERROR.Println("revel/cache:", err)
	return err
}
