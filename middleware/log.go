package middleware

import (
	"bytes"
	"context"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/log"
	"io"
	"net/http"
	"strings"
)

type LogConfig struct {
	InfoLogName    string
	WarnLogName    string
	ErrorLogName   string
	CheckMethod    []string
	CheckExcept    []string
	CheckReqMethod []string
	CheckReqExcept []string
	CheckResMethod []string
	CheckResExcept []string
	Logger         *log.Logger
}

func HttpLog(config *LogConfig) x.HttpMiddleware { // {{{
	checkMethod, checkExcept, checkReqMethod, checkReqExcept, checkResMethod, checkResExcept := parseLogConfig(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			if logCheckMethod(ctx, checkMethod, checkExcept) {
				hrr := newhttpResponseRecorder(w)
				next.ServeHTTP(hrr, r)

				ctx = r.Context()
				checkReq := logCheckReqMethod(ctx, checkReqMethod, checkReqExcept)
				data := getHttpLogData(ctx, hrr, r, checkReq)

				body := ""

				if logCheckResMethod(ctx, checkResMethod, checkResExcept) {
					body = hrr.Body()
				}

				errno, _ := ctx.Value("errno").(int32)
				writeLog(ctx, config, errno, data, body)

			} else {
				next.ServeHTTP(w, r)
			}

		})
	}
} // }}}

func RpcLog(config *LogConfig) x.RpcMiddleware { // {{{
	checkMethod, checkExcept, checkReqMethod, checkReqExcept, checkResMethod, checkResExcept := parseLogConfig(config)

	return func(next x.RpcHandler) x.RpcHandler {
		return func(ctx context.Context, params x.MAP, stream x.Stream) (context.Context, *x.ResponseData, error) {
			ctx, resData, err := next(ctx, params, stream)

			if err != nil {
				x.Warn("rpc err:", err)

				return ctx, resData, err
			}

			if logCheckMethod(ctx, checkMethod, checkExcept) {
				checkReq := logCheckReqMethod(ctx, checkReqMethod, checkReqExcept)
				data := getRpcLogData(ctx, params, checkReq)

				msgs := []any{data}
				errno := resData.GetCode()
				if logCheckResMethod(ctx, checkResMethod, checkResExcept) {
					msgs = append(msgs, log.LogField("data", resData.GetData()))
				}

				if errno > 0 {
					msgs = append(msgs, log.LogField("errno", errno))
					msgs = append(msgs, log.LogField("errmsg", resData.GetMsg()))
				}

				writeLog(ctx, config, errno, msgs...)
			}

			return ctx, resData, err
		}
	}
} // }}}

func writeLog(ctx context.Context, config *LogConfig, errno int32, data ...any) { // {{{
	if errno == 0 {
		if config.InfoLogName != "" {
			config.Logger.Log(config.InfoLogName, data...)
		} else {
			config.Logger.Info(data...)
		}
	} else {
		if config.WarnLogName != "" {
			config.Logger.Log(config.WarnLogName, data...)
		} else {
			config.Logger.Warn(data...)
		}

		debug_trace, _ := ctx.Value("debug_trace").(string)
		if debug_trace != "" {
			data = append(data, debug_trace)
			if config.ErrorLogName != "" {
				config.Logger.Log(config.ErrorLogName, data...)
			} else {
				config.Logger.Error(data...)
			}
		}
	}
} // }}}

func getHttpLogData(ctx context.Context, w *httpResponseRecorder, r *http.Request, checkReq bool) x.MAP { // {{{
	logParams, _ := ctx.Value("log_params").(x.MAP)
	logOmitParams, _ := ctx.Value("log_omit_params").(map[string]struct{})

	_, ip := x.GetHttpCtxIp(ctx, r)
	ret := x.MAP{
		x.ConfGuidKey: ctx.Value("guid"),
		"uri":         r.URL.String(),
		"ip":          ip,
		"ua":          r.UserAgent(),
	}

	for k, v := range logParams {
		ret[k] = v
	}

	status_code, _ := ctx.Value("http_status_code").(int)
	if checkReq && r.Method == "POST" && status_code != http.StatusNotFound {
		if json_form, ok := ctx.Value("json_form").(x.MAP); ok {
			d := make(x.MAP)
			for k, v := range json_form {
				if _, ok := logOmitParams[k]; !ok {
					d[k] = v
				}
			}
			ret["post"] = d
		} else if r.PostForm != nil {
			d := make(x.MAP)
			for k, v := range r.PostForm {
				if _, ok := logOmitParams[k]; !ok {
					d[k] = v
				}
			}
			ret["post"] = d
		}
	}

	return ret
} // }}}

func getRpcLogData(ctx context.Context, params x.MAP, checkReq bool) x.MAP { // {{{
	controller, _ := ctx.Value("controller").(string)
	action, _ := ctx.Value("action").(string)
	logParams, _ := ctx.Value("log_params").(x.MAP)
	logOmitParams, _ := ctx.Value("log_omit_params").(map[string]struct{})

	_, ip := x.GetRpcCtxIp(ctx)
	ret := x.MAP{
		x.ConfGuidKey: ctx.Value("guid"),
		"uri":         controller + "/" + action,
		"ip":          ip,
	}

	for k, v := range logParams {
		ret[k] = v
	}

	if checkReq && len(params) > 0 {
		d := make(x.MAP)
		for k, v := range params {
			if _, ok := logOmitParams[k]; !ok {
				d[k] = v
			}
		}

		ret["post"] = d
	}

	return ret
} // }}}

func parseLogConfig(config *LogConfig) (checkMethod, checkExcept, checkReqMethod, checkReqExcept, checkResMethod, checkResExcept map[string]struct{}) { // {{{
	checkMethod = map[string]struct{}{}
	for _, v := range config.CheckMethod {
		checkMethod[strings.ToLower(v)] = struct{}{}
	}

	checkExcept = map[string]struct{}{}
	for _, v := range config.CheckExcept {
		checkExcept[strings.ToLower(v)] = struct{}{}
	}

	checkReqMethod = map[string]struct{}{}
	for _, v := range config.CheckReqMethod {
		checkReqMethod[strings.ToLower(v)] = struct{}{}
	}

	checkReqExcept = map[string]struct{}{}
	for _, v := range config.CheckReqExcept {
		checkReqExcept[strings.ToLower(v)] = struct{}{}
	}

	checkResMethod = map[string]struct{}{}
	for _, v := range config.CheckResMethod {
		checkResMethod[strings.ToLower(v)] = struct{}{}
	}

	checkResExcept = map[string]struct{}{}
	for _, v := range config.CheckResExcept {
		checkResExcept[strings.ToLower(v)] = struct{}{}
	}

	return checkMethod, checkExcept, checkReqMethod, checkReqExcept, checkResMethod, checkResExcept
} // }}}

// 返回是否需要记日志
func logCheckMethod(ctx context.Context, checkMethod, checkExcept map[string]struct{}) bool { // {{{
	group, _ := ctx.Value("group").(string)
	controller, _ := ctx.Value("controller").(string)
	action, _ := ctx.Value("action").(string)

	if len(checkMethod) > 0 {
		_, exists := checkMethod[group]
		if !exists {
			_, exists = checkMethod[controller]
			if !exists {
				_, exists = checkMethod[controller+"/"+action]
				if !exists {
					return false
				}
			}
		}
	}

	if len(checkExcept) > 0 {
		if _, exists := checkExcept[group]; exists {
			return false
		}

		if _, exists := checkExcept[controller]; exists {
			return false
		}

		if _, exists := checkExcept[controller+"/"+action]; exists {
			return false
		}
	}

	return true
} // }}}

// 返回是否记录请求数据
func logCheckReqMethod(ctx context.Context, checkReqMethod, checkReqExcept map[string]struct{}) bool { // {{{
	group, _ := ctx.Value("group").(string)
	controller, _ := ctx.Value("controller").(string)
	action, _ := ctx.Value("action").(string)

	if len(checkReqMethod) > 0 {
		_, exists := checkReqMethod[group]
		if !exists {
			_, exists = checkReqMethod[controller]
			if !exists {
				_, exists = checkReqMethod[controller+"/"+action]
				if !exists {
					return false
				}
			}
		}
	}

	if len(checkReqExcept) > 0 {
		if _, exists := checkReqExcept[group]; exists {
			return false
		}

		if _, exists := checkReqExcept[controller]; exists {
			return false
		}

		if _, exists := checkReqExcept[controller+"/"+action]; exists {
			return false
		}
	}

	return true
} // }}}

// 返回是否记录返回数据
func logCheckResMethod(ctx context.Context, checkResMethod, checkResExcept map[string]struct{}) bool { // {{{
	group, _ := ctx.Value("group").(string)
	controller, _ := ctx.Value("controller").(string)
	action, _ := ctx.Value("action").(string)

	if len(checkResMethod) > 0 {
		_, exists := checkResMethod[group]
		if !exists {
			_, exists = checkResMethod[controller]
			if !exists {
				_, exists = checkResMethod[controller+"/"+action]
				if !exists {
					return false
				}
			}
		}
	}

	if len(checkResExcept) > 0 {
		if _, exists := checkResExcept[group]; exists {
			return false
		}

		if _, exists := checkResExcept[controller]; exists {
			return false
		}

		if _, exists := checkResExcept[controller+"/"+action]; exists {
			return false
		}
	}

	return true
} // }}}

func getRequestBody(r *http.Request) (rbody []byte, err error) { // {{{
	rbody, err = io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(rbody))

	return rbody, nil
} // }}}

type httpResponseRecorder struct { // {{{
	http.ResponseWriter
	body *bytes.Buffer
}

func newhttpResponseRecorder(w http.ResponseWriter) *httpResponseRecorder {
	return &httpResponseRecorder{
		ResponseWriter: w,
		body:           bytes.NewBuffer(nil),
	}
}

func (h *httpResponseRecorder) WriteHeader(statusCode int) {
	h.ResponseWriter.WriteHeader(statusCode)
}

func (h *httpResponseRecorder) Write(b []byte) (int, error) {
	h.body.Write(b)
	return h.ResponseWriter.Write(b)
}

func (h *httpResponseRecorder) Body() string {
	return h.body.String()
} // }}}
