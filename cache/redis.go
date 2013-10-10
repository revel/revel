package cache

import (
	"menteslibres.net/gosexy/redis"
	"net"
	"strconv"
	"strings"
	"time"
)

// Wraps the Redis client to meet the Cache interface
type RedisCache struct {
	*redis.Client
	defaultExpiration time.Duration
}

// ParseRedisHost parses a host:port string, defaulting to localhost:6379.
func ParseRedisHost(url string) (host string, port uint64, err error) {
	var sPort string

	if strings.Contains(url, ":") {
		host, sPort, err = net.SplitHostPort(url)
		if err != nil {
			return "", 0, err
		}
	} else {
		host = url
		sPort = "6379"
	}

	port, err = strconv.ParseUint(sPort, 10, 0)

	return
}

// NewRedisCacheAuth creates a new cache connection to a Redis instance with
// authentication. See NewRedisCache for more information.
func NewRedisCacheAuth(host string, pass string, defaultExpiration time.Duration) (RedisCache, error) {
	redisCache, err := NewRedisCache(host, defaultExpiration)
	if err != nil {
		return redisCache, err
	}

	_, err = redisCache.Client.Auth(pass)
	if err != nil {
		return RedisCache{}, err
	}

	return redisCache, nil
}

// NewRedisCache creates a new cache connection to a Redis instance.
// The host name can be a standard host:port connection, or a socket connection.
// Socket connections should start with 'file://' (e.g. file:///tmp/example.sock).
func NewRedisCache(host string, defaultExpiration time.Duration) (RedisCache, error) {
	var redisCache RedisCache
	client := redis.New()

	// Check to see if the "hostname" starts with "file://"
	// If it does, connect through a unix socket
	if strings.HasPrefix(host, "file://") {
		err := client.ConnectUnix(strings.TrimPrefix(host, "file://"))
		if err != nil {
			return RedisCache{}, err
		}

		redisCache = RedisCache{client, defaultExpiration}
	} else {
		// Assume standard host:port connection
		rHost, rPort, err := ParseRedisHost(host)
		if err != nil {
			return RedisCache{}, err
		}

		err = client.Connect(rHost, uint(rPort))
		if err != nil {
			return RedisCache{}, err
		}

		redisCache = RedisCache{client, defaultExpiration}
	}

	return redisCache, nil
}

func (c RedisCache) Set(key string, value interface{}, expires time.Duration) error {
	serialized, err := Serialize(value)
	if err != nil {
		return err
	}

	if expires == FOREVER {
		_, err = c.Client.Set(key, serialized)
	} else {
		_, err = c.Client.SetEx(key, c.expirationInSeconds(expires), serialized)
	}

	return err
}

func (c RedisCache) Add(key string, value interface{}, expires time.Duration) error {
	serialized, err := Serialize(value)
	if err != nil {
		return err
	}

	set, err := c.Client.SetNX(key, serialized)
	if err == nil {
		if set {
			// Set the expiration time
			if expires != FOREVER {
				_, err = c.Client.Expire(key, uint64(c.expirationInSeconds(expires)))
			}
		} else {
			return ErrNotStored
		}
	}

	return err
}

func (c RedisCache) Replace(key string, value interface{}, expires time.Duration) error {
	exists, err := c.Client.Exists(key)
	if err != nil {
		return err
	}

	if exists {
		return c.Set(key, value, expires)
	}

	return ErrNotStored
}

func (c RedisCache) Get(key string, ptrValue interface{}) error {
	exists, err := c.Client.Exists(key)
	if err != nil {
		return err
	}

	if !exists {
		return ErrCacheMiss
	}

	item, err := c.Client.Get(key)
	if err != nil {
		return err
	}
	return Deserialize([]byte(item), ptrValue)
}

func (c RedisCache) GetMulti(keys ...string) (Getter, error) {
	values := make(map[string]string)
	for _, k := range keys {
		v, err := c.Client.Get(k)
		if err != nil {
			return nil, err
		}

		values[k] = v
	}
	return RedisItemMapGetter(values), nil
}

func (c RedisCache) Delete(key string) error {
	exists, err := c.Client.Exists(key)
	if err != nil {
		return err
	}

	if !exists {
		return ErrCacheMiss
	}

	_, err = c.Client.Del(key)
	return err
}

func (c RedisCache) increment(key string, delta int64) (newValue uint64, err error) {
	exists, err := c.Client.Exists(key)
	if !exists {
		return 0, ErrCacheMiss
	}

	val, err := c.Client.IncrBy(key, delta)
	if err != nil {
		return 0, err
	}

	// Handle wraparound cases
	// Use DecrBy instead of Set to keep any TTLs
	if val < 0 {
		val, err = c.Client.DecrBy(key, val)
		if err != nil {
			return 0, err
		}
	}

	return uint64(val), nil
}

func (c RedisCache) Increment(key string, delta uint64) (newValue uint64, err error) {
	return c.increment(key, int64(delta))
}

func (c RedisCache) Decrement(key string, delta uint64) (newValue uint64, err error) {
	return c.increment(key, int64(-delta))
}

func (c RedisCache) Flush() error {
	_, err := c.Client.FlushDB()

	return err
}

func (c RedisCache) expirationInSeconds(expires time.Duration) int64 {
	switch expires {
	case FOREVER:
		return 0
	case DEFAULT:
		return int64(c.defaultExpiration.Seconds())
	default:
		return int64(expires.Seconds())
	}
}

// Implement a Getter on top of the returned item map.
type RedisItemMapGetter map[string]string

func (g RedisItemMapGetter) Get(key string, ptrValue interface{}) error {
	item, ok := g[key]
	if !ok {
		return ErrCacheMiss
	}

	return Deserialize([]byte(item), ptrValue)
}
