package gocache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_gocache/"

type HTTPPool struct {
	//记录本机名与端口
	self string
	//节点通讯地址前缀
	basePath string
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	//返回拷贝
	w.Write(view.ByteSlice())
}
