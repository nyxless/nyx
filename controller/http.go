package controller

import (
	"bytes"
	"github.com/nyxless/nyx/x"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type HTTP struct {
	httpContainer
	Tpl *x.Template
}

func (h *HTTP) Prepare(w http.ResponseWriter, r *http.Request, controller, action, group string) { // {{{
	h.W = w
	h.R = r

	if x.ConfTemplateEnabled {
		h.Tpl = x.NewTemplate()
	}

	h.httpContainer.Prepare()
	h.Controller.Prepare(r.Context(), controller, action, group)
	h.SetCtx("ua", h.R.UserAgent())

	if len(x.ConfHttpLogOmitParams) > 0 {
		h.OmitLog(x.ConfHttpLogOmitParams...)
	}

	if h.JsonForm != nil {
		h.SetCtx("json_form", h.JsonForm)
	}

	// guid 用于日志追踪，可由客户端生成, 依次检查: 请求参数 -> header -> 生成
	guid := h.GetString(x.ConfGuidKey, h.GetHeader(x.ConfGuidKey, x.GetUUID()))
	h.SetGuid(guid)

	// lang 用于错误信息按语言展示, 依次检查: 请求参数 -> header -> 配置文件 -> 默认
	lang := h.GetString(x.ConfLangKey, h.GetHeader(x.ConfLangKey, x.DEFAULT_LANG))
	h.SetLang(lang)

	h.SetHeader(x.ConfGuidKey, guid)
	h.SetHeader(x.ConfLangKey, lang)
} // }}}

// 设置 POST 表单大小,   应该在 Init 方法中调用
func (h *HTTP) SetMaxPostSize(m int64) { // {{{
	h.MaxPostSize = m
} // }}}

func (h *HTTP) GetRequestBody() (rbody []byte, err error) { // {{{
	if h.RBody == nil {
		h.RBody, err = io.ReadAll(h.R.Body)
		if err != nil {
			return nil, err
		}

		h.R.Body = io.NopCloser(bytes.NewReader(h.RBody))
	}

	return h.RBody, nil
} // }}}

// 获取上传文件
func (h *HTTP) GetFile(key string) (multipart.File, *multipart.FileHeader, error) { // {{{
	return h.R.FormFile(key)
} // }}}

// 输出文本
func (h *HTTP) RenderText(res any) { // {{{
	data := x.AsBytes(res)
	h.W.Write(data)
} // }}}

// 输出HTTP状态码
func (h *HTTP) RenderStatus(code int) { // {{{
	h.W.WriteHeader(code)
} // }}}

// http 输出到文件下载
func (h *HTTP) RenderFile(rs io.ReadSeeker, filename string) { // {{{
	h.SetHeader("Content-Disposition", "attachment; filename=\""+filename+"\"")
	http.ServeContent(h.W, h.R, filename, x.NowTime(), rs)
} // }}}

// 渲染html模板
func (h *HTTP) RenderHtml(files ...string) { // {{{
	if h.Tpl == nil {
		h.RenderText("Template is not enabled!")
		return
	}

	uri := h.ControllerName + "_" + h.ActionName

	file := ""
	if len(files) > 0 {
		file = files[0]
		if h.Group != "" && strings.Index(file, "/") == -1 {
			file = filepath.Join(h.Group, file)
		}
		if strings.Index(file, ".") == -1 {
			file = file + x.TemplateSuffix
		}
	} else {
		file = uri + x.TemplateSuffix
	}

	err := h.Tpl.Render(h.W, uri, file)

	if nil != err {
		log.Println(err)
	}
} // }}}

// 模板解析变量
func (h *HTTP) Assign(vals ...any) { // {{{
	if h.Tpl != nil {
		h.Tpl.Assign(vals...)
	}
} // }}}

// 重定向URL
func (h *HTTP) Redirect(url string, codes ...int) { // {{{
	code := http.StatusFound //302
	if len(codes) > 0 {
		code = codes[0]
	}
	http.Redirect(h.W, h.R, url, code)
} // }}}

// 回调方法, 只在HttpServer中使用
func (h *HTTP) HttpFinal() { // {{{
	//将 ctx 写回 http.Request, 供中间件使用
	*h.R = *h.R.WithContext(h.Ctx)
} // }}}

// 用户回调方法, 可在业务代码中重写
func (h *HTTP) Final() { // {{{
} // }}}
