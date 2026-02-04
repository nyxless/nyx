package x

import (
	"net/http"
)

type HttpMiddleware func(http.Handler) http.Handler
type RpcMiddleware func(RpcHandler) RpcHandler

type httpMiddlewareGroup struct {
	middleware HttpMiddleware
	groups     map[string]struct{}
}

type rpcMiddlewareGroup struct {
	middleware RpcMiddleware
	groups     map[string]struct{}
}

var httpMiddlewares []httpMiddlewareGroup
var rpcMiddlewares []rpcMiddlewareGroup

func UseHttpMiddleware(middleware HttpMiddleware, groups ...string) { // {{{
	midgroups := map[string]struct{}{}
	for _, group := range groups {
		midgroups[group] = struct{}{}
	}

	httpMiddlewares = append(httpMiddlewares, httpMiddlewareGroup{middleware, midgroups})
} // }}}

func UseRpcMiddleware(middleware RpcMiddleware, groups ...string) { // {{{
	midgroups := map[string]struct{}{}
	for _, group := range groups {
		midgroups[group] = struct{}{}
	}

	rpcMiddlewares = append(rpcMiddlewares, rpcMiddlewareGroup{middleware, midgroups})
} // }}}

func buildHttpMiddlewares(group string, middlewares ...httpMiddlewareGroup) HttpMiddleware { // {{{
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			if middlewares[i].groups == nil || len(middlewares[i].groups) == 0 {
				final = middlewares[i].middleware(final)
			} else if _, ok := middlewares[i].groups[group]; ok {
				final = middlewares[i].middleware(final)
			}
		}

		return final
	}
} // }}}

func buildRpcMiddlewares(group string, middlewares ...rpcMiddlewareGroup) RpcMiddleware { // {{{
	return func(final RpcHandler) RpcHandler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			if middlewares[i].groups == nil || len(middlewares[i].groups) == 0 {
				final = middlewares[i].middleware(final)
			} else if _, ok := middlewares[i].groups[group]; ok {
				final = middlewares[i].middleware(final)
			}
		}

		return final
	}
} // }}}
