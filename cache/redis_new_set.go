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

func (c RedisCache) Life(key string) (interface{}, error) {
	conn := c.pool.Get()
	defer conn.Close()

	return conn.Do("TTL", key)
}

func (c RedisCache) SADD(key string, expires time.Duration, args ...interface{}) error {

	//先把时间换算成秒
	times := int(expires / time.Second)

	conn := c.pool.Get()
	defer conn.Close()

	for _, v := range args {
		if times > 0 {
			//大于0的时候，就先设置数据，再设置时间
			_, err := conn.Do("SADD", key, v)
			if err != nil {
				return err
			}

			_, err = conn.Do("EXPIRE", key, times)
			if err != nil {
				return err
			}

		} else {
			//等于0的时候，就直接设置值。
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

func (c RedisCache) SISMEMBER(key string, value interface{}) ( bool, error) {
	conn := c.pool.Get()
	defer conn.Close()

	res, err := conn.Do("SISMEMBER", key, value)
	if err != nil {
		return false, err
	}

	if int64(res.(int64)) == 1 {
		return true, nil
	} else {
		return false, nil
	}
}
