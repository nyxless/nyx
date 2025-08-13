package x

import (
	"time"
)

func NowTime() time.Time { // {{{
	return time.Now()
} // }}}

// unix时间戳
func Now() int { // {{{
	return int(time.Now().Unix())
} // }}}

func getLoc() *time.Location { // {{{
	if TIME_ZONE == "Local" {
		return time.Local
	}

	if TIME_ZONE == "UTC" {
		return time.UTC
	}

	loc, err := time.LoadLocation(TIME_ZONE)
	if nil != err {
		panic(err)
	}

	return loc
} // }}}

// 返回2013-01-20 格式的日期, 可以指定时间戳，默认当前时间
func Date(times ...any) string { // {{{
	return FormatTime("2006-01-02", times...)
} // }}}

// 返回`2013-01-20 10` 小时整点格式的时间, 可以指定时间戳，默认当前时间
func DateHour(times ...any) string { // {{{
	return FormatTime("2006-01-02 15", times...)
} // }}}

// 返回`2013-01-20 10:20` 分钟整点格式的时间, 可以指定时间戳，默认当前时间
func DateMin(times ...any) string { // {{{
	return FormatTime("2006-01-02 15:04", times...)
} // }}}

// 返回2013-01-20 10:20:01 格式的时间, 可以指定时间戳，默认当前时间
func DateTime(times ...any) string { // {{{
	return FormatTime("2006-01-02 15:04:05", times...)
} // }}}

func FormatTime(layout string, times ...any) string { // {{{
	var t time.Time
	if len(times) > 0 {
		switch val := times[0].(type) {
		case int:
			if val > 0 {
				t = time.Unix(int64(val), 0)
			}
		case time.Time:
			t = val
		default:
			t = time.Now()
		}
	} else {
		t = time.Now()
	}

	loc := getLoc()
	return t.In(loc).Format(layout)
} // }}}

// 日期时间字符串转化为时间戳
func StrToTime(datetime string) int { // {{{
	loc := getLoc()
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", datetime, loc)
	return int(t.Unix())
} // }}}

// 生成时间戳
// 参数：小时,分,秒,月,日,年
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

func ParseTimestamp(a any) time.Time { // {{{
	timestamp := AsInt64(a)
	return time.Unix(timestamp, 0)
} // }}}

func ParseDate(a any) time.Time { // {{{
	timeString := AsString(a)
	layout := "2006-01-02"
	t, _ := time.Parse(layout, timeString)

	return t
} // }}}

func ParseTime(a any) time.Time { // {{{
	timeString := AsString(a)
	layout := "15:04:05"
	t, _ := time.Parse(layout, timeString)

	return t
} // }}}

func ParseDateTime(a any) time.Time { // {{{
	timeString := AsString(a)
	layout := "2006-01-02 15:04:05"
	t, _ := time.Parse(layout, timeString)

	return t
} // }}}

func ParseDateTime64(a any) time.Time { // {{{
	timeString := AsString(a)
	layout := "2006-01-02 15:04:05.000"
	t, _ := time.Parse(layout, timeString)

	return t
} // }}}

// 从start_time开始的消耗时间, 单位毫秒
func Cost(start_time time.Time) int64 { //start_time=time.Now()
	return time.Since(start_time).Milliseconds()
}
