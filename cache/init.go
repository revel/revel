package cache

import (
	"github.com/revel/revel"
	"strings"
	"time"
)

func init() {
	revel.OnAppStart(func() {
		// Set the default expiration time.
		defaultExpiration := time.Hour // The default for the default is one hour.
		if expireStr, found := revel.Config.String("cache.expires"); found {
			var err error
			if defaultExpiration, err = time.ParseDuration(expireStr); err != nil {
				panic("Could not parse default cache expiration duration " + expireStr + ": " + err.Error())
			}
		}

		// make sure you aren't trying to use both memcached and redis
		if revel.Config.BoolDefault("cache.memcached", false) && revel.Config.BoolDefault("cache.redis", false) {
			panic("You've configured both memcached and redis, please only include configuration for one cache!")
		}

		// Use memcached?
		if revel.Config.BoolDefault("cache.memcached", false) {
			hosts := strings.Split(revel.Config.StringDefault("cache.hosts", ""), ",")
			if len(hosts) == 0 {
				panic("Memcache enabled but no memcached hosts specified!")
			}

			Instance = NewMemcachedCache(hosts, defaultExpiration)
			return
		}

		// Use Redis (share same config as memcached)?
		if revel.Config.BoolDefault("cache.redis", false) {
			hosts := strings.Split(revel.Config.StringDefault("cache.hosts", ""), ",")
			if len(hosts) == 0 {
				panic("Redis enabled but no Redis hosts specified!")
			}
			if len(hosts) > 1 {
				panic("Redis currently only supports one host!")
			}
			password := revel.Config.StringDefault("cache.redis.password", "")
			Instance = NewRedisCache(hosts[0], password, defaultExpiration)
			return
		}

		// By default, use the in-memory cache.
		Instance = NewInMemoryCache(defaultExpiration)
	})
}
