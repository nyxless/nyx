package x

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// 解析 uri 得到 controller action params
func ParseRoute(uri, method string) (string, string, MAPS) { // {{{
	var group, controller_name, action_name string

	url_values := MAPS{}

	uri = strings.Trim(uri, " \r\t\v/")

	if "" != uri { // {{{
		//先匹配不带参数路由规则
		if rule, ok := UrlRoutes[uri]; ok {
			if mtd, ok := rule["method"].(map[string]string); ok {
				if _, ok = mtd[method]; ok || len(mtd) == 0 {
					return rule["controller"].(string), rule["action"].(string), url_values
				}
			}
		}

		path := strings.Split(uri, "/")
		path_num := len(path)

		//根据path层数匹配带参数路由规则
		if route, ok := UrlParamRoutes[path_num]; ok { // {{{
			for param_num, rules := range route {
				//处理uri,弹出对应个数的参数值
				trimed_uri := strings.Join(path[:path_num-param_num], "/")
				if rule, ok := rules[trimed_uri]; ok {
					if mtd, ok := rule["method"].(map[string]string); ok {
						if _, ok = mtd[method]; ok || len(mtd) == 0 {
							//处理参数
							if params, ok := rule["params"].([]string); ok {
								for i := 0; i < param_num; i++ {
									url_values[params[i]] = path[path_num-(i+1)]
								}
							}

							return AsString(rule["controller"]), AsString(rule["action"]), url_values
						}
					}
				}
			}
		} //}}}

		for _, prefix_rule := range UrlPrefix {
			prefix_from := prefix_rule["from"]
			prefix_to := prefix_rule["to"]

			if prefix_from != "" && strings.HasPrefix(uri, prefix_from) {
				uri = strings.Replace(uri, prefix_from, prefix_to, 1)
				uri = strings.Trim(uri, " \r\t\v/")
				path = strings.Split(uri, "/")
				path_num = len(path)

				break
			}

			if prefix_from == "" && prefix_to != "" {
				uri = Concat(prefix_to, "/", uri)
				path = strings.Split(uri, "/")
				path_num = len(path)

				break
			}
		}

		//未匹配到路由配置，解析URI
		for i := 0; i < path_num; i++ {

			if group != "" {
				group += "/" + path[i]
			} else {
				group = path[i]
			}

			if controller_name != "" {
				action_name = path[i]
			} else {
				controller_name = group
			}

			if _, ok := routGroups[group]; ok {
				controller_name = ""
				action_name = ""
			}
		}
	} // }}}

	if "" == controller_name {
		if group != "" {
			controller_name = group + "/" + Conf_default_controller
		} else {
			controller_name = Conf_default_controller
		}
	}

	if "" == action_name {
		action_name = Conf_default_action
	}

	return strings.ToLower(controller_name), strings.ToLower(action_name), url_values
} // }}}

// 获取本机ip
func GetLocalIp() string { // {{{
	addrs, _ := net.InterfaceAddrs()
	var ip string = ""
	for _, addr := range addrs {
		//判断是否回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ip
} // }}}

func GetIp(r *http.Request) string { // {{{
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" || ip == "127.0.0.1" || ip == GetLocalIp() {
		ip = r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.RemoteAddr
		}
	} else {
		//X-Forwarded-For 的格式 client1, proxy1, proxy2
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			ip = ips[0]
		}
	}

	//去除端口号
	ips := strings.Split(ip, ":")
	if len(ips) > 0 {
		ip = ips[0]
	}

	return ip
} // }}}

func GetCookie(r *http.Request, key string) string { // {{{
	cookie, err := r.Cookie(key)
	if err == nil {
		return cookie.Value
	}

	return ""
} // }}}

// lifetime<0时删除cookie
// options: domain,secure,httponly,path
func SetCookie(rw http.ResponseWriter, key, val string, lifetime int, options ...interface{}) { // {{{
	domain := ""
	secure := false
	httponly := false
	path := "/"

	if len(options) > 0 {
		domain = AsString(options[0])
	}

	if len(options) > 1 {
		secure = AsBool(options[1])
	}

	if len(options) > 2 {
		httponly = AsBool(options[2])
	}

	if len(options) > 3 {
		path = AsString(options[3])
	}

	cookie := &http.Cookie{
		Name:   key,
		Value:  val,
		Path:   path,
		Domain: domain,
		Secure: secure,
		//SameSite: http.SameSiteNoneMode,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: httponly,
		MaxAge:   lifetime,
		Expires:  time.Now().Add(time.Second * time.Duration(lifetime)),
	}
	http.SetCookie(rw, cookie)
} // }}}
