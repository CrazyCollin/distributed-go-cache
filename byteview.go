package distributed_go_cache

//
// ByteView
// @Description: 表示缓存值
//
type ByteView struct {
	b []byte
}

//
// Len
// @Description: 返回缓存长度
// @receiver v
// @return int
//
func (v ByteView) Len() int {
	return len(v.b)
}

//
// ByteSlice
// @Description: 返回一份缓存的切片拷贝
// @receiver v
// @return []byte
//
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//
// String
// @Description: 返回缓存的字符串形式
// @receiver v
// @return string
//
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
