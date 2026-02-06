package x

import (
	"fmt"
	"strings"
	"sync"
)

//错误信息加载顺序: 配置文件 -> 预定义变量 -> 代码行

var (
	//多语言时指定默认语言
	DEFAULT_LANG = "CN"

	//成功
	ERR_SUC = NewErr(0, "OK")

	//系统错误码
	ERR_SYSTEM         = NewErr(10, "CN", "系统错误", "EN", "system error")
	ERR_AUTH           = NewErr(11, "CN", "认证失败", "EN", "request unauthorized")
	ERR_METHOD_INVALID = NewErr(12, "CN", "请求不合法: %+v", "EN", "invalid request: %+v")
	ERR_PARAMS         = NewErr(14, "CN", "参数错误: %+v", "EN", "invalid param: %+v")
	ERR_OTHER          = NewErr(15, "%+v")

	ErrMap   = map[int32]MAPS{}
	ErrMapRo = map[int32]MAPS{} //只读MAP
	mu       sync.Mutex
)

func NewErr(code int32, msgs ...string) *Error { // {{{
	msgMap := MAPS{}

	i := 0
	for i+1 < len(msgs) {
		msgMap[strings.ToUpper(msgs[i])] = msgs[i+1]
		//确保 DEFAULT_LANG 下有值
		if _, ok := msgMap[DEFAULT_LANG]; !ok {
			msgMap[DEFAULT_LANG] = msgs[i+1]
		}

		i += 2
	}

	if len(msgMap) == 0 && len(msgs) > 0 {
		msgMap[DEFAULT_LANG] = msgs[0]
	}

	mu.Lock()
	ErrMap[code] = msgMap
	mu.Unlock()

	return &Error{code: code, msg: msgMap}
} // }}}

type Error struct {
	code int32
	msg  MAPS
	fmt  []any
	data MAP
}

func (e *Error) GetCode() int32 {
	return e.code
}

func (e *Error) GetMessage(langs ...string) string { // {{{
	if len(e.fmt) > 0 {
		//fmts的可用值为string, 若fmts最后一个值为map, 则认为它是异常时返回的data
		if data, ok := e.fmt[len(e.fmt)-1].(MAP); ok {
			e.fmt = e.fmt[0 : len(e.fmt)-1]
			e.data = data
		}
	}

	var lang, msg string

	if len(langs) > 0 {
		lang = langs[0]
	} else {
		lang = DEFAULT_LANG
	}

	errMsgs, ok := ErrMapRo[e.code]
	if ok {
		msg, ok = errMsgs[lang]
	}

	if !ok {
		msg, ok = e.msg[lang]

		if !ok && lang != DEFAULT_LANG {
			msg = e.msg[DEFAULT_LANG]
		}
	}

	if len(e.fmt) > 0 {
		return fmt.Sprintf(msg, e.fmt)
	}

	return msg
} // }}}

// 从 fmt 参数中提取 data
func (e *Error) GetData() MAP { // {{{
	if e.data == nil && len(e.fmt) > 0 {
		if data, ok := e.fmt[len(e.fmt)-1].(MAP); ok {
			return data
		}
	}

	return e.data
} // }}}

func (e *Error) Error() string {
	return e.GetMessage()
}

// 捕获异常时，可同时返回data(通过fmts参数最后一个类型为map的值)
func Interceptor(guard bool, errmsg any, fmts ...any) { // {{{
	if !guard {
		var err *Error
		if v, ok := errmsg.(*Error); ok {
			v.fmt = fmts
			err = v
		} else {
			err = ERR_OTHER
			err.fmt = []any{errmsg}
		}

		panic(err)
	}
} // }}}
