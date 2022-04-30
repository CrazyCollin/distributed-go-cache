package gocache

import (
	"fmt"
	"gocache/singleflight"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//
// Group
// @Description: 一个缓存的命名空间，拥有唯一名称
//
type Group struct {
	name string
	//缓存未命中时进行回调获取数据
	getter    Getter
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

//
// NewGroup
// @Description:
// @param name
// @param cacheBytes
// @param getter
// @return *Group
//
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		mainCache: cache{cacheBytes: cacheBytes},
		getter:    getter,
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//
// RegisterPeers
// @Description: 实现PeerPicker的HTTPPool可以注入到Group中
// @receiver g
// @param peers
//
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("register PeerPicker called more than once")
	}
	g.peers = peers
}

//
// getFromPeer
// @Description: 实现访问远程节点的再一次封装
// @receiver g
// @param peer
// @param key
// @return ByteView
// @return error
//
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{bytes}, err
}

//
// GetGroup
// @Description:
// @param name
// @return *Group
//
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//
// Get
// @Description: 获取缓存
// @receiver g
// @param key
// @return ByteView
// @return error
//
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("requires key")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GoCache] hit")
		return v, nil
	}
	return g.load(key)
}

//
// load
// @Description: 缓存获取逻辑,首先尝试从远程拿缓存，其次再考虑本地取数据
// @receiver g
// @param key
// @return value
// @return err
//
func (g *Group) load(key string) (value ByteView, err error) {
	//本地缓存不命中
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err != nil {
					return value, nil
				}
				log.Println("[gocache] Failed to get remote data from peer :", peer)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	//回调数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	//填充本地缓存
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
