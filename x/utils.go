package x

import (
	"bytes"
	"encoding/gob"
	"fmt"
	json "github.com/bytedance/sonic" //"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
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

// 返回[min,max)范围内的随机整数
func RandInt(min, max int) int { // {{{
	if min >= max {
		return min
	}

	rangeSize := max - min
	if rangeSize <= 0 {
		return RandIntn(1<<31) + min
	}

	return RandIntn(rangeSize) + min
} // }}}

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

func JsonEncodeToBytes(data any) []byte { // {{{
	content, err := json.MarshalIndent(data, "", "")
	if err != nil {
		if Debug {
			Notice("JsonEncodeToBytes err:", err)
		}

		return nil
	}

	return bytes.Replace(content, []byte("\n"), []byte(""), -1)
} // }}}

func JsonDecodeBytes(data []byte) any { // {{{
	var obj any
	err := json.Unmarshal(data, &obj)
	if err != nil {
		if Debug {
			Notice("JsonDecodeBytes err:", err)
		}

		return nil
	}

	return convertFloat(obj)
} // }}}

func JsonEncode(data any) string { // {{{
	return string(JsonEncodeToBytes(data))
} // }}}

func JsonDecode(data any) any { // {{{
	return JsonDecodeBytes(AsBytes(data))
} // }}}

func JsonUnmarshal(data any, obj any) error { // {{{
	return json.Unmarshal(AsBytes(data), obj)
} // }}}

func GetJsonEncoder(w io.Writer) json.Encoder { // {{{
	return json.ConfigDefault.NewEncoder(w)
} // }}}

func GetJsonDecoder(r io.Reader) json.Decoder { // {{{
	return json.ConfigDefault.NewDecoder(r)
} // }}}

func GobEncode(data any) ([]byte, error) { // {{{
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
} // }}}

func GobDecode(data []byte, obj any) error { // {{{
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	return decoder.Decode(obj)
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

// 安全异步执行函数, 增加错误捕获，防止主协程 panic
func Go(fn func()) { // {{{
	go func() {
		defer func() {
			if err := recover(); err != nil {
				var errmsg string
				switch errinfo := err.(type) {
				case *Error:
					errmsg = errinfo.GetMessage()
				case error:
					errmsg = errinfo.Error()
					errmsg = errmsg + "\n" + string(debug.Stack())
				default:
					errmsg = fmt.Sprint(errinfo)
				}

				if Logger != nil {
					Logger.Error(errmsg)
				} else {
					log.Println("async_func_err", errmsg)
				}
			}

		}()

		fn()
	}()
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

func Colorize(text string, color string) string { // {{{
	colorCodes := map[string]string{
		"reset":     "\033[0m",
		"red":       "\033[31m",
		"green":     "\033[32m",
		"yellow":    "\033[33m",
		"blue":      "\033[34m",
		"magenta":   "\033[35m",
		"cyan":      "\033[36m",
		"white":     "\033[37m",
		"bold":      "\033[1m",
		"underline": "\033[4m",
	}

	// 处理组合颜色如 "green+bold"
	parts := strings.Split(color, "+")
	var codes []string
	for _, part := range parts {
		if code, ok := colorCodes[part]; ok {
			codes = append(codes, code)
		}
	}

	if len(codes) == 0 {
		return text
	}

	return strings.Join(codes, "") + text + colorCodes["reset"]
} // }}}

func Warn(msgs ...any) { // {{{
	if IsTerm() {
		for i, v := range msgs {
			if i == 0 {
				msgs[0] = Colorize(AsString(v), "red+bold")
			} else {
				msgs[i] = Colorize(AsString(v), "red")
			}
		}
	}

	log.Println(msgs...)
} // }}}

func Warnf(str string, msgs ...any) { // {{{
	if len(msgs) > 0 {
		str = fmt.Sprintf(str, msgs...)
	}

	if IsTerm() {
		str = Colorize(str, "red")
	}

	log.Println(str)
} // }}}

func Notice(msgs ...any) { // {{{
	if IsTerm() {
		for i, v := range msgs {
			if i == 0 {
				msgs[0] = Colorize(AsString(v), "yellow+bold")
			} else {
				msgs[i] = Colorize(AsString(v), "yellow")
			}
		}
	}

	log.Println(msgs...)
} // }}}

func Noticef(str string, msgs ...any) { // {{{
	if len(msgs) > 0 {
		str = fmt.Sprintf(str, msgs...)
	}

	if IsTerm() {
		str = Colorize(str, "yellow")
	}

	log.Println(str)
} // }}}

func Success(msgs ...any) { // {{{
	if IsTerm() && len(msgs) > 0 {
		for i, v := range msgs {
			if i == 0 {
				msgs[0] = Colorize(AsString(v), "green+bold")
			} else {
				msgs[i] = Colorize(AsString(v), "green")
			}
		}
	}

	log.Println(msgs...)
} // }}}

func Successf(str string, msgs ...any) { // {{{
	if len(msgs) > 0 {
		str = fmt.Sprintf(str, msgs...)
	}

	if IsTerm() {
		str = Colorize(str, "green")
	}

	log.Println(str)
} // }}}

func Info(msgs ...any) { // {{{
	if IsTerm() && len(msgs) > 0 {
		for i, v := range msgs {
			if i == 0 {
				msgs[0] = Colorize(AsString(v), "cyan+bold")
			} else {
				msgs[i] = Colorize(AsString(v), "cyan")
			}
		}
	}
	log.Println(msgs...)
} // }}}

func Infof(str string, msgs ...any) { // {{{
	if len(msgs) > 0 {
		str = fmt.Sprintf(str, msgs...)
	}

	if IsTerm() {
		str = Colorize(str, "cyan")
	}

	log.Println(str)
} // }}}

// 便于调式时直接使用
func Println(s ...any) { // {{{
	log.Println(s...)
} // }}}

func Printf(f string, s ...any) { // {{{
	log.Printf(f, s...)
} // }}}

func Panic(s any) { // {{{
	errmsg := fmt.Sprint("Error: ", s)

	if IsTerm() {
		errmsg = Colorize(errmsg, "red")
	}

	panic(errmsg)
} // }}}
