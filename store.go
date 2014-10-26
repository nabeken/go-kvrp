package hugoreview

import (
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	redisMaxIdle     = 3
	redisIdleTimeout = 240 * time.Second
)

type Store interface {
	GetHost(key string) *Container
	DeleteHost(key string) error
	SetHost(key string, c *Container) error
}

type Container struct {
	ID   string `redis:"id"`
	Host string `redis:"host"`
}

type RedisStore struct {
	pool *redis.Pool
}

func NewRedisStore() *RedisStore {
	host := os.Getenv("REDIS_PORT_6379_TCP_ADDR")
	port := os.Getenv("REDIS_PORT_6379_TCP_PORT")
	pool := &redis.Pool{
		MaxIdle:     redisMaxIdle,
		IdleTimeout: redisIdleTimeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host+":"+port)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return &RedisStore{
		pool: pool,
	}
}

func (s *RedisStore) GetHost(key string) *Container {
	conn := s.pool.Get()
	defer conn.Close()

	c := &Container{}
	val, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return c
	}
	redis.ScanStruct(val, c)
	return c
}

func (s *RedisStore) SetHost(key string, c *Container) error {
	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("HMSET", key, "id", c.ID, "host", c.Host)
	return err
}

func (s *RedisStore) DeleteHost(key string) error {
	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	return err
}
