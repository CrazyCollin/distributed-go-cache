<a name="Yn2qS"></a>
# 简单的分布式缓存设计
<a name="MUbDv"></a>
## 简单介绍
groupcache是一个k-v键值对缓存，也是memcache的go版本。groupcache不像其他缓存数据库是C/S架构，每一个节点都是一个客户端或者是服务端。当本地节点在缓存中查找不到对应的数据之后，会先通过一致性哈希办法从其他peer节点获取对应缓存数据，假如也没有查找到缓存数据，则直接通过回调函数在本地获取对应缓存数据并将其放入缓存中

同时对于节点本地缓存的管理部分，并没有删除，采用了LRU算法用于处理超过缓冲区大小的缓存

对于节点之间的通信，采取http的方式进行缓存数据的传送

在这里设计了一个mini版的groupcache，对groupcache大部分精髓部分都进行模仿<br />![image.png](https://cdn.jsdelivr.net/gh/CrazyCollin/image@master/uPic/oIT3Iu.png)
<a name="alevo"></a>

## 查询缓存流程
![](https://cdn.jsdelivr.net/gh/CrazyCollin/image@master/uPic/IFh1fo.jpg)
<a name="wdXji"></a>

## 交互逻辑
整体的交互逻辑设置在Group部分，由Group负责对底层缓存的获取，Group是一个缓存的命名空间，拥有唯一命名空间，在Group中包含了对远程Peer节点的获取缓存的接口，本地获取数据的回调函数以及并发处理的策略
```go
type Group struct {
	//缓存空间命名
	name string
	//缓存未命中时进行回调获取数据
	getter    Getter
	//缓存
	mainCache cache
	//远程数据获取接口
	peers     PeerPicker
	//并发处理请求策略
	loader    *singleflight.Group
}
```
主要是获取缓存的逻辑，先在本地缓存中获取，其次通过一致性哈希调用远程节点，最后才本地回调处理
```go
//本地调用策略
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
```
```go
//远程调用策略
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
```
```go
//本地回调策略
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
```
<a name="rJM6L"></a>
## 节点通信
在各个节点之间，采用了http的方式进行通讯，同时对于通讯数据部分采用protobuf进行压缩，提高通讯效率，同时将http通讯部分抽象解耦出来，作为唯一单独的http请求接口，封装一致性哈希算法，以及protobuf解编码，对其余节点将其存放至map中
```go
type HTTPPool struct {
	//记录本机名与端口
	self string
	//节点通讯地址前缀
	basePath string
	mu       sync.Mutex
	//一致性哈希Map
	peers *consistenthash.Map
	//本地客户端获取远程节点数据map
	httpGetters map[string]*httpGetter
}
```
客户端远程调用数据
```go
func (g *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		g.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	//发出http请求
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned:%v", res.Status)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body:%v", err)
	}
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body error:%v", err)
	}
	return nil
}
```
服务端处理客户端远程调用请求
```go
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path:" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, fmt.Sprintf("group %s not found", groupName), http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	//对查询结果用protobuf封装
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	//返回拷贝
	w.Write(body)
}
```
<a name="Lvwxu"></a>
## 并发控制
有时候对于面对并发的相同key的请求而言，假如缓存中并不存在相应key数据，或者缓存刚好过期，同时对于本地回调策略的数据库不加以限制访问，那么大量并发的请求就会将数据库压垮，这也叫做缓存穿透或者缓存击穿<br />对于这种策略，groupcache采用了singleflight进行了并发处理，这也是groupcache的精髓之处
```go
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
```
<a name="JMfsn"></a>
## 更新缓存策略
主要是采用了LRU算法对缓存进行管理，具体就是双向链表+map对存储对entry缓存节点进行管理，双向链表采用的是go原生封装的链表<br />缓存结构体
```go
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
```
<a name="hj4wT"></a>
## 一致性哈希
对于groupcache这种分布式缓存，采取普通的哈希办法一定是不行的，一旦节点下线或者上线新的节点，大量的数据映射将会失效，将会出现缓存雪崩，假如对各个节点进行重新映射，这个数据迁移成本也是巨大的，对于这种情况，一般是采用一致性哈希作为节点映射

一致性哈希算法主要是将key映射到2^32个空间上，采用一定的哈希策略将key映射到对应的主机上面，同时为了防止大量的key映射到同一个节点上造成单个访问不均匀的数据倾斜问题，采用了虚拟节点加以优化<br />![image.png](https://cdn.jsdelivr.net/gh/CrazyCollin/image@master/uPic/JtO2iW.png)<br />一致性哈希添加节点策略
```go
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
```
一致性哈希获取实际节点策略
```go
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
```
<a name="LpPv6"></a>
## 项目地址
> [https://github.com/CrazyCollin/distributed-go-cache](https://github.com/CrazyCollin/distributed-go-cache)

参考链接：极客兔兔
> [https://geektutu.com/](https://geektutu.com/)

感谢极客兔兔大佬的无私奉献

