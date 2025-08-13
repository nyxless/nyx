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

// String 返回字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 返回整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
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
	if time_format != "" {
		entry.Time = t.Format(time_format)
	} else {
		entry.Time = strconv.FormatInt(t.Unix(), 10)
	}

	entry.Level = level_name
	entry.Msg = msg
	entry.Args = args
	entry.Formated = is_format_msg

	return entry
}

func PutEntry(entry *Entry) {
	// 重置entry状态
	entry.Level = ""
	entry.Time = ""
	entry.File = ""
	entry.Msg = ""
	entry.Args = []any{}
	entry.Formated = false
	entryPool.Put(entry)
}
