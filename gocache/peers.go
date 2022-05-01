package gocache

import (
	pb "gocache/gocachepb"
)

//
// PeerPicker
// @Description: 根据key选择节点
//
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//
// PeerGetter
// @Description: 节点必须实现以支持节点缓存查询
//
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
