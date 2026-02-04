package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type CorsOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int

	// 是否包含 *
	hasWildcard bool
}

func Cors(opts *CorsOptions) func(http.Handler) http.Handler { // {{{
	if opts == nil {
		opts = &CorsOptions{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders: []string{"Content-Type", "Authorization", "Accept", "Origin", "X-Requested-With"},
			MaxAge:         86400,
		}
	}

	// 预处理：检查是否有通配符
	for i, origin := range opts.AllowedOrigins {
		if origin == "*" {
			opts.hasWildcard = true
			break
		} else {
			opts.AllowedOrigins[i] = extractDomain(origin)
		}
	}

	// 预拼接常用头部
	allowedMethods := strings.Join(opts.AllowedMethods, ", ")
	allowedHeaders := strings.Join(opts.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(opts.ExposedHeaders, ", ")

	maxage := ""
	if opts.MaxAge > 0 {
		maxage = strconv.Itoa(opts.MaxAge)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 请求来源域名
			requestDomain := extractDomain(origin)
			if requestDomain == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed := false
			if opts.hasWildcard {
				allowed = true
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowedDomain := range opts.AllowedOrigins {
					if isDomainAllowed(allowedDomain, requestDomain) {
						allowed = true
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			if allowed && opts.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// 处理预检请求
			if r.Method == "OPTIONS" && allowed {
				if allowedMethods != "" {
					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				}
				if allowedHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				}
				if maxage != "" {
					w.Header().Set("Access-Control-Max-Age", maxage)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// 非预检请求设置暴露头部
			if allowed && exposedHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			next.ServeHTTP(w, r)
		})
	}
} // }}}

// 从 URL 中提取域名
func extractDomain(url string) string { // {{{
	// 移除协议
	if idx := strings.Index(url, "://"); idx != -1 {
		url = url[idx+3:]
	}

	// 移除端口, 会同时移除路径(假如存在)
	if idx := strings.Index(url, ":"); idx != -1 {
		url = url[:idx]
		//移除路径(假如存在)
	} else if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
} // }}}

// 判断域名是否允许访问
func isDomainAllowed(allowedDomain, requestDomain string) bool { // {{{
	if allowedDomain == "*" {
		return true
	}

	if allowedDomain == requestDomain {
		return true
	}

	// 通配符域名匹配 (如 *.example.com)
	if strings.HasPrefix(allowedDomain, "*.") {
		baseDomain := strings.TrimPrefix(allowedDomain, "*.")

		if requestDomain == baseDomain {
			return true
		}

		if strings.HasSuffix(requestDomain, "."+baseDomain) {
			// 确保只匹配一级子域名
			wildcardParts := strings.Split(baseDomain, ".")
			requestParts := strings.Split(requestDomain, ".")

			if len(requestParts) == len(wildcardParts)+1 {
				return true
			}
		}
	}

	return false
} // }}}
