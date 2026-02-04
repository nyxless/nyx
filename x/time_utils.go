package x

import (
	"time"
)

// 当前时区location, 内部使用, TIME_ZONE 定义默认时区
func getLoc() *time.Location { // {{{
	if TIME_ZONE == "Local" {
		return time.Local
	}

	if TIME_ZONE == "UTC" {
		return time.UTC
	}

	loc, err := time.LoadLocation(TIME_ZONE)
	if nil != err {
		Panic(err)
	}

	return loc
} // }}}

// 当前 time.Time
func NowTime() time.Time { // {{{
	return time.Now()
} // }}}

// int 类型 unix时间戳
func Now() int { // {{{
	return int(time.Now().Unix())
} // }}}

// 时间字符串按 layout 格式转换为 time.Time
func ParseTime(s string, layouts ...string) time.Time { // {{{
	loc := getLoc()
	if len(layouts) > 0 {
		if t, err := time.ParseInLocation(layouts[0], s, loc); err == nil {
			return t
		}

		return time.Time{}
	}

	layouts = []string{ // {{{
		// ISO 8601 格式
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",

		// 数据库格式
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",

		// HTTP 头部格式
		time.RFC1123Z,
		time.RFC1123,
		time.RFC850,
		time.RFC822Z,
		time.RFC822,

		// Unix 格式
		time.UnixDate,
		time.ANSIC,
		time.RubyDate,

		// 短格式
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006",
		"20060102",

		// 时间格式
		"15:04:05.999",
		"15:04:05",
		"15:04",
		"3:04:05 PM",
		"3:04 PM",
		time.Kitchen,

		// 带时区变体
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -07:00",
		"Mon Jan 2 15:04:05 MST 2006",

		// 自然语言格式
		"January 2, 2006 15:04:05",
		"Jan 2, 2006 15:04:05",
		"Jan 2, 2006 3:04:05 PM",

		// 带毫秒变体
		time.StampNano,
		time.StampMicro,
		time.StampMilli,
		time.Stamp,

		// 其他
		"02-Jan-2006 15:04:05",
		"02-Jan-2006",
		"2006-01",
		"January 2006",
	} // }}}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t
		}
	}

	return time.Time{}
} // }}}

// 将任意时间类型转换为 layout 格式的字符串, 默认当前时间
func FormatTime(layout string, times ...any) string { // {{{
	var t time.Time
	if len(times) > 0 {
		switch val := times[0].(type) {
		case time.Time:
			t = val
		case int:
			t = time.Unix(int64(val), 0)
		case int32:
			t = time.Unix(int64(val), 0)
		case int64:
			t = time.Unix(val, 0)
		case string:
			t = ParseTime(val)
		default:
			t = time.Time{}
		}
	} else {
		t = time.Now()
	}

	loc := getLoc()
	return t.In(loc).Format(layout)
} // }}}

// 将任意时间类型转换为 2013-01-20 格式的日期, 默认当前时间
func Date(times ...any) string { // {{{
	return FormatTime("2006-01-02", times...)
} // }}}

// 将任意时间类型转换为 2006-01-02 15 格式的时间, 默认当前时间
func DateHour(times ...any) string { // {{{
	return FormatTime("2006-01-02 15", times...)
} // }}}

// 将任意时间类型转换为 2006-01-02 15:04 格式的时间, 默认当前时间
func DateMin(times ...any) string { // {{{
	return FormatTime("2006-01-02 15:04", times...)
} // }}}

// 返回2006-01-02 15:04:05 格式的时间, 可以指定时间戳，默认当前时间
func DateTime(times ...any) string { // {{{
	return FormatTime("2006-01-02 15:04:05", times...)
} // }}}

// 时间字符串转化为 int 时间戳
func StrToTime(datetime string, layouts ...string) int { // {{{
	return int(ParseTime(datetime, layouts...).Unix())
} // }}}

// 将任意类型转换为 time.Time
func AsTime(a any) (t time.Time) { // {{{
	if a == nil {
		return time.Time{}
	}

	switch val := a.(type) {
	case time.Time:
		t = val
	case int:
		t = time.Unix(int64(val), 0)
	case int32:
		t = time.Unix(int64(val), 0)
	case int64:
		t = time.Unix(val, 0)
	case string:
		t = ParseTime(val)
	default:
		t = time.Time{}
	}

	return t
} // }}}

// 生成 int 时间戳, 参数：小时,分,秒,月,日,年
func MkTime(t ...int) int { // {{{
	var M time.Month
	loc := getLoc()
	h, m, s, d, y := 0, 0, 0, 0, 0

	l := len(t)

	if l > 0 {
		h = t[0]
	}

	if l > 1 {
		m = t[1]
	}

	if l > 2 {
		s = t[2]
	}

	if l > 3 {
		M = time.Month(t[3])
	}

	if l > 4 {
		d = t[4]
	}

	if l > 5 {
		y = t[5]
	} else {
		tn := time.Now().In(loc)
		y = tn.Year()
		if l < 5 {
			d = tn.Day()
		}
		if l < 4 {
			M = tn.Month()
		}
		if l < 3 {
			s = tn.Second()
		}
		if l < 2 {
			m = tn.Minute()
		}
		if l < 1 {
			h = tn.Hour()
		}
	}

	td := time.Date(y, M, d, h, m, s, 0, loc)
	return int(td.Unix())
} // }}}

// 从start_time开始的消耗时间, 单位毫秒
func Cost(start_time time.Time) int64 { //start_time=time.Now()
	return time.Since(start_time).Milliseconds()
}
