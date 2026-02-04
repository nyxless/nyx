package x

import (
	"context"
	"net"
	"net/http"
	"strings"
)

var localIp string

func init() {
	localIp = getLocalIp()
}

// 解析 uri 得到 controller action params
func ParseRoute(uri, method string) (group, controller_name, action_name string, url_values MAPS) { // {{{
	url_values = MAPS{}
	uri = strings.Trim(uri, " \r\t\v/")

	if "" != uri { // {{{
		//全部转换为小写
		low_uri := strings.ToLower(uri)

		//先匹配不带参数路由规则
		if rule, ok := UrlRoutes[low_uri]; ok {
			if mtd, ok := rule["method"].(map[string]string); ok {
				if _, ok = mtd[method]; ok || len(mtd) == 0 {
					return rule["group"].(string), rule["controller"].(string), rule["action"].(string), url_values
				}
			}
		}

		path := strings.Split(low_uri, "/")
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
								ori_path := strings.Split(uri, "/")
								for i := 0; i < param_num; i++ {
									url_values[params[i]] = ori_path[path_num-(i+1)]
								}
							}

							return AsString(rule["group"]), AsString(rule["controller"]), AsString(rule["action"]), url_values
						}
					}
				}
			}
		} //}}}

		for _, prefix_rule := range UrlPrefix {
			prefix_from := prefix_rule["from"]
			prefix_to := prefix_rule["to"]

			if prefix_from != "" && strings.HasPrefix(low_uri, prefix_from) {
				low_uri = strings.Replace(low_uri, prefix_from, prefix_to, 1)
				group, controller_name, action_name = ParseUri(low_uri)
				return
			}

			if prefix_from == "" && prefix_to != "" {
				low_uri = Concat(prefix_to, "/", low_uri)
				group, controller_name, action_name = ParseUri(low_uri)
				return
			}
		}

		group, controller_name, action_name = parsePath(path)
	} // }}}

	if "" == controller_name {
		controller_name = ConfDefaultController
	}

	if "" == action_name {
		action_name = ConfDefaultAction
	}

	return
} // }}}

func ParseUri(uri string) (group, controller_name, action_name string) { // {{{
	uri = strings.Trim(uri, " \r\t\v/")
	path := strings.Split(uri, "/")

	return parsePath(path)
} // }}}

func parsePath(path []string) (group, controller_name, action_name string) { // {{{
	var current_path string

	for i := 0; i < len(path); i++ {

		if current_path != "" {
			current_path += "/" + path[i]
		} else {
			current_path = path[i]
		}

		if controller_name != "" {
			action_name = path[i]
		} else {
			controller_name = current_path
		}

		//如果存在 group 和 controller 命名冲突，优先匹配 group
		if _, ok := RouteGroups[current_path]; ok {
			group = current_path
			controller_name = ""
			action_name = ""
		}
	}

	if "" == controller_name {
		if group != "" {
			controller_name = group + "/" + ConfDefaultController
		} else {
			controller_name = ConfDefaultController
		}
	}

	if "" == action_name {
		action_name = ConfDefaultAction
	}

	return
} // }}}

// 获取本机ip
func getLocalIp() string { // {{{
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

func GetLocalIp() string { // {{{
	return localIp
} // }}}

// 获取 http 客户端 ip
func GetHttpIp(r *http.Request) string { // {{{
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
	ip_prefix, _, found := strings.Cut(ip, ":")
	if found {
		return ip_prefix
	}

	return ip
} // }}}

// context缓存中获取ip
func GetHttpCtxIp(ctx context.Context, r *http.Request) (context.Context, string) { // {{{
	if ip, ok := ctx.Value("ip").(string); ok {
		return ctx, ip
	}

	ip := GetHttpIp(r)
	return context.WithValue(ctx, "ip", ip), ip
} // }}}
