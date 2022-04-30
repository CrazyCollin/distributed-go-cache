package singleflight

import "sync"

//
// call
// @Description: 正在进行中或者已经结束的请求，使用wg锁防止重入
//
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//
// Group
// @Description: 管理不同key的请求（call）
//
type Group struct {
	mu sync.Mutex
	//管理key为键的请求
	m map[string]*call
}

//
// Do
// @Description: 针对相同的key，保证函数fn只会被调用一次
// @receiver g
// @param key
// @param fn
// @return interface{}
// @return error
//
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	//懒加载
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//查看是否存在以key为参数的函数调用
	if c, ok := g.m[key]; ok {
		//等待调用完毕
		c.wg.Wait()
		return c.val, c.err
	}
	//第一次进行以key为参数的调用
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	//调用完毕，删除正在调用记录
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
