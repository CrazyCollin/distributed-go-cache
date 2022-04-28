package distributed_go_cache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

//
// TestGetter
// @Description: 测试回调函数是否正常
// @param t
//
func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback error")
	}
}

var db = map[string]string{
	"Sally":  "110",
	"Collin": "150",
	"Link":   "150",
}

func TestGet(t *testing.T) {
	//统计某个key回调函数调用次数
	loadCounts := make(map[string]int, len(db))
	gocache := NewGroup("weight", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[Processing Search Key]:", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s cannot find", key)
		}))
	for k, v := range db {
		//测试缓存未命中情况
		if view, err := gocache.Get(k); err != nil || view.String() != v {
			t.Fatal("failed get value")
		}
		//测试缓存命中情况
		if _, err := gocache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %v miss", k)
		}
	}

	if view, err := gocache.Get("unknown"); err == nil {
		t.Fatalf("%s should be empty", view)
	}
}
