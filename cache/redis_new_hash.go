package cache

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

func (c RedisCache) HSET(key, field, value string, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	times := int(expires / time.Second)

	if times > 0 {
		_, err := conn.Do("HSET", key, field, value)
		if err != nil {
			return err
		}
		_, err = conn.Do("EXPIRE", key, times)
		if err != nil {
			return err
		}
	} else {
		_, err := conn.Do("HSET", key, field, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c RedisCache) HGET(key, field string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	raw, err := conn.Do("HGET", key, field)
	if err != nil {
		return "", err
	} else if raw == nil {
		return "", ErrCacheMiss
	}

	item, err := redis.Bytes(raw, err)
	if err != nil {
		return "", err
	} else {
		return string(item), nil
	}

}

func (c RedisCache) HKEYS(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	var results []string

	items, err := redis.Values(conn.Do("HKEYS", key))
	if err != nil {
		return nil, err
	} else if items == nil {
		return nil, ErrCacheMiss
	}

	for _, v := range items {
		item, err := redis.Bytes(v, nil)
		if err != nil {
			return nil, err
		} else {
			results = append(results, string(item))
		}
	}
	return results, nil

}

func (c RedisCache) HLEN(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("HLEN", key)
	if err != nil {
		return -1, err
	} else {
		return int64(res.(int64)), nil
	}
}

func (c RedisCache) HSETMAP(key string, args map[string]string, expires time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	for k,v := range args {
		if err := c.HSET(key, k, v, expires); err != nil {
			return err
		}
	}
	return nil
}

func (c RedisCache) HGETMAP(key string, fields ...string) (map[string]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	values := make(map[string]string, len(fields))

	for _, v := range fields {
		res, err := c.HGET(key, v)
		if err != nil {
			return nil, err
		} else {
			values[v] = res
		}
	}

	return values, nil
}

func (c RedisCache) HDEL(key, field string) error {
	conn := c.pool.Get()
	defer conn.Close()

	_ , err := conn.Do("HDEL", key, field)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (c RedisCache) HMDEL(key string, fields ...string) error {
	conn := c.pool.Get()
	defer conn.Close()

	for _, v := range fields {
		if err := c.HDEL(key, v); err != nil {
			return err
		}
	}
	return nil
}
