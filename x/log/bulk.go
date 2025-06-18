package log

import (
	"sync"
)

// 默认每个bulk缓存切片大小
var DefaultBulkSize int = 32

type Bulk struct {
	entrys []*Entry
}

func (b *Bulk) IsFull() bool {
	return len(b.entrys) == cap(b.entrys)
}

func (b *Bulk) Append(entry *Entry) {
	b.entrys = append(b.entrys, entry)
}

func (b *Bulk) GetEntrys() []*Entry {
	return b.entrys
}

var bulkPool = sync.Pool{
	New: func() interface{} {
		return &Bulk{
			entrys: make([]*Entry, 0, DefaultBulkSize),
		}
	},
}

func GetBulk() *Bulk {
	return bulkPool.Get().(*Bulk)
}

func PutBulk(bulk *Bulk) {
	bulk.entrys = []*Entry{}
	bulkPool.Put(bulk)
}
