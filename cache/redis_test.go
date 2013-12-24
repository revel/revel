package cache

import (
	"net"
	"testing"
	"time"
)

// These tests require redis running on localhost:6379 (the default)
const testRedisServer = "localhost:6379"

var newRedisCache = func(t *testing.T, defaultExpiration time.Duration) Cache {
	c, err := net.Dial("tcp", testRedisServer)
	if err == nil {
		c.Write([]byte("FLUSHDB\r\n"))
		c.Close()
		c, err := NewRedisCache(testRedisServer, defaultExpiration)

		if err != nil {
			t.Errorf("couldn't connect to redis on %s", testRedisServer)
			t.FailNow()
		}
		return c
	}
	t.Errorf("couldn't connect to redis on %s", testRedisServer)
	t.FailNow()
	panic("")
}

func TestRedisCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newRedisCache)
}

func TestRedisCache_IncrDecr(t *testing.T) {
	incrDecr(t, newRedisCache)
}

func TestRedisCache_Expiration(t *testing.T) {
	expiration(t, newRedisCache)
}

func TestRedisCache_EmptyCache(t *testing.T) {
	emptyCache(t, newRedisCache)
}

func TestRedisCache_Replace(t *testing.T) {
	testReplace(t, newRedisCache)
}

func TestRedisCache_Add(t *testing.T) {
	testAdd(t, newRedisCache)
}

func TestRedisCache_GetMulti(t *testing.T) {
	testGetMulti(t, newRedisCache)
}

// Test parsing of redis host
func TestParseRedisHost(t *testing.T) {
	var host string
	var port uint64
	var err error

	// Test Parse w/ host:port
	host, port, err = ParseRedisHost("testhost:63799")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if host != "testhost" {
		t.Errorf("Expected testhost, but got: %s", host)
	}
	if port != 63799 {
		t.Errorf("Expected 63799, but got: %d", port)
	}

	// Test Parse w/ IP:PORT
	host, port, err = ParseRedisHost("127.0.0.1:63799")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if host != "127.0.0.1" {
		t.Errorf("Expected 127.0.0.1, but got: %s", host)
	}
	if port != 63799 {
		t.Errorf("Expected 63799, but got: %d", port)
	}

	// Test Parse w/ only host
	host, port, err = ParseRedisHost("testhost")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if host != "testhost" {
		t.Errorf("Expected testhost, but got: %s", host)
	}
	if port != 6379 {
		t.Errorf("Expected 6379, but got: %d", port)
	}
}
