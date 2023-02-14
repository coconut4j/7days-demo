package lru

import "container/list"

//字典的定义是 map[string]*list.Element，键是字符串，值是双向链表中对应节点的指针。
//maxBytes 是允许使用的最大内存，nbytes 是当前已使用的内存，OnEvicted 是某条记录被移除时的回调函数，可以为 nil。
//键值对 entry 是双向链表节点的数据类型，在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
//为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。

type Cache struct {
	maxBytes  int64
	nbytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

type entry struct {
	Key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (Value, bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}

func (c *Cache) RemoveOld() {
	back := c.ll.Back()
	if back != nil {
		c.ll.Remove(back)
		kv := back.Value.(*entry)
		delete(c.cache, kv.Key)
		c.nbytes -= int64(len(kv.Key)) + int64(kv.value.Len())
		//回调
		if c.OnEvicted != nil {
			c.OnEvicted(kv.Key, kv.value)
		}
	}
}

// 如果键存在，则更新对应节点的值，并将该节点移到队尾。
// 不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
// 更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOld()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
