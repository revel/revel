package cache

import (
	"fmt"
	"github.com/robfig/revel"
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

		// Use memcached?
		if revel.Config.BoolDefault("cache.memcached", false) {
			hosts := strings.Split(revel.Config.StringDefault("cache.hosts", ""), ",")
			if len(hosts) == 0 {
				panic("Memcache enabled but no memcached hosts specified!")
			}

			Instance = NewMemcachedCache(hosts, defaultExpiration)
			return
		}

		// Use redis?
		if revel.Config.BoolDefault("cache.redis", false) {
			var err error
			host := revel.Config.StringDefault("cache.hosts", "")
			pass := revel.Config.StringDefault("cache.password", "")
			if host == "" {
				panic("Redis enabled but no redis hosts specified!")
			}

			if pass == "" {
				Instance, err = NewRedisCache(host, defaultExpiration)
			} else {
				Instance, err = NewRedisCacheAuth(host, pass, defaultExpiration)
			}

			if err != nil {
				panic(fmt.Sprintf("Error connecting to redis! %s", err))
			}

			return
		}

		// By default, use the in-memory cache.
		Instance = NewInMemoryCache(defaultExpiration)
	})
}
