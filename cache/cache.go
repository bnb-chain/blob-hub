package cache

import (
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

const DefaultCacheSize = 1024

type LocalCache struct {
	*lru.Cache
}

func NewLocalCache(size uint64) (Cache, error) {
	cache, err := lru.New(int(size))
	if err != nil {
		return nil, err
	}
	return &LocalCache{
		cache,
	}, nil
}

func (c *LocalCache) Get(key string) (interface{}, bool) {
	return c.Cache.Get(key)
}

func (c *LocalCache) Set(key string, value interface{}) {
	c.Cache.Add(key, value)
}
