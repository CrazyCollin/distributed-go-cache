package gocache

import (
	"fmt"
	"gocache/consistenthash"
	pb "gocache/gocachepb"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_gocache/"
	defaultReplicas = 50
)

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

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

//
// Log
// @Description:
// @receiver p
// @param format
// @param v
//
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

//
// ServeHTTP
// @Description: 处理请求
// @receiver p
// @param w
// @param r
//
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

//
// Set
// @Description: 实例化一致性哈希算法，并传入实例节点
// @receiver p
// @param peers
//
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	//在表中增加节点
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

//
// PickPeer
// @Description: 哈希列表中调用实际节点
// @receiver p
// @param key
// @return PeerGetter
// @return bool
//
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//调用不为空，且不为本身节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

//
// httpGetter
// @Description: http的实际客户端结构体，一个远程节点对应一个结构体
//
type httpGetter struct {
	baseURL string
}

//
// Get
// @Description:
// @receiver g
// @param group
// @param key
// @return []byte
// @return error
//
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
