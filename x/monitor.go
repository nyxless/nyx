package x

import (
	"net/http"
	_ "net/http/pprof"
	"strings"
)

func RunMonitor(port string) { // {{{
	Info("monitor Listen: ", port)
	http.ListenAndServe(":"+port, &monitorHandler{})
} // }}}

type monitorHandler struct {
}

func (m *monitorHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { // {{{
	if ConfPprofEnabled && strings.HasPrefix(r.URL.Path, "/debug/pprof") { //如果开启了pprof, 相关请求走DefaultServeMux
		http.DefaultServeMux.ServeHTTP(rw, r)
		return
	} else if ConfDebugRpcEnabled && strings.HasPrefix(r.URL.Path, "/debug/rpc/") { //如果开启了 rpc 选项, 可使用 http 协议代理方式调式 rpc 方法
		DebugRpc(rw, r)
		return
	} else if strings.HasPrefix(r.URL.Path, ConfMonitorPath) { //用于lvs监控
		rw.Write([]byte("ok\n"))
		return
	}
} // }}}
