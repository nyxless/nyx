package x

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	json "github.com/bytedance/sonic" //"encoding/json"
	"github.com/cespare/xxhash/v2"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	RAND_KIND_NUM   = 0 // 纯数字
	RAND_KIND_LOWER = 1 // 小写字母
	RAND_KIND_UPPER = 2 // 大写字母
	RAND_KIND_ALL   = 3 // 数字、大小写字母
)

type randState struct {
	seed uint32
	_    [60]byte // 填充到64字节（独占缓存行)
}

var randPool = sync.Pool{ // {{{
	New: func() interface{} {
		// 初始化时创建新的随机状态
		rs := &randState{}
		rs.seed = uint32(uintptr(unsafe.Pointer(&rs))) ^ uint32(time.Now().UnixNano())

		return rs
	},
} // }}}

// 返回随机uint32 (线程安全)
func Uint32() uint32 { // {{{
	state := randPool.Get().(*randState)
	defer randPool.Put(state)

	// Xorshift算法核心
	state.seed ^= state.seed << 13
	state.seed ^= state.seed >> 17
	state.seed ^= state.seed << 5
	return state.seed
} // }}}

// 返回[0,n)范围内的随机整数, 替换math.rand.Intn
func Intn(n int) int {
	return int(Uint32()) % n
}

// 随机字符串
func Rand(size int, kind int) []byte { // {{{
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	//rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + Intn(scope))
	}
	return result
} // }}}

func RandStr(size int, kind ...int) string { // {{{
	k := RAND_KIND_ALL
	if len(kind) > 0 {
		k = kind[0]
	}
	return string(Rand(size, k))
} // }}}

// 获取map树的某个节点 m[k1][k2]...[kn]
func GetMapNode(m map[string]interface{}, keys ...string) (interface{}, bool) { // {{{
	if len(keys) == 0 {
		return nil, false
	}

	currentKey := keys[0]
	if len(keys) == 1 {
		if value, ok := m[currentKey]; ok {
			return value, true
		}

		return nil, false
	}

	if nextMap, ok := m[currentKey].(map[string]interface{}); ok {
		return GetMapNode(nextMap, keys[1:]...)
	}

	return nil, false
} // }}}

// 从interface{}树中获得一个MAP类型, 失败返回nil
func GetMap(i interface{}, keys ...string) MAP { // {{{
	m, ok := i.(MAP)
	if !ok {
		return nil
	}

	if len(keys) > 0 {
		n, ok := GetMapNode(m, keys...)
		if !ok {
			return nil
		}

		m, ok = n.(MAP)
		if !ok {
			return nil
		}
	}

	return m
} // }}}

// 从interface{}树中获得一个节点, 失败返回nil
func GetNode(i any, keys ...string) any { // {{{
	if len(keys) > 0 {
		m, ok := i.(MAP)
		if !ok {
			return nil
		}

		n, ok := GetMapNode(m, keys...)
		if !ok {
			return nil
		}

		return n
	}

	return i
} // }}}

// 从interface{}树中获得一个Slice类型, 失败返回nil
func GetSlice(i any, keys ...string) []any { // {{{
	if len(keys) > 0 {
		m, ok := i.(MAP)
		if !ok {
			return nil
		}

		n, ok := GetMapNode(m, keys...)
		if !ok {
			return nil
		}

		s, ok := n.([]any)
		if !ok {
			return nil
		}

		return s
	}

	if s, ok := i.([]any); ok {
		return s
	}

	return nil
} // }}}

// 合并MAP(一级)
func MapMerge[T, K comparable](m map[T]K, ms ...map[T]K) map[T]K { // {{{
	for _, v := range ms {
		for i, j := range v {
			m[i] = j
		}
	}

	return m
} // }}}

func ArrayColumn[T comparable](m []map[string]T, column string, uniqs ...bool) []T { // {{{
	if len(m) == 0 {
		return nil
	}

	n := []T{}

	if len(uniqs) > 0 && uniqs[0] {
		u := map[T]int{}
		for _, i := range m {
			k := i[column]
			if _, ok := u[k]; !ok {
				n = append(n, k)
				u[k] = 1
			}
		}
	} else {
		for _, i := range m {
			n = append(n, i[column])
		}
	}

	return n
} //}}}

func ArrayColumnMap[T comparable](m []map[string]T, column string, index string) map[T]T { // {{{
	n := map[T]T{}
	for _, i := range m {
		n[i[index]] = i[column]
	}

	return n
} //}}}

func JsonEncodeBytes(data any) []byte { // {{{
	content, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return nil
	}

	return bytes.Replace(content, []byte("\n"), []byte(""), -1)
} // }}}

func JsonEncode(data any) string { // {{{
	return string(JsonEncodeBytes(data))
} // }}}

func JsonDecode(data any) any { // {{{
	var obj any
	err := json.Unmarshal(AsBytes(data), &obj)
	if err != nil {
		return nil
	}

	return convertFloat(obj)
} // }}}

// 格式化科学法表示的数字
func convertFloat(r any) any { // {{{
	switch val := r.(type) {
	case map[string]any:
		s := map[string]any{}
		for k, v := range val {
			s[k] = convertFloat(v)
		}
		return s
	case []any:
		s := []any{}
		for _, v := range val {
			s = append(s, convertFloat(v))
		}
		return s
	case float64:
		if float64(int(val)) == val {
			return int(val)
		}
		return val
	default:
		return r
	}
} // }}}

// 判断array/slice中是否存在某值
func InArray[T comparable](search T, s []T) bool { // {{{
	for _, v := range s {
		if search == v {
			return true
		}
	}

	return false
} // }}}

// 判断map中是否存在某值
func InMap[T, K comparable](search T, m map[K]T) bool { // {{{
	for _, v := range m {
		if search == v {
			return true
		}
	}

	return false
} // }}}

// array 去重, trim:去除首尾空格，删除空元素
func ArrayUnique[T comparable](arr []T) []T { // {{{
	check_uniq := map[T]int{}
	narray := []T{}
	for _, v := range arr {
		if _, ok := check_uniq[v]; !ok {
			narray = append(narray, v)
			check_uniq[v] = 1
		}
	}

	return narray
} // }}}

// array string 新增，并去重
func ArrayMerge[T comparable](arr []T, n ...[]T) []T { // {{{
	for _, v := range n {
		arr = append(arr, v...)
	}

	return ArrayUnique(arr)
} // }}}

// array string 删除
func ArrayRem[T comparable](arr []T, n T) []T { // {{{
	narr := []T{}
	for _, v := range arr {
		if v != n {
			narr = append(narr, v)
		}
	}

	return narr
} // }}}

func MD5(str string) string { // {{{
	h := md5.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

func MD5File(file string) (string, error) { // {{{
	str, err := FileGetContents(file)
	if err != nil {
		return "", err
	}

	return MD5(str), nil
} // }}}

func Sha1(str string) string { // {{{
	h := sha1.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
} // }}}

func Crc32(str string) int { // {{{
	ieee := crc32.NewIEEE()
	io.WriteString(ieee, str)
	return int(ieee.Sum32())
} // }}}

// 为字符串生成哈希值
func Hash(s string) uint64 { // {{{
	return xxhash.Sum64String(s)
} // }}}

// 为slice生成哈希值
func HashSlice(s []any) uint64 { // {{{
	var sb strings.Builder
	for i, k := range s {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(AsString(k))
	}

	return xxhash.Sum64String(sb.String())
} // }}}

// 为map生成哈希值
func HashMap(m map[string]any) uint64 { // {{{
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(AsString(m[k]))
		sb.WriteString("|")
	}

	return xxhash.Sum64String(sb.String())
} // }}}

// 拼接字符串
func Concat(str ...string) string { // {{{
	var builder strings.Builder

	for _, val := range str {
		builder.WriteString(val)
	}

	return builder.String()
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

func FormatTime(layout string, times ...any) string {
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
}

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
func Cost(start_time time.Time) int { //start_time=time.Now()
	return int(time.Now().Sub(start_time).Nanoseconds() / 1000 / 1000)
}

// 强制转换为bool
func AsBool(b interface{}, def ...bool) bool { // {{{
	if val, ok := b.(bool); ok {
		return val
	}

	val, ok := b.(string)
	if !ok {
		val = fmt.Sprint(b)
	}

	ret, err := strconv.ParseBool(val)
	if nil != err {
		if len(def) > 0 {
			ret = def[0]
		}
	}

	return ret
} // }}}

// 强制转换为int
func AsInt(num interface{}, def ...int) int { // {{{
	numint := 0
	switch val := num.(type) {
	case int:
		numint = val
	case string:
		numint = ToInt(val)
	case int64:
		numint = int(val)
	case int32:
		numint = int(val)
	case float64:
		numint = int(val)
	case float32:
		numint = int(val)
	case uint:
		numint = int(val)
	case uint32:
		numint = int(val)
	case uint64:
		numint = int(val)
	default:
		numint = ToInt(fmt.Sprint(num))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为int32
func AsInt32(num interface{}, def ...int32) int32 { // {{{
	var numint int32
	switch val := num.(type) {
	case int32:
		numint = val
	case int:
		numint = int32(val)
	case string:
		numint = int32(ToInt(val))
	case int64:
		numint = int32(val)
	case float64:
		numint = int32(val)
	case float32:
		numint = int32(val)
	case uint:
		numint = int32(val)
	case uint32:
		numint = int32(val)
	case uint64:
		numint = int32(val)
	default:
		numint = int32(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为int64
func AsInt64(num interface{}, def ...int64) int64 { // {{{
	var numint int64
	switch val := num.(type) {
	case int64:
		numint = val
	case int:
		numint = int64(val)
	case string:
		numint = ToInt64(val)
	case int32:
		numint = int64(val)
	case float64:
		numint = int64(val)
	case float32:
		numint = int64(val)
	case uint:
		numint = int64(val)
	case uint32:
		numint = int64(val)
	case uint64:
		numint = int64(val)
	default:
		numint = ToInt64(fmt.Sprint(num))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint
func AsUint(num interface{}, def ...uint) uint { // {{{
	var numint uint
	switch val := num.(type) {
	case uint:
		numint = val
	case uint32:
		numint = uint(val)
	case uint64:
		numint = uint(val)
	case int:
		numint = uint(val)
	case string:
		numint = uint(ToInt(val))
	case int64:
		numint = uint(val)
	case int32:
		numint = uint(val)
	case float64:
		numint = uint(val)
	case float32:
		numint = uint(val)
	default:
		numint = uint(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint32
func AsUint32(num interface{}, def ...uint32) uint32 { // {{{
	var numint uint32
	switch val := num.(type) {
	case uint32:
		numint = val
	case uint:
		numint = uint32(val)
	case uint64:
		numint = uint32(val)
	case int:
		numint = uint32(val)
	case string:
		numint = uint32(ToInt(val))
	case int32:
		numint = uint32(val)
	case int64:
		numint = uint32(val)
	case float64:
		numint = uint32(val)
	case float32:
		numint = uint32(val)
	default:
		numint = uint32(ToInt(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为uint64
func AsUint64(num interface{}, def ...uint64) uint64 { // {{{
	var numint uint64
	switch val := num.(type) {
	case uint64:
		numint = val
	case uint:
		numint = uint64(val)
	case uint32:
		numint = uint64(val)
	case int:
		numint = uint64(val)
	case string:
		numint = uint64(ToInt64(val))
	case int32:
		numint = uint64(val)
	case int64:
		numint = uint64(val)
	case float64:
		numint = uint64(val)
	case float32:
		numint = uint64(val)
	default:
		numint = uint64(ToInt64(fmt.Sprint(num)))
	}

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为float64
func AsFloat(num interface{}, def ...float64) float64 { // {{{
	return AsFloat64(num, def...)
} //}}}

// 强制转换为float32
func AsFloat32(num interface{}, def ...float32) float32 { // {{{
	var numint float32

	numint = float32(AsFloat64(num))

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为float64
func AsFloat64(num interface{}, def ...float64) float64 { // {{{
	var numint float64
	switch val := num.(type) {
	case float64:
		numint = val
	case int:
		numint = float64(val)
	case string:
		numint = ToFloat(val)
	case int32:
		numint = float64(val)
	case int64:
		numint = float64(val)
	case float32:
		numint = float64(val)
	default:
		numint = ToFloat(fmt.Sprint(num))
	}

	numint = ParseNaN(numint)

	if numint == 0 && len(def) > 0 {
		return def[0]
	}

	return numint
} // }}}

// 强制转换为string
func AsString(v any, def ...string) string { // {{{
	if v == nil {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprint(v)
	}
} // }}}

// 强制转换为bytes
func AsBytes(arg interface{}, def ...[]byte) []byte { // {{{
	if arg == nil {
		if len(def) > 0 {
			return def[0]
		}

		return nil
	}

	switch val := arg.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(AsString(arg))
	}
} // }}}

// string转换为int
func ToInt(num string) int { // {{{
	num = strings.TrimSpace(num)
	if num == "" {
		return 0
	}

	idx := strings.Index(num, ".")
	if idx > 0 {
		return int(ToFloat(num))
	}

	idx = strings.Index(num, "+")
	if idx > 0 && (num[idx-1] == 'e' || num[idx-1] == 'E') {
		return int(ToFloat(num))
	}

	numint, err := strconv.Atoi(num)
	if nil != err {
		return 0
	}
	return numint
} // }}}

// string转换为int32
func ToInt32(v string) int32 {
	return int32(ToInt(v))
}

// string转换为int64
func ToInt64(num string) int64 { // {{{
	num = strings.TrimSpace(num)
	if num == "" {
		return 0
	}

	numint, err := strconv.ParseInt(num, 10, 64)
	if nil != err {
		return 0
	}

	return numint
} // }}}

// NaN转换0
func ParseNaN(num float64) float64 { // {{{
	if math.IsNaN(num) || math.IsInf(num, 0) {
		num = 0
	}

	return num
} // }}}

// string转换为float64
func ToFloat(num string) float64 { // {{{
	return ToFloat64(num)
} // }}}

func ToFloat32(num string) float32 { // {{{
	return float32(ToFloat64(num))
} // }}}

func ToFloat64(num string) float64 { // {{{
	numfloat, err := strconv.ParseFloat(num, 64)
	if nil != err {
		return 0
	}

	return numfloat
} // }}}

// int转换为string
func ToString(num int) string { // {{{
	return strconv.Itoa(num)
} // }}}

// string的切片转换为int切片
func ToIntSlice(nums []string) []int { // {{{
	intnums := []int{}

	for _, v := range nums {
		intnums = append(intnums, ToInt(v))
	}

	return intnums
} // }}}

// string的切片转换为int32切片
func ToInt32Slice(nums []string) []int32 { // {{{
	intnums := []int32{}

	for _, v := range nums {
		intnums = append(intnums, int32(ToInt(v)))
	}

	return intnums
} // }}}

// string的切片转换为int64切片
func ToInt64Slice(nums []string) []int64 { // {{{
	intnums := []int64{}

	for _, v := range nums {
		intnums = append(intnums, ToInt64(v))
	}

	return intnums
} // }}}

// int的切片转换为string切片
func ToStringSlice(nums []int) []string { // {{{
	strnums := []string{}

	for _, v := range nums {
		strnums = append(strnums, strconv.Itoa(v))
	}

	return strnums
} // }}}

// interface{}的切片转换为int切片
func AsIntSlice(v interface{}) []int { // {{{
	if v == nil {
		return nil
	}

	var nums []int

	switch val := v.(type) {
	case []int:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt(i))
		}
	case []int32:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []int64:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为int32切片
func AsInt32Slice(v interface{}) []int32 { // {{{
	if v == nil {
		return nil
	}

	var nums []int32

	switch val := v.(type) {
	case []int32:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt32(i))
		}
	case []int:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []int64:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int32(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt32(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为int64切片
func AsInt64Slice(v interface{}) []int64 { // {{{
	if v == nil {
		return nil
	}

	var nums []int64

	switch val := v.(type) {
	case []int64:
		return val
	case []string:
		for _, i := range val {
			nums = append(nums, ToInt64(i))
		}
	case []int:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []int32:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []float64:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	case []float32:
		for _, i := range val {
			nums = append(nums, int64(i))
		}
	default:
		value := reflect.ValueOf(v)
		value = value.Convert(value.Type())
		typ := reflect.TypeOf(v).Kind()

		if typ == reflect.Slice || typ == reflect.Array {
			for i := 0; i < value.Len(); i++ {
				nums = append(nums, AsInt64(value.Index(i).Interface()))
			}
		}
	}

	return nums
} // }}}

// interface{}的切片转换为string切片
func AsStringSlice(v any) (strs []string) { // {{{
	if v == nil {
		return
	}

	switch val := v.(type) {
	case []string:
		return val
	case []any:
		for _, i := range val {
			strs = append(strs, AsString(i))
		}
	case [][]byte:
		for _, i := range val {
			strs = append(strs, string(i))
		}
	case []int:
		for _, i := range val {
			strs = append(strs, strconv.Itoa(i))
		}
	case []int8:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int16:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int32:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(int64(i), 10))
		}
	case []int64:
		for _, i := range val {
			strs = append(strs, strconv.FormatInt(i, 10))
		}
	case []uint:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint8:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint16:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint32:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(uint64(i), 10))
		}
	case []uint64:
		for _, i := range val {
			strs = append(strs, strconv.FormatUint(i, 10))
		}
	case []float32:
		for _, i := range val {
			strs = append(strs, strconv.FormatFloat(float64(i), 'f', -1, 32))
		}
	case []float64:
		for _, i := range val {
			strs = append(strs, strconv.FormatFloat(i, 'f', -1, 64))
		}
	case []bool:
		for _, i := range val {
			strs = append(strs, strconv.FormatBool(i))
		}
	case []fmt.Stringer:
		for _, i := range val {
			strs = append(strs, i.String())
		}

	default:
	}

	return
} // }}}

// 将[]T 转换为[]any
func AsSlice[T any](arr []T) []any { // {{{
	narr := []any{}
	for _, v := range arr {
		narr = append(narr, v)
	}

	return narr
} // }}}

// []int, []string, []interface, 连接成字符串
func Join(v interface{}, seps ...string) string { // {{{
	arr := AsStringSlice(v)
	if arr == nil {
		return ""
	}

	sep := ","
	if len(seps) > 0 {
		sep = seps[0]
	}

	return strings.Join(arr, sep)
} // }}}

func AsMap(a any) MAP { // {{{
	if m, ok := a.(MAP); ok {
		return m
	}

	if m, ok := a.(Mapper); ok {
		return m.ToMap()
	}

	data := MAP{}

	objVal := reflect.ValueOf(a)
	kind := objVal.Kind()

	if kind == reflect.Ptr {
		objVal = objVal.Elem()
		kind = objVal.Kind()
	}

	switch kind {
	case reflect.Map:
		keys := objVal.MapKeys()
		for _, k := range keys {
			data[AsString(k)] = objVal.MapIndex(k).Interface()
		}
	case reflect.Struct:
		t := objVal.Type()
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("json")
			if tag == "-" || tag == "nil" {
				continue
			}

			if tag == "" {
				tag = strings.ToLower(t.Field(i).Name)
			}

			data[tag] = objVal.Field(i).Interface()
		}
	}

	return data
} // }}}

func AsStringMap(a any) MAPS { // {{{
	m := AsMap(a)
	n := MAPS{}
	for k, v := range m {
		n[k] = AsString(v)
	}

	return n
} // }}}

func AsIntMap(a any) MAPI { // {{{
	m := AsMap(a)
	n := MAPI{}
	for k, v := range m {
		n[k] = AsInt(v)
	}

	return n
} // }}}

func AsMapSlice(a any) []MAP { // {{{
	n := []MAP{}
	if s, ok := a.([]any); ok {
		for _, v := range s {
			n = append(n, AsMap(v))
		}
	}

	return n
} // }}}

// 判断ver是否大于oldver
func VersionCompare(ver, oldver string) bool { // {{{
	vs1 := strings.Split(ver, ".")
	vs2 := strings.Split(oldver, ".")
	len1 := len(vs1)
	len2 := len(vs2)

	l := len1
	if len1 < len2 {
		l = len2
	}

	for i := 0; i < l; i++ {
		vs1 = append(vs1, "")
		vs2 = append(vs2, "")

		v1 := ToInt(vs1[i])
		v2 := ToInt(vs2[i])
		if v1 > v2 {
			return true
		} else if v1 < v2 {
			return false
		}
	}

	return false
} // }}}

// x.x.x.x格式的IP转换为数字
func Ip2long(ipstr string) (ip int) { // {{{
	r := `^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`
	reg, err := regexp.Compile(r)
	if err != nil {
		return
	}
	ips := reg.FindStringSubmatch(ipstr)
	if ips == nil {
		return
	}

	ip1, _ := strconv.Atoi(ips[1])
	ip2, _ := strconv.Atoi(ips[2])
	ip3, _ := strconv.Atoi(ips[3])
	ip4, _ := strconv.Atoi(ips[4])

	if ip1 > 255 || ip2 > 255 || ip3 > 255 || ip4 > 255 {
		return
	}

	ip += int(ip1 * 0x1000000)
	ip += int(ip2 * 0x10000)
	ip += int(ip3 * 0x100)
	ip += int(ip4)

	return
} // }}}

// 数字格式的IP转换为x.x.x.x
func Long2ip(ip int) string { // {{{
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip<<8>>24, ip<<16>>24, ip<<24>>24)
} // }}}

func FileGetContents(path string, options ...interface{}) (string, error) { // {{{
	//远程文件
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") { //{{{
		c := NewHttpClient(3)
		header := http.Header{}
		if len(options) > 0 {
			if hd, ok := options[0].(http.Header); ok {
				header = hd
			}
		}

		httpres, err := c.Get(path, header)
		if nil != err || httpres.GetCode() != 200 {
			return "", fmt.Errorf("http code: %v", httpres.GetCode())
		}

		return httpres.GetResponse(), nil
	} // }}}

	//本地文件
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
} // }}}

func IsFile(path string) (bool, error) { // {{{
	m, err := os.Stat(path)
	if err == nil {
		return m.Mode().IsRegular(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
} // }}}

func IsDir(path string) (bool, error) { // {{{
	m, err := os.Stat(path)
	if err == nil {
		return m.IsDir(), nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
} // }}}

// 检测是否终端设备
var is_term string

func IsTerm() bool { // {{{
	if is_term == "" {
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			is_term = "Y"
		} else {
			is_term = "N"
		}
	}

	return is_term == "Y"
} // }}}

func RedString(s string) string {
	return Concat("\033[31m", s, "\033[0m")
}

func GreenString(s string) string {
	return Concat("\033[32m", s, "\033[0m")
}

func YellowString(s string) string {
	return Concat("\033[33m", s, "\033[0m")
}

// 便于调式时直接使用
func Println(s ...any) { // {{{
	if IsTerm() {
		fmt.Println(s...)
	}
} // }}}

func Printf(f string, s ...any) { // {{{
	if IsTerm() {
		fmt.Printf(f, s...)
	}
} // }}}
