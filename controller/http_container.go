package controller

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nyxless/nyx/x"
)

const defaultBufWriterSize = 4096

type httpContainer struct {
	Controller
	W           http.ResponseWriter
	R           *http.Request
	RBody       []byte
	Form        url.Values
	JsonForm    x.MAP
	FormMaps    map[string]map[string]any
	MaxPostSize int64 //post 表单大小

	bufWriter     *bufio.Writer // 流式输出缓冲
	bufWriterSize int           // 缓冲区大小，<=0 时使用 defaultBufWriterSize
}

func (h *httpContainer) Prepare() { // {{{
	if h.R.Method == "GET" || h.R.Method == "HEAD" || h.R.Method == "OPTIONS" {
		h.Form = h.R.URL.Query()
		return
	}

	contentType := h.R.Header.Get("Content-Type")

	var err error
	if strings.Contains(contentType, "application/json") {
		h.Form = h.R.URL.Query()
		err = x.GetJsonDecoder(h.R.Body).Decode(&h.JsonForm)
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err = h.R.ParseForm(); err == nil {
			h.Form = h.R.Form
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		if h.MaxPostSize == 0 {
			h.MaxPostSize = x.ConfMaxPostSize
		}

		if err = h.R.ParseMultipartForm(h.MaxPostSize); err == nil {
			h.Form = h.R.Form
		}
	}

	if err != nil {
		http.Error(h.W, err.Error(), http.StatusBadRequest)
	}
} // }}}

func (h *httpContainer) GetParams() x.MAP { // {{{
	params := x.MAP{}
	if h.JsonForm != nil {
		params = h.JsonForm
	}

	for k, v := range h.Form {
		if _, ok := params[k]; !ok && len(v) > 0 {
			params[k] = strings.TrimSpace(v[0])
		}
	}

	return params
} // }}}

func (h *httpContainer) GetParam(key string) any { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return v
		}
	}

	if v := h.Form[key]; len(v) > 0 {
		return strings.TrimSpace(v[0])
	}

	return nil
} // }}}

func (h *httpContainer) GetString(key string, defaultValues ...string) string { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsString(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return strings.TrimSpace(v[0])
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ""
} // }}}

func (h *httpContainer) GetInt(key string, defaultValues ...int) int { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetInt8(key string, defaultValues ...int8) int8 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt8(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt8(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetInt16(key string, defaultValues ...int16) int16 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt16(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt16(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetInt32(key string, defaultValues ...int32) int32 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt32(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt32(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetInt64(key string, defaultValues ...int64) int64 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt64(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToInt64(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetUint(key string, defaultValues ...uint) uint { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsUint(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToUint(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetUint8(key string, defaultValues ...uint8) uint8 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsUint8(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToUint8(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetUint16(key string, defaultValues ...uint16) uint16 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsUint16(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToUint16(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetUint32(key string, defaultValues ...uint32) uint32 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsUint32(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToUint32(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetUint64(key string, defaultValues ...uint64) uint64 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsUint64(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if n, ok := x.ToUint64(v[0]); ok {
			return n
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetBool(key string, defaultValues ...bool) bool { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsBool(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if ret, err := strconv.ParseBool(v[0]); err == nil {
			return ret
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return false
} // }}}

func (h *httpContainer) GetFloat(key string, defaultValues ...float64) float64 { // {{{
	return h.GetFloat64(key, defaultValues...)
} // }}}

func (h *httpContainer) GetFloat32(key string, defaultValues ...float32) float32 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsFloat32(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if ret, err := strconv.ParseFloat(v[0], 64); err == nil {
			return float32(ret)
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetFloat64(key string, defaultValues ...float64) float64 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsFloat64(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		if ret, err := strconv.ParseFloat(v[0], 64); err == nil {
			return ret
		}
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return 0
} // }}}

func (h *httpContainer) GetJsonMap(key string) x.MAP { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsMap(x.JsonDecode(v))
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return x.AsMap(x.JsonDecode(v[0]))
	}

	return x.MAP{}
} // }}}

func (h *httpContainer) GetSlice(key string, defaultValues ...[]any) []any { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsSlice(v, defaultValues...)
		}
	}

	collect := func(v []string) []any {
		ret := make([]any, 0, len(v))
		for _, s := range v {
			ret = append(ret, strings.TrimSpace(s))
		}

		return ret
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		// 格式: key=123&key=456
		if len(v) > 1 {
			return collect(v)
		}

		// 格式: key=123,456
		return x.Split(v[0])
	}

	// 格式: key[]=123&key[]=456
	key = key + "[]"
	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return collect(v)
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []any{}
} // }}}

func (h *httpContainer) GetStringSlice(key string, defaultValues ...[]string) []string { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsStringSlice(v, defaultValues...)
		}
	}

	collect := func(v []string) []string {
		ret := make([]string, 0, len(v))
		for _, s := range v {
			ret = append(ret, strings.TrimSpace(s))
		}

		return ret
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		// 格式: key=123&key=456
		if len(v) > 1 {
			return collect(v)
		}

		// 格式: key=123,456
		return x.SplitString(v[0])
	}

	// 格式: key[]=123&key[]=456
	key = key + "[]"
	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return collect(v)
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []string{}
} // }}}

func (h *httpContainer) GetIntSlice(key string, defaultValues ...[]int) []int { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsIntSlice(v, defaultValues...)
		}
	}

	collect := func(v []string) []int {
		ret := make([]int, 0, len(v))
		for _, s := range v {
			ret = append(ret, x.AsInt(strings.TrimSpace(s)))
		}

		return ret
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		// 格式: key=123&key=456
		if len(v) > 1 {
			return collect(v)
		}

		// 格式: key=123,456
		return x.SplitInt(v[0])
	}

	// 格式: key[]=123&key[]=456
	key = key + "[]"
	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return collect(v)
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []int{}
} // }}}

func (h *httpContainer) GetInt32Slice(key string, defaultValues ...[]int32) []int32 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt32Slice(v, defaultValues...)
		}
	}

	collect := func(v []string) []int32 {
		ret := make([]int32, 0, len(v))
		for _, s := range v {
			ret = append(ret, x.AsInt32(strings.TrimSpace(s)))
		}

		return ret
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		// 格式: key=123&key=456
		if len(v) > 1 {
			return collect(v)
		}

		// 格式: key=123,456
		return x.SplitInt32(v[0])
	}

	// 格式: key[]=123&key[]=456
	key = key + "[]"
	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return collect(v)
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []int32{}
} // }}}

func (h *httpContainer) GetInt64Slice(key string, defaultValues ...[]int64) []int64 { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsInt64Slice(v, defaultValues...)
		}
	}

	collect := func(v []string) []int64 {
		ret := make([]int64, 0, len(v))
		for _, s := range v {
			ret = append(ret, x.AsInt64(strings.TrimSpace(s)))
		}

		return ret
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		// 格式: key=123&key=456
		if len(v) > 1 {
			return collect(v)
		}

		// 格式: key=123,456
		return x.SplitInt64(v[0])
	}

	// 格式: key[]=123&key[]=456
	key = key + "[]"
	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return collect(v)
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []int64{}
} // }}}

func (h *httpContainer) GetMapSlice(key string) []x.MAP { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsMapSlice(v)
		}
	}

	return []x.MAP{}
} // }}}

func (h *httpContainer) GetBytes(key string, defaultValues ...[]byte) []byte { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsBytes(v, defaultValues...)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return []byte(v[0])
	}

	if len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return []byte{}
} // }}}

func (h *httpContainer) GetMap(key string) x.MAP { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsMap(v)
		}
	}

	if h.FormMaps == nil {
		h.FormMaps = extractFormMaps(h.Form)
	}

	if v, ok := h.FormMaps[key]; ok {
		return v
	}

	return x.MAP{}
} // }}}

func (h *httpContainer) GetStringMap(key string) x.MAPS { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsStringMap(v)
		}
	}

	if h.FormMaps == nil {
		h.FormMaps = extractFormMaps(h.Form)
	}

	if v, ok := h.FormMaps[key]; ok {
		return x.AsStringMap(v)
	}

	return x.MAPS{}
} // }}}

func (h *httpContainer) GetIntMap(key string) x.MAPI { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsIntMap(v)
		}
	}

	if h.FormMaps == nil {
		h.FormMaps = extractFormMaps(h.Form)
	}

	if v, ok := h.FormMaps[key]; ok {
		return x.AsIntMap(v)
	}

	return x.MAPI{}
} // }}}

func extractFormMaps(form url.Values) map[string]map[string]any { // {{{
	result := make(map[string]map[string]any)
	for key, values := range form {
		if len(values) == 0 {
			continue
		}

		if strings.Contains(key, "[") || strings.Contains(key, ".") {
			extract(result, key, values[0])
		}
	}

	return result
} // }}}

func extract(target map[string]map[string]any, key, value string) { // {{{
	key = strings.ReplaceAll(key, "]", "")
	key = strings.ReplaceAll(key, "[", ".")

	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return
	}

	baseKey := parts[0]
	innerParts := parts[1:]

	if _, exists := target[baseKey]; !exists {
		target[baseKey] = make(map[string]any)
	}

	setValue(target[baseKey], innerParts, value)
} // }}}

func setValue(current map[string]any, parts []string, value string) { // {{{
	if len(parts) == 1 {
		current[parts[0]] = value
		return
	}

	part := parts[0]
	var nextMap map[string]any

	if existing, exists := current[part]; exists {
		if m, ok := existing.(map[string]any); ok {
			nextMap = m
		} else {
			nextMap = make(map[string]any)
			current[part] = nextMap
		}
	} else {
		nextMap = make(map[string]any)
		current[part] = nextMap
	}

	setValue(nextMap, parts[1:], value)
} // }}}

func (h *httpContainer) GetTime(key string) time.Time { // {{{
	if h.JsonForm != nil {
		if v, ok := h.JsonForm[key]; ok {
			return x.AsTime(v)
		}
	}

	if v, ok := h.Form[key]; ok && len(v) > 0 {
		return x.AsTime(v[0])
	}

	return time.Time{}
} // }}}

func (h *httpContainer) GetIp() (ip string) { // {{{
	h.Ctx, ip = x.GetHttpCtxIp(h.Ctx, h.R)
	return
} // }}}

func (h *httpContainer) GetHeader(key string, defaultValues ...string) (ret string) { // {{{
	ret = h.R.Header.Get(key)

	if ret == "" && len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ret
} // }}}

func (h *httpContainer) GetHeaders() (ret x.MAPS) { // {{{
	ret = make(x.MAPS, len(h.R.Header))
	for k, v := range h.R.Header {
		ret[strings.ToLower(k)] = v[0]
	}

	return ret
} // }}}

func (h *httpContainer) SetHeader(key, val string) { // {{{
	h.W.Header().Set(key, val)
} // }}}

func (h *httpContainer) SetHeaders(headers x.MAPS) { // {{{
	for k, v := range headers {
		h.W.Header().Set(k, v)
	}
} // }}}

// 接口正常输出，json 格式
func (h *httpContainer) Render(data ...any) { // {{{
	var retdata any
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(x.MAP)
	}

	h.render(x.ErrSuc.GetCode(), x.ErrSuc.GetMessage(), retdata)
} // }}}

// 接口异常输出, json 格式
func (h *httpContainer) RenderError(err any) { // {{{
	errno, errmsg, retdata := h.GetErrorResponse(err)
	h.render(errno, errmsg, retdata)
} // }}}

// 输出JSON
func (h *httpContainer) render(errno int32, errmsg string, retdata any) { // {{{
	resData := h.RenderResponser(errno, errmsg, retdata)

	h.W.Header().Set("Content-Type", "application/json;charset=UTF-8")
	data := x.JsonEncodeToBytes(resData)

	h.W.Write(data)
} // }}}

// SetBufWriterSize 设置流式输出缓冲区大小。
// 必须在第一次调用 StreamWriter / RenderStream / FlushStream 之前调用，
// 否则已创建的 bufWriter 不会被重新初始化。
func (h *httpContainer) SetBufWriterSize(size int) { // {{{
	if size > 0 {
		h.bufWriterSize = size
	}
} // }}}

// getBufWriter 懒加载缓冲写入器，整个请求生命周期内复用同一个实例。
func (h *httpContainer) getBufWriter() *bufio.Writer { // {{{
	if h.bufWriter == nil {
		size := h.bufWriterSize
		if size <= 0 {
			size = defaultBufWriterSize
		}
		h.bufWriter = bufio.NewWriterSize(h.W, size)
	}

	return h.bufWriter
} // }}}

// flushBufWriter 内部收尾 flush，忽略错误（供 Final 使用）
func (h *httpContainer) flushBufWriter() { // {{{
	if h.bufWriter == nil {
		return
	}

	if err := h.bufWriter.Flush(); err != nil {
		return
	}

	if flusher, ok := h.W.(http.Flusher); ok {
		flusher.Flush()
	}
} // }}}

// StreamWriter 返回框架管理的缓冲写入器，供业务自行组装
// csv.Writer、json.Encoder 等流式输出使用。
//
// 业务调用示例：
//
//	cw := csv.NewWriter(h.StreamWriter())
//	cw.Write([]string{"id", "name"})
//	cw.Flush()
//	h.FlushStream()
func (h *httpContainer) StreamWriter() io.Writer { // {{{
	return h.getBufWriter()
} // }}}

// FlushStream 将缓冲区数据推送到客户端，并触发 http.Flusher。
// 配合 StreamWriter 使用；RenderStream 内部也会调用。
func (h *httpContainer) FlushStream() error { // {{{
	select {
	case <-h.R.Context().Done():
		// 客户端已断开
		return fmt.Errorf("Client has terminated the request!")
	default:
		// 继续处理
	}

	if h.bufWriter != nil {
		if err := h.bufWriter.Flush(); err != nil {
			return err
		}
	}

	flusher, ok := h.W.(http.Flusher)
	if !ok {
		return fmt.Errorf("Streaming unsupported")
	}

	flusher.Flush()

	return nil
} // }}}

// RenderStream 输出 HTTP 流（一次性写入 []byte 并 flush）。
// 内部基于 StreamWriter + FlushStream 实现，与业务侧组装
// csv.Writer 等共享同一个缓冲区。
func (h *httpContainer) RenderStream(data any) error { // {{{
	stream, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("render data type is not []byte!")
	}

	if h.W == nil || stream == nil {
		return nil
	}

	_, err := h.StreamWriter().Write(stream)
	return err
} // }}}
