package distributed_go_cache

import (
	"CrazyCollin/distributed-go-cache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

//
// add
// @Description: 封装lru的add方法，添加并发支持
// @receiver c
// @param key
// @param value
//
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//延迟初始化，懒汉式创建
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

//
// get
// @Description: 封装lru的get方法，添加并发支持
// @receiver c
// @param key
// @return value
// @return ok
//
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
