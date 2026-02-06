package controller

import (
	"context"
	"github.com/nyxless/nyx/x"
)

type RPC struct {
	rpcContainer
}

func (r *RPC) Prepare(ctx context.Context, params x.MAP, controller, action, group string, stream x.Stream) { // {{{
	r.RpcForm = params
	r.rpcStream = stream
	r.Controller.Prepare(ctx, controller, action, group)

	if len(x.ConfRpcLogOmitParams) > 0 {
		r.OmitLog(x.ConfRpcLogOmitParams...)
	}

	// guid 用于日志追踪，可由客户端生成, 依次检查: 请求参数 -> header -> 生成
	guid := r.GetString(x.ConfGuidKey, r.GetHeader(x.ConfGuidKey, x.GetUUID()))
	r.SetGuid(guid)

	// lang 用于错误信息按语言展示, 依次检查: 请求参数 -> header -> 配置文件 -> 默认
	lang := r.GetString(x.ConfLangKey, r.GetHeader(x.ConfLangKey, x.DEFAULT_LANG))
	r.SetLang(lang)

	r.SetHeader(x.ConfGuidKey, guid)
	r.SetHeader(x.ConfLangKey, lang)

} // }}}
