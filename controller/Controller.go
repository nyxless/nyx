package controller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/nyxless/nyx/x"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type iRequest struct {
	Form    url.Values
	RpcForm x.MAP
}

const (
	HTTP_MODE = iota
	RPC_MODE
	CLI_MODE
)

type Controller struct {
	W             http.ResponseWriter
	R             *http.Request
	RBody         []byte
	IR            *iRequest
	Ctx           context.Context
	startTime     time.Time
	Mode          int
	rpcInHeaders  metadata.MD
	rpcOutHeaders x.MAPS
	rpcContent    x.MAP
	Controller    string
	Action        string
	logParams     x.MAP    //需要额外记录在日志中的参数
	logOmitParams []string //不希望记录在日志中的参数
	Tpl           *x.Template
	maxPostSize   int64 //post 表单大小
}

// 默认的初始化方法，可通过在项目中重写此方法实现公共入口方法
func (c *Controller) Init() {}

func (c *Controller) Prepare(w http.ResponseWriter, r *http.Request, controller, action string) { // {{{
	c.W = w
	c.R = r

	if x.Conf_template_enabled {
		c.Tpl = x.NewTemplate()
	}

	c.RBody, _ = c.getRequestBody(r)

	c.maxPostSize = x.Conf_max_post_size

	r.ParseMultipartForm(c.maxPostSize)

	c.IR = &iRequest{Form: r.Form}
	c.prepare(context.Background(), HTTP_MODE, controller, action)

	// guid 用于日志追踪，可由客户端生成, 依次检查: 请求参数 -> header -> 生成
	guid := c.GetString("guid", c.GetHeader("guid", x.RandStr(32)))

	c.Ctx = context.WithValue(c.Ctx, "guid", guid)
} // }}}

func (c *Controller) getRequestBody(r *http.Request) ([]byte, error) { // {{{
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(buf))
	return buf, nil
} // }}}

func (c *Controller) PrepareRpc(params x.MAP, ctx context.Context, controller, action string) { // {{{
	c.IR = &iRequest{RpcForm: params}
	c.prepare(ctx, RPC_MODE, controller, action)

	//rpc 接口鉴权
	appid := c.GetHeader("appid")
	secret := c.GetHeader("secret")

	x.Interceptor(secret == x.Conf_rpc_auth[appid], x.ERR_RPCAUTH, appid)
} // }}}

func (c *Controller) PrepareCli(params url.Values, controller, action string) { // {{{
	c.IR = &iRequest{Form: params}
	c.prepare(context.Background(), CLI_MODE, controller, action)
} // }}}

func (c *Controller) prepare(ctx context.Context, mode int, controller, action string) { // {{{
	c.startTime = time.Now()
	c.Mode = mode
	c.Controller = controller
	c.Action = action
	c.Ctx = ctx
} // }}}

// 以下 GetX 方法用于获取参数
func (c *Controller) GetCookie(key string, defaultValues ...string) string { // {{{
	ret := x.GetCookie(c.R, key)
	if ret == "" && len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ret
} // }}}

func (c *Controller) GetHeader(key string, defaultValues ...string) (ret string) { // {{{
	if HTTP_MODE == c.Mode {
		ret = c.R.Header.Get(key)
	} else if RPC_MODE == c.Mode {
		if c.rpcInHeaders == nil {
			c.rpcInHeaders, _ = metadata.FromIncomingContext(c.Ctx)
		}

		if c.rpcInHeaders != nil {
			if v, ok := c.rpcInHeaders[key]; ok {
				ret = v[0]
			}
		}
	}

	if ret == "" && len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ret
} // }}}

func (c *Controller) getFormValue(key string) string { // {{{
	if RPC_MODE == c.Mode {
		if c.IR.RpcForm == nil {
			return ""
		}

		if val, ok := c.IR.RpcForm[key]; ok {
			if v, ok := val.(string); ok {
				return v
			} else if b, ok := val.([]byte); ok {
				return string(b)
			}

			return ""
		}
	} else {
		if c.IR.Form == nil {
			return ""
		}

		if vs := c.IR.Form[key]; len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
	}

	return ""
} // }}}

// 获取参数, 默认string类型
func (c *Controller) GetParam(key string, defaultValues ...string) string { // {{{
	return c.GetString(key, defaultValues...)
} // }}}

// 获取string类型参数
func (c *Controller) GetString(key string, defaultValues ...string) string { // {{{
	ret := c.getFormValue(key)
	if ret == "" {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

// 获取bytes类型参数
func (c *Controller) GetBytes(key string, defaultValues ...[]byte) []byte { // {{{
	if RPC_MODE == c.Mode {
		if c.IR.RpcForm == nil {
			if len(defaultValues) > 0 {
				return defaultValues[0]
			}

			return nil
		}

		if val, ok := c.IR.RpcForm[key]; ok {
			if b, ok := val.([]byte); ok {
				return b
			} else if v, ok := val.(string); ok {
				return []byte(v)
			}

			return nil
		}
	} else {
		if c.IR.Form == nil {
			if len(defaultValues) > 0 {
				return defaultValues[0]
			}

			return nil
		}

		if vs := c.IR.Form[key]; len(vs) > 0 {
			return []byte(vs[0])
		}
	}

	return nil
} // }}}

// 获取指定字符连接的字符串并转换成[]string
func (c *Controller) GetSlice(key string, separators ...string) []string { //{{{
	value := c.GetString(key)
	if "" == value {
		return nil
	}

	separator := ","
	if len(separators) > 0 {
		separator = separators[0]
	}

	slice := []string{}
	for _, part := range strings.Split(value, separator) {
		slice = append(slice, strings.Trim(part, " \r\t\v"))
	}

	return slice
} // }}}

// 获取指定字符连接的字符串并转换成[]int
func (c *Controller) GetIntSlice(key string, separators ...string) []int { //{{{
	slice := c.GetSlice(key, separators...)

	if nil == slice {
		return nil
	}

	intslice := []int{}
	for _, val := range slice {
		if val, err := strconv.Atoi(val); nil == err {
			intslice = append(intslice, val)
		}
	}

	return intslice
} // }}}

// 获取所有参数
func (c *Controller) GetParams() x.MAP { // {{{
	if RPC_MODE == c.Mode {
		return c.IR.RpcForm
	}

	if c.IR.Form == nil {
		return nil
	}

	params := x.MAP{}

	for k, v := range c.IR.Form {
		if len(v) > 0 {
			params[k] = strings.TrimSpace(v[0])
		}
	}

	return params
} // }}}

// 获取application/json 的数据，转换为jsonMap
func (c *Controller) GetJsonParams() x.MAP { // {{{
	ret := c.RBody
	if len(ret) > 0 {
		json := x.JsonDecode(ret)
		if m, ok := json.(x.MAP); ok {
			return m
		}
	}
	return nil
} // }}}

// 获取数组类型参数
func (c *Controller) GetArray(key string) []string { // {{{
	if c.IR.Form == nil {
		return nil
	}

	ret := []string{}
	retry := true
	for {
		if vs := c.IR.Form[key]; len(vs) > 0 {
			for _, v := range vs {
				ret = append(ret, strings.TrimSpace(v))
			}
			break
		}

		if !retry {
			break
		}

		if strings.HasSuffix(key, "[]") {
			key = key[:len(key)-2]
		} else {
			key = key + "[]"
		}

		retry = false
	}

	return ret
} // }}}

// 获取MAP类型参数
func (c *Controller) GetMap(key string) x.MAPS { // {{{
	if c.IR.Form == nil {
		return nil
	}

	ret := x.MAPS{}
	for k, v := range c.IR.Form {
		if strings.HasPrefix(k, key+"[") && k != key+"[]" && k[len(k)-1] == ']' && len(v) > 0 {
			idx := k[len(key)+1 : len(k)-1]
			ret[idx] = strings.TrimSpace(v[0])
		}
	}

	return ret
} // }}}

// 获取Int类型参数
func (c *Controller) GetInt(key string, defaultValues ...int) int { // {{{
	ret, err := strconv.Atoi(c.getFormValue(key))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

// 获取Int32类型参数
func (c *Controller) GetInt32(key string, defaultValues ...int32) int32 { // {{{
	ret, err := strconv.Atoi(c.getFormValue(key))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return int32(ret)
} // }}}

// 获取Int64类型参数
func (c *Controller) GetInt64(key string, defaultValues ...int64) int64 { // {{{
	ret, err := strconv.ParseInt(c.getFormValue(key), 10, 64)
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

// 获取bool类型参数
func (c *Controller) GetBool(key string, defaultValues ...bool) bool { // {{{
	ret, err := strconv.ParseBool(c.getFormValue(key))
	if err != nil {
		if len(defaultValues) > 0 {
			return defaultValues[0]
		}
	}

	return ret
} // }}}

// 获取json字符串并转换为MAP
func (c *Controller) GetJsonMap(key string) x.MAP { // {{{
	ret := c.getFormValue(key)
	if ret != "" {
		json := x.JsonDecode(ret)
		if m, ok := json.(x.MAP); ok {
			return m
		}
	}
	return nil
} // }}}

// 获取上传文件
func (c *Controller) GetFile(key string) (multipart.File, *multipart.FileHeader, error) { // {{{
	return c.R.FormFile(key)
} // }}}

func (c *Controller) GetIp() string { // {{{
	if HTTP_MODE == c.Mode {
		return x.GetIp(c.R)
	}

	if RPC_MODE == c.Mode {
		pr, ok := peer.FromContext(c.Ctx)
		if !ok {
			return ""
		}

		if pr.Addr == net.Addr(nil) {
			return ""
		}

		addr := strings.Split(pr.Addr.String(), ":")
		return addr[0]
	}

	return ""
} // }}}

func (c *Controller) GetRequestUri() string { // {{{
	if HTTP_MODE == c.Mode && nil != c.R {
		return c.R.URL.String()
	}

	if RPC_MODE == c.Mode && nil != c.IR {
		return x.Concat(c.Controller, "/", c.Action)
	}

	return ""
} // }}}

func (c *Controller) GetUA() string { // {{{
	if HTTP_MODE == c.Mode && nil != c.R {
		return c.R.UserAgent()
	}

	return ""
} // }}}

// lifetime<0时删除cookie
// options: domain,secure,httponly,path
func (c *Controller) SetCookie(key, val string, lifetime int, options ...any) { // {{{
	x.SetCookie(c.W, key, val, lifetime, options...)
} // }}}

func (c *Controller) SetHeader(key, val string) { // {{{
	if HTTP_MODE == c.Mode {
		c.W.Header().Set(key, val)
	} else if RPC_MODE == c.Mode {
		if c.rpcOutHeaders == nil {
			c.rpcOutHeaders = x.MAPS{}
		}
		c.rpcOutHeaders[key] = val
	}
} // }}}

func (c *Controller) SetHeaders(headers http.Header) { // {{{
	c_header := c.W.Header()
	for k, v := range headers {
		c_header.Set(k, v[0])
	}
} // }}}

// 接口正常输出json, 若要改变返回json格式，可在业务代码中重写此方法
func (c *Controller) Render(data ...any) { // {{{
	var retdata any
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(x.MAP)
	}

	res := c.RenderResponser(x.ERR_SUC.GetCode(), x.ERR_SUC.GetMessage(), retdata)

	if RPC_MODE == c.Mode {
		c.renderRpc(res)
		return
	}

	c.RenderJson(res)
} // }}}

// 接口异常输出json，在HttpApiServer中回调, 若要改变返回json格式，可在业务代码中重写此方法
func (c *Controller) RenderError(err any) { // {{{
	errno, errmsg, retdata := c.GetErrorResponse(err)

	res := c.RenderResponser(errno, errmsg, retdata)

	c.logAccessWarn(res)

	if RPC_MODE == c.Mode {
		c.renderRpc(res)
		return
	}

	c.RenderJson(res)
} // }}}

// 根据捕获的错误获取需要返回的错误码、错误信息及数据
func (c *Controller) GetErrorResponse(err any) (int, string, x.MAP) { // {{{
	var errno int
	var errmsg string
	var isbizerr bool

	var retdata = make(x.MAP)

	switch errinfo := err.(type) {
	case string:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = errinfo
	case *x.Error:
		lang := c.GetString("lang")
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage(lang)
		isbizerr = true
	case *x.Errorf:
		lang := c.GetString("lang")
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage(lang)
		retdata = errinfo.GetData()
		isbizerr = true
	case error:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = errinfo.Error()
	default:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = fmt.Sprint(errinfo)
	}

	if !isbizerr {
		debug_trace := debug.Stack()

		c.logFatal(errmsg, string(debug_trace))

		fmt.Println(errmsg)
		os.Stderr.Write(debug_trace)

		if x.Conf_env_mode != "DEV" {
			lang := c.GetString("lang")
			errmsg = x.ERR_SYSTEM.GetMessage(lang)
		}
	}

	if len(retdata) == 0 {
		retdata = x.MAP{}
	}

	return errno, errmsg, retdata
} // }}}

// 格式化输出
func (c *Controller) RenderResponser(errno int, errmsg string, retdata any) x.MAP { // {{{
	return x.MAP{
		"code":    errno,
		"msg":     errmsg,
		"time":    time.Now().Unix(),
		"consume": c.Cost(),
		"data":    retdata, //错误时，也可附带一些数据
	}
} // }}}

// 输出JSON
func (c *Controller) RenderJson(res any) { // {{{
	if nil != c.W {
		c.W.Header().Set("Content-Type", "application/json;charset=UTF-8")
	}

	c.render(x.JsonEncodeBytes(res))
} // }}}

// 输出文本
func (c *Controller) RenderText(res any) { // {{{
	c.render(x.AsBytes(res))
} // }}}

// 输出HTTP状态码
func (c *Controller) RenderStatus(code int) { // {{{
	c.W.WriteHeader(code)
} // }}}

// 渲染html模板
func (c *Controller) RenderHtml(files ...string) { // {{{
	file := ""
	if len(files) > 0 {
		file = files[0]
	}

	uri := c.Controller + "_" + c.Action

	if "" == file {
		file = uri + x.TemplateSuffix
	}

	if c.Tpl == nil {
		c.RenderText("Template is not enabled!")
		return
	}

	err := c.Tpl.Render(c.W, uri, file)

	if nil != err {
		fmt.Println(err)
	}
} // }}}

// 重定向URL
func (c *Controller) Redirect(url string, codes ...int) { // {{{
	code := http.StatusFound //302
	if len(codes) > 0 {
		code = codes[0]
	}
	http.Redirect(c.W, c.R, url, code)
} // }}}

func (c *Controller) render(data []byte) { // {{{
	c.logAccessInfo(string(data))

	if c.Mode == HTTP_MODE {
		c.W.Write(data)
	} else {
		fmt.Printf("%s", data)
	}
} // }}}

func (c *Controller) renderRpc(data x.MAP) { // {{{
	c.logAccessInfo(data)

	header := metadata.New(c.rpcOutHeaders)
	grpc.SendHeader(c.Ctx, header)
	c.rpcContent = data
} // }}}

func (c *Controller) logFatal(data ...any) { // {{{
	if x.Logger != nil {
		x.Logger.Fatal(append(data, c.GenLog())...)
	}
} // }}}

func (c *Controller) logAccessInfo(data any) { // {{{
	// 是否关闭访问日志
	enabled := x.Conf_access_log_enabled
	if !enabled || x.Logger == nil {
		return
	}

	// 使用自定义日志
	if x.Conf_access_log_success_level_name != "" {
		x.Logger.Log(x.Conf_access_log_success_level_name, c.GenLog(), data)
	} else {
		x.Logger.Info(c.GenLog(), data)
	}
} // }}}

// 逻辑同 logAccessInfo
func (c *Controller) logAccessWarn(data any) { // {{{
	enabled := x.Conf_access_log_enabled
	if !enabled || x.Logger == nil {
		return
	}

	if x.Conf_access_log_error_level_name != "" {
		x.Logger.Log(x.Conf_access_log_error_level_name, c.GenLog(), data)
	} else {
		x.Logger.Warn(c.GenLog(), data)
	}
} // }}}

// 获取日志内容
func (c *Controller) GenLog() x.MAP { // {{{
	ret := make(x.MAP)
	uri := c.GetRequestUri()

	if HTTP_MODE == c.Mode && nil != c.R {
		//访问ip
		ret["ip"] = c.GetIp()
		//请求路径
		ret["uri"] = uri

		if c.R.Method == "POST" {
			ret["post"] = c.R.PostForm
		}

		ret["ua"] = c.R.UserAgent()
	}

	if RPC_MODE == c.Mode && nil != c.IR {
		ret["guid"] = c.GetHeader("guid")
		ret["ip"] = c.GetIp()
		ret["uri"] = uri
		ret["post"] = c.IR.RpcForm
	}

	for k, v := range c.logParams {
		ret[k] = v
	}

	//post 时，过滤敏感字段
	if nil != ret["post"] {
		if ret_post, ok := ret["post"].(url.Values); ok {
			c.OmitLog(x.Conf_access_log_omit_params...)

			for _, v := range c.logOmitParams {
				if _, ok := ret_post[v]; ok {
					delete(ret_post, v)
				}
			}
		}
	}

	return ret
} // }}}

// 在业务日志中添加自定义字段
func (c *Controller) AddLog(k string, v any) { // {{{
	if nil == c.logParams {
		c.logParams = x.MAP{}
	}

	c.logParams[k] = v
} // }}}

// 在业务日志中删除字段(比如密码等敏感字段)
func (c *Controller) OmitLog(v ...string) { // {{{
	if len(v) == 0 {
		return
	}

	if nil == c.logOmitParams {
		c.logOmitParams = []string{}
	}

	c.logOmitParams = append(c.logOmitParams, v...)
} // }}}

func (c *Controller) Cost() int64 {
	return time.Now().Sub(c.startTime).Nanoseconds() / 1000 / 1000
}

func (c *Controller) GetRpcContent() x.MAP { // {{{
	return c.rpcContent
} // }}}

// 便于调式时直接使用
func (c *Controller) Println(s ...any) { // {{{
	fmt.Println(s...)
} // }}}
