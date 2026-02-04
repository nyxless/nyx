package controller

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
	"time"
)

type rpcContainer struct {
	Controller
	RpcForm       x.MAP
	rpcInHeaders  metadata.MD
	rpcOutHeaders x.MAPS
	rpcStream     x.Stream //grpc stream
}

func (r *rpcContainer) GetParams() x.MAP { // {{{
	return r.RpcForm
} // }}}

func (r *rpcContainer) GetParam(key string) any { // {{{
	return r.RpcForm[key]
} // }}}

func (r *rpcContainer) GetString(key string, defaultValues ...string) string { // {{{
	return x.AsString(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetInt(key string, defaultValues ...int) int { // {{{
	return x.AsInt(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetInt8(key string, defaultValues ...int8) int8 { // {{{
	return x.AsInt8(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetInt16(key string, defaultValues ...int16) int16 { // {{{
	return x.AsInt16(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetInt32(key string, defaultValues ...int32) int32 { // {{{
	return x.AsInt32(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetInt64(key string, defaultValues ...int64) int64 { // {{{
	return x.AsInt64(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetBool(key string, defaultValues ...bool) bool { // {{{
	return x.AsBool(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetFloat(key string, defaultValues ...float64) float64 { // {{{
	return r.GetFloat64(key, defaultValues...)
} // }}}

func (r *rpcContainer) GetFloat32(key string, defaultValues ...float32) float32 { // {{{
	return x.AsFloat32(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetFloat64(key string, defaultValues ...float64) float64 { // {{{
	return x.AsFloat64(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetJsonMap(key string) x.MAP { // {{{
	return x.AsMap(x.JsonDecode(r.RpcForm[key]))
} // }}}

func (r *rpcContainer) GetSlice(key string, separators ...string) []any { // {{{
	return x.AsSlice(r.RpcForm[key], separators...)
} // }}}

func (r *rpcContainer) GetStringSlice(key string, separators ...string) []string { // {{{
	return x.AsStringSlice(r.RpcForm[key], separators...)
} // }}}

func (r *rpcContainer) GetIntSlice(key string, separators ...string) []int { // {{{
	return x.AsIntSlice(r.RpcForm[key], separators...)
} // }}}

func (r *rpcContainer) GetInt32Slice(key string, separators ...string) []int32 { // {{{
	return x.AsInt32Slice(r.RpcForm[key], separators...)
} // }}}

func (r *rpcContainer) GetInt64Slice(key string, separators ...string) []int64 { // {{{
	return x.AsInt64Slice(r.RpcForm[key], separators...)
} // }}}

func (r *rpcContainer) GetMapSlice(key string) []x.MAP { // {{{
	return x.AsMapSlice(r.RpcForm[key])
} // }}}

func (r *rpcContainer) GetBytes(key string, defaultValues ...[]byte) []byte { // {{{
	return x.AsBytes(r.RpcForm[key], defaultValues...)
} // }}}

func (r *rpcContainer) GetMap(key string) x.MAP { // {{{
	return x.AsMap(r.RpcForm[key])
} // }}}

func (r *rpcContainer) GetStringMap(key string) x.MAPS { // {{{
	return x.AsStringMap(r.RpcForm[key])
} // }}}

func (r *rpcContainer) GetIntMap(key string) x.MAPI { // {{{
	return x.AsIntMap(r.RpcForm[key])
} // }}}

func (r *rpcContainer) GetTime(key string) time.Time { // {{{
	return x.AsTime(r.RpcForm[key])
} // }}}

func (r *rpcContainer) GetIp() (ip string) { // {{{
	r.Ctx, ip = x.GetRpcCtxIp(r.Ctx)
	return ip
} // }}}

func (r *rpcContainer) GetHeader(key string, defaultValues ...string) (ret string) { // {{{
	key = strings.ToLower(key)

	if r.rpcInHeaders == nil {
		r.rpcInHeaders, _ = metadata.FromIncomingContext(r.Ctx)
	}

	if r.rpcInHeaders != nil {
		if v, ok := r.rpcInHeaders[key]; ok {
			ret = v[0]
		}
	}

	if ret == "" && len(defaultValues) > 0 {
		return defaultValues[0]
	}

	return ret
} // }}}

func (r *rpcContainer) GetHeaders() (ret x.MAPS) { // {{{
	if r.rpcInHeaders == nil {
		r.rpcInHeaders, _ = metadata.FromIncomingContext(r.Ctx)
	}

	ret = make(x.MAPS, len(r.rpcInHeaders))
	for k, v := range r.rpcInHeaders {
		ret[k] = v[0]
	}

	return ret
} // }}}

func (r *rpcContainer) SetHeader(key, val string) { // {{{
	if r.rpcOutHeaders == nil {
		r.rpcOutHeaders = x.MAPS{}
	}
	r.rpcOutHeaders[key] = val
} // }}}

func (r *rpcContainer) SetHeaders(headers x.MAPS) { // {{{
	if r.rpcOutHeaders == nil {
		r.rpcOutHeaders = x.MAPS{}
	}

	for k, v := range headers {
		r.rpcOutHeaders[k] = v
	}
} // }}}

func (r *rpcContainer) Render(data ...any) { // {{{
	var retdata any
	if len(data) > 0 {
		retdata = data[0]
	} else {
		retdata = make(x.MAP)
	}

	res := r.RenderResponser(x.ERR_SUC.GetCode(), x.ERR_SUC.GetMessage(), retdata)
	r.render(res)
} // }}}

func (r *rpcContainer) RenderError(err any) { // {{{
	errno, errmsg, retdata := r.GetErrorResponse(err)
	res := r.RenderResponser(errno, errmsg, retdata)

	r.render(res)
} // }}}

func (r *rpcContainer) RenderStream(data any) error { // {{{
	r.Render(data)
	return r.ResError
} // }}}

func (r *rpcContainer) render(data *x.ResponseData) { // {{{
	header := metadata.New(r.rpcOutHeaders)
	grpc.SendHeader(r.Ctx, header)

	if r.Ctx.Err() == context.Canceled {
		r.ResError = fmt.Errorf("client cancelled the request")
		return
	}

	if r.rpcStream != nil {
		r.ResError = r.rpcStream.Send(x.BuildReply(data))
	}
} // }}}
