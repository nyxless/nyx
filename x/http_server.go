package x

import (
	"context"
	"embed"
	"fmt"
	"github.com/nyxless/nyx/x/endless"
	"io/fs"
	"log"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	//"runtime"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

var (
	defaultApis               = [][]any{}
	defaultRouteApiFuncs      = map[string]map[string]http.HandlerFunc{}
	defaultHttpMaxHeaderBytes = 0 // 0时, 将使用默认配置DefaultMaxHeaderBytes(1M)
	RouteGroups               = map[string]map[string]map[string]struct{}{}
)

var (
	staticUseEmbed  bool
	embedStatic     embed.FS
	embedStaticPath string
)

// 设置使用embed.FS
func StaticEmbed(filesys embed.FS, fs_path string) { // {{{
	staticUseEmbed = true
	embedStatic = filesys
	embedStaticPath = fs_path
} // }}}

// 添加http 方法对应的controller实例, 支持分组; 默认url路径: controller/action, 分组时路径: group/controller/action
func AddApi(c any, groups ...string) { // {{{
	group := ""
	if len(groups) > 0 {
		group = groups[0]
	}
	defaultApis = append(defaultApis, []any{c, group})
} // }}}

// 添加 route 调用方法, 在脚手架生成代码 (autoload/register_methods.go) 中调用
func AddRouteApiFunc(controller_name, action_name string, f http.HandlerFunc) { // {{{
	if defaultRouteApiFuncs[controller_name] == nil {
		defaultRouteApiFuncs[controller_name] = map[string]http.HandlerFunc{}
	}

	defaultRouteApiFuncs[controller_name][action_name] = f
} // }}}

// 设置MaxHeaderBytes, 单位M
func SetHttpMaxHeaderBytes(m int) { // {{{
	if m > 0 {
		defaultHttpMaxHeaderBytes = m << 20
	}
} // }}}

func NewHttpServer(addr string, port, rtimeout, wtimeout int, useGraceful, enable_pprof, enable_static bool, static_path, static_root string) *HttpServer { // {{{
	if "" != static_root && !filepath.IsAbs(static_root) {
		static_root = filepath.Join(AppRoot, static_root)
	}

	server := &HttpServer{
		addr:           addr,
		port:           port,
		rtimeout:       rtimeout,
		wtimeout:       wtimeout,
		useGraceful:    useGraceful,
		maxHeaderBytes: defaultHttpMaxHeaderBytes,
		handler: &httpHandler{
			routeMap:      make(map[string]map[string]reflect.Type),
			actionMap:     make(map[string]map[string]int),
			routeFuncs:    defaultRouteApiFuncs,
			groupHandlers: make(map[string]http.HandlerFunc),
			methodRule:    make(map[string]map[string]map[string]struct{}),
			enablePprof:   enable_pprof,
			enableStatic:  enable_static,
			staticPath:    "/" + strings.Trim(static_path, "/"),
			staticRoot:    static_root,
		},
	}

	server.handler.addControllers()
	server.handler.buildGroupHandlers()
	server.handler.parseMethodRule()

	return server
} // }}}

type HttpServer struct {
	addr           string
	port           int
	rtimeout       int
	wtimeout       int
	useGraceful    bool
	maxHeaderBytes int
	handler        *httpHandler
}

func (this *HttpServer) Run() {
	if len(this.handler.routeMap) == 0 {
		Warn("Api controller was not found, pls add controller using func `AddApi` or shell `nyx init`")
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	addr := fmt.Sprintf("%s:%d", this.addr, this.port)

	Info("HttpServer Listen", addr)

	rtimeout := time.Duration(this.rtimeout) * time.Millisecond
	wtimeout := time.Duration(this.wtimeout) * time.Millisecond

	//使用endless, 支持graceful reload
	if this.useGraceful {
		Warn(endless.ListenAndServe(addr, this.handler, rtimeout, wtimeout, this.maxHeaderBytes))
	} else {
		server := &http.Server{
			Addr:           addr,
			Handler:        this.handler,
			ReadTimeout:    rtimeout,
			WriteTimeout:   wtimeout,
			MaxHeaderBytes: this.maxHeaderBytes,
		}

		Warn(server.ListenAndServe())
	}
}

// controller中以此后缀结尾的方法会参与路由
const CONTROLLER_SUFFIX = "Controller"
const ACTION_SUFFIX = "Action"

type httpHandler struct {
	routeMap      map[string]map[string]reflect.Type        //key:controller: {key:method : value:reflect.type}
	actionMap     map[string]map[string]int                 //key:controller: {key:method : value:method_index}
	routeFuncs    map[string]map[string]http.HandlerFunc    //controller.action 函数缓存, 替换反射调用
	groupHandlers map[string]http.HandlerFunc               //group buildHttpMiddlewares 后的函数缓存
	methodRule    map[string]map[string]map[string]struct{} //r.Method 校验
	enablePprof   bool                                      //是否解析静态资源
	enableStatic  bool                                      //是否解析静态资源
	staticPath    string                                    //静态资源访问路径前缀
	staticRoot    string                                    //静态资源文件根目录
}

func (this *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { // {{{
	ctx := r.Context()

	defer func() {
		if err := recover(); err != nil {
			var errmsg string
			switch errinfo := err.(type) {
			case *Error:
				errmsg = errinfo.GetMessage()
			case error:
				errmsg = errinfo.Error()
				log.Println(errmsg)
				debug.PrintStack()
			default:
				errmsg = fmt.Sprint(errinfo)
			}

			log.Println("ServeHTTP: ", errmsg)
			http.Error(rw, errmsg, http.StatusInternalServerError)
		}
	}()

	rw.Header().Set("Server", "NYXServer")

	var group, controller_name, action_name string
	var url_values MAPS

	if this.enablePprof && strings.HasPrefix(r.URL.Path, "/debug/pprof") { //如果开启了pprof, 相关请求走DefaultServeMux
		this.monitorPprof(rw, r)
		return
	} else if this.enableStatic && strings.HasPrefix(r.URL.Path, this.staticPath) { //如果开启了静态资源服务, 相关请求走fileServrer
		this.serveFile(rw, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/status") { //用于lvs监控
		this.monitorStatus(rw, r)
		return
	}

	//根据路径路由: User/GetUserInfo
	group, controller_name, action_name, url_values = ParseRoute(r.URL.Path, r.Method)

	//校验 http METHOD
	if !this.checkMethod(controller_name+"/"+action_name, r.Method) {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if len(url_values) > 0 {
		new_query := r.URL.Query()
		for k, v := range url_values {
			new_query.Add(k, v)
		}
		r.URL.RawQuery = new_query.Encode()
	}

	ctx = context.WithValue(ctx, "group", group)
	ctx = context.WithValue(ctx, "controller", controller_name)
	ctx = context.WithValue(ctx, "action", action_name)

	r = r.WithContext(ctx)

	//路由解析之后加载中间件
	if hd, ok := this.groupHandlers[group]; ok {
		hd(rw, r)
	} else {
		this.defaultHandler(rw, r)
	}
} // }}}

func (this *httpHandler) defaultHandler(rw http.ResponseWriter, r *http.Request) { // {{{
	ctx := r.Context()

	defer func() {
		*r = *(r.WithContext(ctx))
	}()

	group := ctx.Value("group").(string)
	controller_name := ctx.Value("controller").(string)
	action_name := ctx.Value("action").(string)

	// 先尝试执行预生成函数
	if cf, ok := this.routeFuncs[controller_name]; ok {
		if f, ok := cf[action_name]; ok {
			f(rw, r)
			ctx = r.Context()
			return
		}
	}

	canhandler := false
	var controllerType reflect.Type
	if controller_name != "" && action_name != "" {
		if routMapSub, ok := this.routeMap[controller_name]; ok {
			if controllerType, ok = routMapSub[action_name]; ok {
				canhandler = true
			}
		}
	}

	if !canhandler {
		ctx = context.WithValue(ctx, "http_status_code", http.StatusNotFound)
		http.NotFound(rw, r)
		return
	}

	//未预生成代码，使用反射
	Info("Pre-generated code for " + controller_name + "/" + action_name + " was not found.  Reflection is used now OR U can gen code using shell `nyx init`")

	vc := reflect.New(controllerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if err := recover(); err != nil {
			in = []reflect.Value{reflect.ValueOf(err)}
			method := vc.Method(this.actionMap[controller_name]["RenderError"])
			method.Call(in)
		}

		in = make([]reflect.Value, 0)
		method := vc.Method(this.actionMap[controller_name]["Final"])
		method.Call(in)

		method = vc.Method(this.actionMap[controller_name]["HttpFinal"])
		method.Call(in)
	}()

	in = make([]reflect.Value, 5)
	in[0] = reflect.ValueOf(rw)
	in[1] = reflect.ValueOf(r)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	in[4] = reflect.ValueOf(group)
	method = vc.Method(this.actionMap[controller_name]["Prepare"])
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.Method(this.actionMap[controller_name]["Init"])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(this.actionMap[controller_name][action_name])
	method.Call(in)

} // }}}

// 预生成 middleware 函数缓存
func (this *httpHandler) buildGroupHandlers() { // {{{
	this.groupHandlers = map[string]http.HandlerFunc{}

	for group := range RouteGroups {
		if len(httpMiddlewares) > 0 {
			groupHandler := buildHttpMiddlewares(group, httpMiddlewares...)(http.HandlerFunc(this.defaultHandler))
			this.groupHandlers[group], _ = groupHandler.(http.HandlerFunc)
		} else {
			this.groupHandlers[group] = http.HandlerFunc(this.defaultHandler)
		}
	}
} // }}}

// 预生成 methodRule, 配置http_server.method_rule 转换-> { full_path: {"forbid": {"POST":{}}, "allow": {"GET":{},"PUT":{}}}} //full_path: group 和 controller 规则转换为 action 规则
func (this *httpHandler) parseMethodRule() { // {{{
	this.methodRule = map[string]map[string]map[string]struct{}{}

	fullpaths := make(map[string][]string)
	for group, controllers := range RouteGroups { // {{{
		for controller, actions := range controllers {
			for action := range actions {
				path := controller + "/" + action

				fullpaths[group] = append(fullpaths[group], path)
				fullpaths[controller] = append(fullpaths[controller], path)
				fullpaths[path] = []string{path}
			}
		}
	} // }}}

	conf_method_rule := Conf.GetMapSlice("http_server", "method_rule")
	for _, rule := range conf_method_rule { // {{{
		paths := AsStringSlice(rule["path"])
		allows := AsStringSlice(rule["allow"])
		forbids := AsStringSlice(rule["forbid"])
		for _, rulePath := range paths {
			path := strings.ToLower(rulePath)
			if cas, exists := fullpaths[path]; exists {
				for _, ca := range cas {
					if this.methodRule[ca] == nil {
						this.methodRule[ca] = make(map[string]map[string]struct{})
					}

					// 添加allow
					if len(allows) > 0 {
						if this.methodRule[ca]["allow"] == nil {
							this.methodRule[ca]["allow"] = make(map[string]struct{})
						}
						for _, v := range allows {
							this.methodRule[ca]["allow"][strings.ToUpper(v)] = struct{}{}
						}
					}

					// 添加forbid
					if len(forbids) > 0 {
						if this.methodRule[ca]["forbid"] == nil {
							this.methodRule[ca]["forbid"] = make(map[string]struct{})
						}
						for _, v := range forbids {
							this.methodRule[ca]["forbid"][strings.ToUpper(v)] = struct{}{}
						}
					}
				}
			}
		}
	} // }}}

} // }}}

// 校验 r.Method
func (this *httpHandler) checkMethod(path, method string) bool { // {{{
	if _, exists := this.methodRule[path]; exists {
		if _, exists := this.methodRule[path]["forbid"]; exists {
			if _, exists := this.methodRule[path]["forbid"][method]; exists {
				return false
			}
		}

		if _, exists := this.methodRule[path]["allow"]; exists {
			_, pass := this.methodRule[path]["allow"][method]
			return pass
		}
	}

	return true
} // }}}

// 静态资源服务
func (this *httpHandler) serveFile(rw http.ResponseWriter, r *http.Request) {
	// {{{
	var filesys http.FileSystem
	if staticUseEmbed {
		if embedStaticPath != "" {
			embedStaticPath = strings.Trim(embedStaticPath, "/")
			embedSub, err := fs.Sub(embedStatic, embedStaticPath)
			if err != nil {
				Panic(err)
			}
			filesys = http.FS(embedSub)
		} else {
			filesys = http.FS(embedStatic)
		}
	} else {
		filesys = http.Dir(this.staticRoot)
	}
	http.StripPrefix(this.staticPath, http.FileServer(filesys)).ServeHTTP(rw, r)
} // }}}

// pprof监控
func (this *httpHandler) monitorPprof(rw http.ResponseWriter, r *http.Request) {
	http.DefaultServeMux.ServeHTTP(rw, r)
}

// 用于lvs监控
func (this *httpHandler) monitorStatus(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok\n"))
}

func (this *httpHandler) addControllers() {
	for _, v := range defaultApis {
		this.addController(v[0], AsString(v[1]))
	}
}

func (this *httpHandler) addController(c interface{}, groups ...string) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.ToLower(strings.TrimSuffix(ct.Name(), CONTROLLER_SUFFIX))
	group := ""

	if len(groups) > 0 && groups[0] != "" {
		group = strings.ToLower(groups[0])
		controller_name = strings.Trim(group, " \r\t\v/") + "/" + controller_name
	}

	if _, ok := this.routeMap[controller_name]; ok {
		return
	}

	if RouteGroups[group] == nil {
		RouteGroups[group] = map[string]map[string]struct{}{}
	}

	RouteGroups[group][controller_name] = map[string]struct{}{}

	this.routeMap[controller_name] = make(map[string]reflect.Type)
	this.actionMap[controller_name] = make(map[string]int)

	var action_fullname string
	var action_name string
	for i := 0; i < rt.NumMethod(); i++ {
		action_fullname = rt.Method(i).Name
		if strings.HasSuffix(action_fullname, ACTION_SUFFIX) {
			action_name = strings.ToLower(strings.TrimSuffix(action_fullname, ACTION_SUFFIX))
			this.routeMap[controller_name][action_name] = ct
			this.actionMap[controller_name][action_name] = rt.Method(i).Index
			RouteGroups[group][controller_name][action_name] = struct{}{}
		}
	}
	methodRenderError, _ := rt.MethodByName("RenderError")
	methodPrepare, _ := rt.MethodByName("Prepare")
	methodInit, _ := rt.MethodByName("Init")
	methodFinal, _ := rt.MethodByName("Final")
	methodHttpFinal, _ := rt.MethodByName("HttpFinal")
	this.actionMap[controller_name]["RenderError"] = methodRenderError.Index
	this.actionMap[controller_name]["Prepare"] = methodPrepare.Index
	this.actionMap[controller_name]["Init"] = methodInit.Index
	this.actionMap[controller_name]["Final"] = methodFinal.Index
	this.actionMap[controller_name]["HttpFinal"] = methodHttpFinal.Index
} // }}}

//
//
//
