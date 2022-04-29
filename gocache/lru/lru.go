package lru

import "container/list"

type Cache struct {
	//允许使用的最大内存
	maxBytes int64
	//当前已使用内存
	nbytes int64
	ll     *list.List
	cache  map[string]*list.Element
	//某条记录被移除时的回调函数
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

//
// New
// @Description: 实例化Cache
// @param maxBytes
// @param OnEvicted
// @return *Cache
//
func New(maxBytes int64, OnEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: OnEvicted,
	}
}

//
// Get
// @Description: 查找策略，命中的节点移至队首
// @receiver c
// @param key
// @return value
// @return ok
//
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//
// RemoveOldest
// @Description: 删除策略
// @receiver c
//
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
	}
}

//
// Add
// @Description: 添加
// @receiver c
// @param key
// @param value
//
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
