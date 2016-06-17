// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"fmt"
	"reflect"
	"time"

	"github.com/revel/revel"
	"github.com/robfig/go-cache"
)

type InMemoryCache struct {
	cache.Cache
}

func NewInMemoryCache(defaultExpiration time.Duration) InMemoryCache {
	return InMemoryCache{*cache.New(defaultExpiration, time.Minute)}
}

func (c InMemoryCache) Get(key string, ptrValue interface{}) error {
	value, found := c.Cache.Get(key)
	if !found {
		return ErrCacheMiss
	}

	v := reflect.ValueOf(ptrValue)
	if v.Type().Kind() == reflect.Ptr && v.Elem().CanSet() {
		v.Elem().Set(reflect.ValueOf(value))
		return nil
	}

	err := fmt.Errorf("revel/cache: attempt to get %s, but can not set value %v", key, v)
	revel.ERROR.Println(err)
	return err
}

func (c InMemoryCache) GetMulti(keys ...string) (Getter, error) {
	return c, nil
}

func (c InMemoryCache) Set(key string, value interface{}, expires time.Duration) error {
	// NOTE: go-cache understands the values of DefaultExpiryTime and ForEverNeverExpiry
	c.Cache.Set(key, value, expires)
	return nil
}

func (c InMemoryCache) Add(key string, value interface{}, expires time.Duration) error {
	err := c.Cache.Add(key, value, expires)
	if err == cache.ErrKeyExists {
		return ErrNotStored
	}
	return err
}

func (c InMemoryCache) Replace(key string, value interface{}, expires time.Duration) error {
	if err := c.Cache.Replace(key, value, expires); err != nil {
		return ErrNotStored
	}
	return nil
}

func (c InMemoryCache) Delete(key string) error {
	if found := c.Cache.Delete(key); !found {
		return ErrCacheMiss
	}
	return nil
}

func (c InMemoryCache) Increment(key string, n uint64) (newValue uint64, err error) {
	newValue, err = c.Cache.Increment(key, n)
	if err == cache.ErrCacheMiss {
		return 0, ErrCacheMiss
	}
	return
}

func (c InMemoryCache) Decrement(key string, n uint64) (newValue uint64, err error) {
	newValue, err = c.Cache.Decrement(key, n)
	if err == cache.ErrCacheMiss {
		return 0, ErrCacheMiss
	}
	return
}

func (c InMemoryCache) Flush() error {
	c.Cache.Flush()
	return nil
}
