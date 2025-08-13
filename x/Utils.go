package x

import (
	"bytes"
	"fmt"
	json "github.com/bytedance/sonic" //"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
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
func RandUint32() uint32 { // {{{
	state := randPool.Get().(*randState)
	defer randPool.Put(state)

	// Xorshift算法核心
	state.seed ^= state.seed << 13
	state.seed ^= state.seed >> 17
	state.seed ^= state.seed << 5
	return state.seed
} // }}}

// 返回[0,n)范围内的随机整数, 替换math.rand.Intn
func RandIntn(n int) int {
	return int(RandUint32()) % n
}

func rand(size int, kind int) []byte { // {{{
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	//rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = RandIntn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + RandIntn(scope))
	}
	return result
} // }}}

// 随机字符串
func RandStr(size int, kind ...int) string { // {{{
	k := RAND_KIND_ALL
	if len(kind) > 0 {
		k = kind[0]
	}
	return string(rand(size, k))
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

func JsonEncodeToBytes(data any) []byte { // {{{
	content, err := json.MarshalIndent(data, "", "")
	if err != nil {
		return nil
	}

	return bytes.Replace(content, []byte("\n"), []byte(""), -1)
} // }}}

func JsonEncode(data any) string { // {{{
	return string(JsonEncodeToBytes(data))
} // }}}

func JsonDecode(data any) any { // {{{
	var obj any
	err := json.Unmarshal(AsBytes(data), &obj)
	if err != nil {
		return nil
	}

	return convertFloat(obj)
} // }}}

func GetJsonEncoder(w io.Writer) json.Encoder { // {{{
	return json.ConfigDefault.NewEncoder(w)
} // }}}

func GetJsonDecoder(w io.Reader) json.Decoder { // {{{
	return json.ConfigDefault.NewDecoder(w)
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

// array 新增，并去重
func ArrayMerge[T comparable](arr []T, n ...[]T) []T { // {{{
	for _, v := range n {
		arr = append(arr, v...)
	}

	return ArrayUnique(arr)
} // }}}

// array 删除
func ArrayRem[T comparable](arr []T, n T) []T { // {{{
	narr := []T{}
	for _, v := range arr {
		if v != n {
			narr = append(narr, v)
		}
	}

	return narr
} // }}}

// 拼接字符串
func Concat(str ...string) string { // {{{
	var builder strings.Builder

	for _, val := range str {
		builder.WriteString(val)
	}

	return builder.String()
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

// 打印内存使用情况
func PrintMemUsage() { // {{{
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", m.Alloc/1024/1024)
	fmt.Printf("\tTotalAlloc = %v MiB", m.TotalAlloc/1024/1024)
	fmt.Printf("\tSys = %v MiB", m.Sys/1024/1024)
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
} // }}}

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
