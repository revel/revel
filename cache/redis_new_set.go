package cache

import (
	"time"
)

func (c RedisCache) CheckRedis() error {
	conn := c.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PING"); err != nil {
		return err
	} else {
		return nil
	}
}

func (c RedisCache) Life(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("TTL", key)
	if err != nil {
		return -1, err
	} else {
		return int64(res.(int64)), nil
	}
}

func (c RedisCache) Type(key string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("TYPE", key)
	if err != nil {
		return "", err
	} else {
		return string(res.(string)), nil
	}
}

func (c RedisCache) SADD(key string, expires time.Duration, args ...interface{}) error {

	times := int(expires / time.Second)

	conn := c.pool.Get()
	defer conn.Close()

	for _, v := range args {
		if times > 0 {
			_, err := conn.Do("SADD", key, v)
			if err != nil {
				return err
			}

			_, err = conn.Do("EXPIRE", key, times)
			if err != nil {
				return err
			}

		} else {
			_, err := conn.Do("SADD", key, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c RedisCache) SREM(key string, args ...interface{}) error {
	conn := c.pool.Get()
	defer conn.Close()

	for _, v := range args {
		_, err := conn.Do("SREM", key, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c RedisCache) SISMEMBER(key string, value interface{}) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("SISMEMBER", key, value)
	if err != nil {
		return false, err
	}

	return int64(res.(int64)) == 1, nil
}

func (c RedisCache) SCARD(key string) (int64, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("SCARD", key)
	if err != nil {
		return -1, err
	}

	return int64(res.(int64)), nil
}
