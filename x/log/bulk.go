package log

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

func (b *Bulk) Reset() {
	b.entrys = b.entrys[:0]
}
