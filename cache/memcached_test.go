package cache

import (
	"net"
	"testing"
	"time"
)

// These tests require memcached running on localhost:11211 (the default)
const testServer = "localhost:11211"

var newMemcachedCache = func(t *testing.T, defaultExpiration time.Duration) Cache {
	c, err := net.Dial("tcp", testServer)
	if err == nil {
		c.Write([]byte("flush_all\r\n"))
		c.Close()
		return NewMemcachedCache([]string{testServer}, defaultExpiration)
	}
	t.Errorf("couldn't connect to memcached on %s", testServer)
	t.FailNow()
	panic("")
}

func TestMemcachedCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newMemcachedCache)
}

func TestMemcachedCache_IncrDecr(t *testing.T) {
	incrDecr(t, newMemcachedCache)
}

func TestMemcachedCache_Expiration(t *testing.T) {
	expiration(t, newMemcachedCache)
}

func TestMemcachedCache_EmptyCache(t *testing.T) {
	emptyCache(t, newMemcachedCache)
}

func TestMemcachedCache_Replace(t *testing.T) {
	testReplace(t, newMemcachedCache)
}

func TestMemcachedCache_Add(t *testing.T) {
	testAdd(t, newMemcachedCache)
}
