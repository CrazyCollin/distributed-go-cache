package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

var defaultHashFunc = crc32.ChecksumIEEE

type Map struct {
	//哈希函数
	hash Hash
	//虚拟节点倍数
	replicas int
	//哈希环
	keys []int
	//虚拟节点和真实节点的映射表，键为虚拟节点hash值，值为真实节点名称
	hashMap map[int]string
}

//
// New
// @Description: 初始化虚拟节点倍数和哈希函数
// @param replicas
// @param fn
// @return *Map
//
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = defaultHashFunc
	}
	return m
}

//
// Add
// @Description: 根据节点名称添加节点
// @receiver m
// @param keys
//
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	//对哈希环进行排序
	sort.Ints(m.keys)
}

//
// Get
// @Description: 一致性哈希 得到key值对应节点
// @receiver m
// @param key
// @return string
//
func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}
	//计算哈希值
	hash := int(m.hash([]byte(key)))
	//keys范围长度内，找到大于等于计算得到hash值的第一个
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//哈希环获取
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
