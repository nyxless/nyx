package log

import (
	"strconv"
	"sync"
	"time"
)

// 日志条目
type Entry struct {
	Level    string
	Time     string
	File     string
	Msg      string
	Args     []any
	Formated bool
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

func LogField(key string, value any) Field {
	return Field{Key: key, Value: value}
}

var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{}
	},
}

func GetEntry(is_format_msg bool, time_format, level_name, msg string, args ...any) *Entry {
	entry := entryPool.Get().(*Entry)

	t := time.Now()

	if time_format == "" || time_format == "TIMESTAMP" {
		entry.Time = strconv.FormatInt(t.Unix(), 10)
	} else {
		entry.Time = t.Format(time_format)
	}

	entry.Level = level_name
	entry.Msg = msg
	entry.Args = args
	entry.Formated = is_format_msg

	return entry
}

func PutEntry(entry *Entry) {
	entry.Level = ""
	entry.Time = ""
	entry.File = ""
	entry.Msg = ""
	entry.Args = entry.Args[:0]
	entry.Formated = false
	entryPool.Put(entry)
}
