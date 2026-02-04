package controller

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type Controller struct {
	Ctx            context.Context
	Group          string
	ControllerName string
	ActionName     string
	ResData        *x.ResponseData
	ResError       error

	guid      string
	startTime time.Time
	lang      string //语言

	logParams     x.MAP               //需要额外记录在日志中的参数
	logOmitParams map[string]struct{} //不希望记录在日志中的参数
}

// 默认的初始化方法，可通过在项目中重写此方法实现公共入口方法
func (c *Controller) Init() {}

func (c *Controller) Prepare(ctx context.Context, controller, action, group string) { // {{{
	c.startTime = time.Now()
	c.Group = group
	c.ControllerName = controller
	c.ActionName = action
	c.Ctx = ctx
} // }}}

func (c *Controller) GetCtx(key string, defaultValues ...string) any { // {{{
	return c.Ctx.Value(key)
} // }}}

func (c *Controller) SetCtx(key, value any) { // {{{
	c.Ctx = context.WithValue(c.Ctx, key, value)
} // }}}

func (c *Controller) SetGuid(guid string) { // {{{
	c.guid = guid
	c.SetCtx(x.ConfGuidKey, guid)
} // }}}

func (c *Controller) SetLang(lang string) { // {{{
	c.lang = lang
	c.SetCtx(x.ConfLangKey, lang)
} // }}}

// 根据捕获的错误获取需要返回的错误码、错误信息及数据
func (c *Controller) GetErrorResponse(err any) (int32, string, x.MAP) { // {{{
	var errno int32
	var errmsg string
	var isbizerr bool

	var retdata = make(x.MAP)

	switch errinfo := err.(type) {
	case string:
		errno = x.ERR_SYSTEM.GetCode()
		errmsg = errinfo
	case *x.Error:
		errno = errinfo.GetCode()
		errmsg = errinfo.GetMessage(c.lang)
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

		c.SetCtx("debug_trace", errmsg+"\n"+string(debug_trace))

		fmt.Println(errmsg)
		os.Stderr.Write(debug_trace)

		if x.ConfEnvMode != "dev" {
			errmsg = x.ERR_SYSTEM.GetMessage(c.lang)
		}
	}

	if len(retdata) == 0 {
		retdata = x.MAP{}
	}

	c.SetCtx("error_code", errno)

	return errno, errmsg, retdata
} // }}}

// 格式化输出
func (c *Controller) RenderResponser(errno int32, errmsg string, retdata any) *x.ResponseData { // {{{
	c.ResData = &x.ResponseData{
		Code:    errno,
		Consume: int32(x.Cost(c.startTime)),
		Msg:     errmsg,
		Time:    time.Now().Unix(),
		Data:    retdata, //错误时，也可附带一些数据
	}

	c.SetCtx("errno", errno)

	return c.ResData
} // }}}

func (c *Controller) GetResponseData() (context.Context, *x.ResponseData, error) { // {{{
	return c.Ctx, c.ResData, c.ResError
} // }}}

// 在业务日志中添加自定义字段
func (c *Controller) AddLog(k string, v any) { // {{{
	if nil == c.logParams {
		c.logParams = x.MAP{}
		c.SetCtx("log_params", c.logParams)
	}

	c.logParams[k] = v
} // }}}

// 在业务日志中删除字段(比如密码等敏感字段)
func (c *Controller) OmitLog(v ...string) { // {{{
	if len(v) == 0 {
		return
	}

	if nil == c.logOmitParams {
		c.logOmitParams = map[string]struct{}{}
		c.SetCtx("log_omit_params", c.logOmitParams)
	}

	for _, k := range v {
		c.logOmitParams[k] = struct{}{}
	}
} // }}}

// 便于调式时直接使用
func (c *Controller) Println(s ...any) { // {{{
	log.Println(s...)
} // }}}

func (c *Controller) Printf(f string, s ...any) { // {{{
	log.Printf(f, s...)
} // }}}
