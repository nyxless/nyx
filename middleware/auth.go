package middleware

import (
	"context"
	"github.com/nyxless/nyx/controller"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/cache"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	DefaultApiAuthFn = func(config *AuthConfig, r *http.Request) (string, bool) { // {{{
		// 从Header获取认证参数
		appID := r.Header.Get("appid")
		nonce := r.Header.Get("nonce")
		timestamp := r.Header.Get("timestamp")
		authorization := r.Header.Get("authorization")

		// 提取Bearer token
		token := strings.TrimPrefix(authorization, "Bearer ")

		if appID == "" || nonce == "" || timestamp == "" || token == "" {
			return "", false
		}

		// 获取对应的secret
		secret, ok := config.AppSecrets[appID]
		if !ok {
			return "", false
		}

		// 检查TTL
		if config.CheckTTL > 0 {
			ts, err := strconv.Atoi(timestamp)
			if err != nil {
				return "", false
			}

			if int(time.Now().Unix())-ts > config.CheckTTL {
				return "", false
			}
		}

		// 防重放攻击检查
		if config.CheckNonce && config.LocalCache != nil {
			if _, err := config.LocalCache.Get([]byte(nonce)); err == nil {
				return "", false
			}

			config.LocalCache.Set([]byte(nonce), []byte(""), config.CheckTTL)
		}

		return appID, x.VerifySha256(token, appID+nonce+timestamp, secret)
	} // }}}

	DefaultRpcAuthFn = func(config *AuthConfig, ctx context.Context) (string, bool) { // {{{
		var appID, secret string
		if headers, ok := metadata.FromIncomingContext(ctx); ok {
			if v, ok := headers["appid"]; ok {
				appID = v[0]
			}

			if v, ok := headers["secret"]; ok {
				secret = v[0]
			}
		}

		conf_secret, ok := config.AppSecrets[appID]
		if !ok {
			return "", false
		}

		return appID, conf_secret == secret
	} // }}}

	DefaultApiErrHandler = func(w http.ResponseWriter, r *http.Request) { // {{{
		ctx := r.Context()
		group, _ := ctx.Value("group").(string)
		controllerName, _ := ctx.Value("controller").(string)
		actionName, _ := ctx.Value("action").(string)

		c := &controller.HTTP{}
		c.Prepare(w, r, controllerName, actionName, group)
		c.RenderError(x.ERR_AUTH)
		c.Final()
		return
	} // }}}

	DefaultRpcErrHandler = func(ctx context.Context, params map[string]any, stream x.Stream) (context.Context, *x.ResponseData, error) { // {{{
		group, _ := ctx.Value("group").(string)
		controllerName, _ := ctx.Value("controller").(string)
		actionName, _ := ctx.Value("action").(string)

		c := &controller.RPC{}
		c.Prepare(ctx, params, controllerName, actionName, group, stream)
		c.RenderError(x.ERR_AUTH)
		return c.GetResponseData()
	} // }}}
)

type AuthConfig struct {
	AppSecrets  map[string]string //appid:secret
	CheckTTL    int
	CheckNonce  bool
	CheckMethod []string
	CheckExcept []string
	CheckAllow  map[string][]string //appid:[a/b,c/d]
	CheckForbid map[string][]string
	LocalCache  cache.LocalCache
}

func ApiAuth(config *AuthConfig) x.HttpMiddleware { // {{{
	checkMethod, checkExcept, checkAllow, checkForbid := parseConfig(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if CheckMethod(ctx, checkMethod, checkExcept) {
				appID, passed := DefaultApiAuthFn(config, r)
				if !passed || !CheckAllow(ctx, appID, checkAllow, checkForbid) {
					DefaultApiErrHandler(w, r)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
} // }}}

func RpcAuth(config *AuthConfig) x.RpcMiddleware { // {{{
	checkMethod, checkExcept, checkAllow, checkForbid := parseConfig(config)

	return func(next x.RpcHandler) x.RpcHandler {
		return func(ctx context.Context, params map[string]any, stream x.Stream) (context.Context, *x.ResponseData, error) {
			if CheckMethod(ctx, checkMethod, checkExcept) {
				appID, passed := DefaultRpcAuthFn(config, ctx)
				if !passed || !CheckAllow(ctx, appID, checkAllow, checkForbid) {
					return DefaultRpcErrHandler(ctx, params, stream)
				}
			}

			return next(ctx, params, stream)
		}
	}
} // }}}

func parseConfig(config *AuthConfig) (checkMethod, checkExcept map[string]struct{}, checkAllow, checkForbid map[string]map[string]struct{}) { // {{{
	checkMethod = map[string]struct{}{}
	for _, v := range config.CheckMethod {
		checkMethod[strings.ToLower(v)] = struct{}{}
	}

	checkExcept = map[string]struct{}{}
	for _, v := range config.CheckExcept {
		checkExcept[strings.ToLower(v)] = struct{}{}
	}

	checkAllow = map[string]map[string]struct{}{}
	for appid, v := range config.CheckAllow {
		checkAllow[appid] = map[string]struct{}{}
		for _, m := range v {
			checkAllow[appid][strings.ToLower(m)] = struct{}{}
		}
	}

	checkForbid = map[string]map[string]struct{}{}
	for appid, v := range config.CheckForbid {
		checkForbid[appid] = map[string]struct{}{}
		for _, m := range v {
			checkForbid[appid][strings.ToLower(m)] = struct{}{}
		}
	}

	return checkMethod, checkExcept, checkAllow, checkForbid
} // }}}

// 返回是否需要鉴权
func CheckMethod(ctx context.Context, checkMethod, checkExcept map[string]struct{}) bool { // {{{
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

func CheckAllow(ctx context.Context, appID string, checkAllows, checkForbids map[string]map[string]struct{}) bool { // {{{
	group, _ := ctx.Value("group").(string)
	controller, _ := ctx.Value("controller").(string)
	action, _ := ctx.Value("action").(string)

	checkAllow, _ := checkAllows[appID]
	checkForbid, _ := checkForbids[appID]

	if len(checkAllow) > 0 {
		_, exists := checkAllow[group]
		if !exists {
			_, exists = checkAllow[controller]
			if !exists {
				_, exists = checkAllow[controller+"/"+action]
				if !exists {
					return false
				}
			}
		}
	}

	if len(checkForbid) > 0 {
		if _, exists := checkForbid[group]; exists {
			return false
		}

		if _, exists := checkForbid[controller]; exists {
			return false
		}

		if _, exists := checkForbid[controller+"/"+action]; exists {
			return false
		}
	}

	return true
} // }}}

// 设置全局api鉴权函数
func SetApiAuthFn(fn func(*AuthConfig, *http.Request) (string, bool)) { // {{{
	DefaultApiAuthFn = fn
} // }}}

// 设置全局rpc鉴权函数
func SetRpcAuthFn(fn func(*AuthConfig, context.Context) (string, bool)) { // {{{
	DefaultRpcAuthFn = fn
} // }}}

// 用于生成调式数据
func MakeTokenData(appid, secret string) (string, string, string, string) {
	nonce := x.RandStr(8)
	timestamp := x.AsString(x.Now())

	x.Println("appid:", appid, "nonce:", nonce, "timestamp:", timestamp, "authorization: Bearer ", x.Sha256(appid+nonce+timestamp, secret))

	return appid, nonce, timestamp, x.Sha256(appid+nonce+timestamp, secret)
}
