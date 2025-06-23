package x

import (
	"embed"
	"fmt"
	"github.com/nyxless/nyx/x/endless"
	"io/fs"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"reflect"
	//"runtime"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

// 执行路由方法的函数
type RouteApiFunc func(http.ResponseWriter, *http.Request)

var (
	defaultApis               = [][]any{}
	defaultRouteApiFuncs      = map[string]map[string]RouteApiFunc{}
	defaultHttpMaxHeaderBytes = 0 //0时, 将使用默认配置DefaultMaxHeaderBytes(1M)
	routGroups                = map[string]int{}
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

// 添加 route 调用方法, 在脚手架生成代码时使用
func AddRouteApiFunc(controller_name, action_name string, f RouteApiFunc) { // {{{
	if defaultRouteApiFuncs[controller_name] == nil {
		defaultRouteApiFuncs[controller_name] = map[string]RouteApiFunc{}
	}

	defaultRouteApiFuncs[controller_name][action_name] = f
} // }}}

// 设置MaxHeaderBytes, 单位M
func SetHttpMaxHeaderBytes(m int) { // {{{
	if m > 0 {
		defaultHttpMaxHeaderBytes = m << 20
	}
} // }}}

func NewHttpServer(addr string, port, timeout int, useGraceful, enable_pprof, enable_static bool, static_path, static_root string) *HttpServer { // {{{
	if "" != static_root && !filepath.IsAbs(static_root) {
		static_root = filepath.Join(AppRoot, static_root)
	}

	server := &HttpServer{
		addr:           addr,
		port:           port,
		timeout:        timeout,
		useGraceful:    useGraceful,
		maxHeaderBytes: defaultHttpMaxHeaderBytes,
		handler: &httpHandler{
			routMap:      make(map[string]map[string]reflect.Type),
			methodMap:    make(map[string]map[string]int),
			routeFuncs:   defaultRouteApiFuncs,
			enablePprof:  enable_pprof,
			enableStatic: enable_static,
			staticPath:   "/" + strings.Trim(static_path, "/"),
			staticRoot:   static_root,
		},
	}

	for _, v := range defaultApis {
		server.AddController(v[0], AsString(v[1]))
	}

	return server
} // }}}

type HttpServer struct {
	addr           string
	port           int
	timeout        int
	useGraceful    bool
	maxHeaderBytes int
	handler        *httpHandler
}

func (this *HttpServer) AddController(c interface{}, group ...string) {
	this.handler.addController(c, group...)
}

func (this *HttpServer) Run() {
	if len(this.handler.routMap) == 0 || len(this.handler.routeFuncs) == 0 {
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	addr := fmt.Sprintf("%s:%d", this.addr, this.port)

	Println("HttpServer Listen", addr)

	rtimeout := time.Duration(this.timeout) * time.Millisecond
	wtimeout := time.Duration(this.timeout) * time.Millisecond

	//使用endless, 支持graceful reload
	if this.useGraceful {
		log.Println(endless.ListenAndServe(addr, this.handler, rtimeout, wtimeout, this.maxHeaderBytes))
	} else {
		server := &http.Server{
			Addr:           addr,
			Handler:        this.handler,
			ReadTimeout:    rtimeout,
			WriteTimeout:   wtimeout,
			MaxHeaderBytes: this.maxHeaderBytes,
		}

		log.Println(server.ListenAndServe())
	}
}

// controller中以此后缀结尾的方法会参与路由
const CONTROLLER_SUFFIX = "Controller"
const ACTION_SUFFIX = "Action"

type httpHandler struct {
	routMap      map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	methodMap    map[string]map[string]int          //key:controller: {key:method value:method_index}
	routeFuncs   map[string]map[string]RouteApiFunc //controller.action 函数缓存, 替换反射调用
	enablePprof  bool                               //是否解析静态资源
	enableStatic bool                               //是否解析静态资源
	staticPath   string                             //静态资源访问路径前缀
	staticRoot   string                             //静态资源文件根目录
}

func (this *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) { // {{{
	defer func() {
		if err := recover(); err != nil {
			var errmsg string
			switch errinfo := err.(type) {
			case *Error:
				errmsg = errinfo.GetMessage()
			case *Errorf:
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
	//跨域设置
	ref := r.Referer()
	if "" == ref {
		ref = r.Header.Get("Origin")
	}
	if ref != "" {
		if u, err := url.Parse(ref); nil == err {
			cors_domain := Conf.GetString("cors_domain")
			if len(cors_domain) > 0 {
				allowed := false
				if "*" == cors_domain || strings.Contains(","+cors_domain+",", ","+u.Host+",") {
					allowed = true
				} else if strings.Contains(","+cors_domain, ",*.") {
					domains := strings.Split(cors_domain, ",")
					for _, v := range domains {
						if v[0] == '*' && strings.Contains(u.Host+",", string(v[1:])+",") {
							allowed = true
							break
						}
					}
				}

				if allowed {
					rw.Header().Set("Access-Control-Allow-Origin", u.Scheme+"://"+u.Host)
					rw.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
		}
	}

	var controller_name, action_name string
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
	} else { //根据路径路由: User/GetUserInfo
		controller_name, action_name, url_values = ParseRoute(r.URL.Path, r.Method)

		if len(url_values) > 0 {
			new_query := r.URL.Query()
			for k, v := range url_values {
				new_query.Add(k, v)
			}
			r.URL.RawQuery = new_query.Encode()
		}
	}

	// 先尝试执行预生成函数代码
	if cf, ok := this.routeFuncs[controller_name]; ok {
		if f, ok := cf[action_name]; ok {
			f(rw, r)
			return
		}
	}

	canhandler := false
	var controllerType reflect.Type
	if controller_name != "" && action_name != "" {
		if routMapSub, ok := this.routMap[controller_name]; ok {
			if controllerType, ok = routMapSub[action_name]; ok {
				canhandler = true
			}
		}
	}

	if !canhandler {
		http.NotFound(rw, r)
		return
	}

	vc := reflect.New(controllerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if err := recover(); err != nil {
			in = []reflect.Value{reflect.ValueOf(err)}
			method := vc.Method(this.methodMap[controller_name]["RenderError"])
			method.Call(in)
		}
	}()

	in = make([]reflect.Value, 4)
	in[0] = reflect.ValueOf(rw)
	in[1] = reflect.ValueOf(r)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	method = vc.Method(this.methodMap[controller_name]["Prepare"])
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name]["Init"])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name][action_name])
	method.Call(in)
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
				panic(err)
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

func (this *httpHandler) addController(c interface{}, group ...string) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), CONTROLLER_SUFFIX)
	if len(group) > 0 && group[0] != "" {
		routGroups[group[0]] = 1
		controller_name = strings.Trim(group[0], " \r\t\v/") + "/" + controller_name
	}

	controller_name = strings.ToLower(controller_name)

	if _, ok := this.routMap[controller_name]; ok {
		return
	} else {
		this.routMap[controller_name] = make(map[string]reflect.Type)
		this.methodMap[controller_name] = make(map[string]int)
	}
	var action_fullname string
	var action_name string
	for i := 0; i < rt.NumMethod(); i++ {
		action_fullname = rt.Method(i).Name
		if strings.HasSuffix(action_fullname, ACTION_SUFFIX) {
			action_name = strings.ToLower(strings.TrimSuffix(action_fullname, ACTION_SUFFIX))
			this.routMap[controller_name][action_name] = ct
			this.methodMap[controller_name][action_name] = rt.Method(i).Index
		}
	}
	methodRenderError, _ := rt.MethodByName("RenderError")
	methodPrepare, _ := rt.MethodByName("Prepare")
	methodInit, _ := rt.MethodByName("Init")
	this.methodMap[controller_name]["RenderError"] = methodRenderError.Index
	this.methodMap[controller_name]["Prepare"] = methodPrepare.Index
	this.methodMap[controller_name]["Init"] = methodInit.Index
} // }}}

//
//
//
