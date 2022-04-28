package distributed_go_cache

//
// ByteView
// @Description: 表示缓存值
//
type ByteView struct {
	b []byte
}

func (v ByteView) Len() int {
	return len(v.b)
}
