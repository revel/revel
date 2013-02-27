package cache

import (
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

		// By default, use the in-memory cache.
		Instance = NewInMemoryCache(defaultExpiration)
	})
}
