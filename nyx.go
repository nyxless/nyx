package nyx

import (
	"flag"
	"fmt"
	"github.com/nyxless/nyx/middleware"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/cache"
	"github.com/nyxless/nyx/x/log"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	SYSTEM_VERSION = "1.0.0"

	SERVER_HTTP = "http"
	SERVER_RPC  = "rpc"
	SERVER_TCP  = "tcp"
	SERVER_WS   = "ws"
	SERVER_CLI  = "cli"
)

type Nyx struct {
	Mode      []string
	cliPath   string
	cliParams string
}

func NewNyx() *Nyx {
	nyx := &Nyx{}
	nyx.Init()

	return nyx
}

func (this *Nyx) Init() { // {{{
	this.printLogo()
	this.envInit()
	this.genPidFile()
} // }}}

func (this *Nyx) printLogo() { // {{{
	logo := `
       _/      _/  _/      _/  _/      _/   
      _/_/    _/    _/  _/      _/  _/
     _/  _/  _/      _/          _/
    _/    _/_/      _/        _/  _/
   _/      _/      _/      _/      _/    ` + "Version: " + SYSTEM_VERSION + ` 
-------------------------------------------------------
`
	if x.IsTerm() {
		fmt.Printf("\033[1;31;33m%s\033[0m\n", logo)
	}
} // }}}

// 初始化
func (this *Nyx) envInit() { // {{{
	var config_file, mode, uri, params string
	var debug bool

	flag.StringVar(&config_file, "c", "", "config file")
	flag.BoolVar(&debug, "d", false, "use debug mode")
	flag.StringVar(&mode, "m", "", "run mode, http,rpc,tcp,ws,cli") // 支持同时运行多个, 逗号分隔

	flag.StringVar(&uri, "p", "", "path when cli-mode")
	flag.StringVar(&params, "q", "", "params when cli-mode")

	flag.Parse()

	x.Debug = debug //全局 debug 开关

	app_root, err := os.Getwd()
	if err != nil {
		x.Warn("Error: ", err)
		os.Exit(0)
	}

	if path.Base(app_root) == "bin" {
		app_root = path.Dir(app_root)
	}

	x.AppRoot = app_root //服务根路径

	config, config_file, err := x.NewConfig(config_file, x.AppRoot+"/conf/app.conf", "../conf/app.conf", "./conf/app.conf", "app.conf") //按顺序寻找配置文件
	if nil != err {
		x.Warn("Error: ", err)
		os.Exit(0)
	}

	x.Info("Config Init: ", config_file)

	x.Conf = config

	if mode != "" {
		this.Mode = strings.Split(mode, ",")
	}

	if uri != "" {
		this.cliPath = uri
	}

	if params != "" {
		this.cliParams = params
	}

	if time_zone := x.Conf.GetString("time_zone"); time_zone != "" {
		x.TIME_ZONE = time_zone
	}

	// 缓存配置文件变量(框架中用到的)
	this.cacheConf()

	// 初始化日志
	err = this.useLogger()
	if nil != err {
		x.Println("Error: ", err)
		os.Exit(0)
	}

	// 初始化本地缓存
	this.useLocalCache()

	x.Info("Run Cmd: ", os.Args)
	if x.Debug {
		x.Info("Debug model: ", x.Colorize("open", "green+bold+underline"))
	}
	x.Info("Time: ", time.Now().Format("2006-01-02 15:04:05"))
} // }}}

// 加载 http 中间件
func (this *Nyx) useHttpMiddlewares() { // {{{
	// 加载 http Cors 中间件
	if x.Conf.GetDefBool(false, "cors", "enabled") { // {{{
		opts := &middleware.CorsOptions{
			AllowedOrigins:   x.Conf.GetDefStringSlice([]string{"*"}, "cors", "allowed_origins"),
			AllowedMethods:   x.Conf.GetDefStringSlice([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}, "cors", "allowed_methods"),
			AllowedHeaders:   x.Conf.GetDefStringSlice([]string{"Content-Type", "Authorization", "Accept", "Origin", "X-Requested-With"}, "cors", "allowed_headers"),
			AllowCredentials: x.Conf.GetDefBool(false, "cors", "allow_credentials"),
			MaxAge:           x.Conf.GetDefInt(86400, "cors", "max_age"),
			ExposedHeaders:   x.Conf.GetDefStringSlice([]string{}, "cors", "exposed_headers"),
		}

		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "cors", "allowed_groups")
		x.UseHttpMiddleware(middleware.Cors(opts), allowed_groups...)

		x.Info("Load http middleware: ", "Cors")
	} // }}}

	// 加载 http auth 中间件
	if x.Conf.GetDefBool(false, "auth", "api_check", "enabled") { // {{{
		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "auth", "api_check", "allowed_groups")

		confAuthApp := map[string]string{}
		confAuthAppApiAllow := map[string][]string{}
		confAuthAppApiForbid := map[string][]string{}

		for _, v := range x.Conf.GetMapSlice("auth", "app") {
			appid := x.AsString(v["appid"])
			confAuthApp[appid] = x.AsString(v["secret"])

			confAuthAppApiAllow[appid] = x.AsStringSlice(v["api_allow"])
			confAuthAppApiForbid[appid] = x.AsStringSlice(v["api_forbid"])
		}

		c := &middleware.AuthConfig{
			AppSecrets:  confAuthApp,
			CheckTTL:    x.Conf.GetDefInt(3600, "auth", "api_check", "ttl"),
			CheckNonce:  x.Conf.GetDefBool(false, "auth", "api_check", "check_nonce"),
			CheckMethod: x.Conf.GetStringSlice("auth", "api_check", "method"),
			CheckExcept: x.Conf.GetStringSlice("auth", "api_check", "except"),
			CheckAllow:  confAuthAppApiAllow,
			CheckForbid: confAuthAppApiForbid,
			LocalCache:  x.LocalCache,
		}

		x.UseHttpMiddleware(middleware.ApiAuth(c), allowed_groups...)

		x.Info("Load http middleware: ", "ApiAuth")
	} // }}}

	// 加载 http Compress 中间件
	if x.Conf.GetDefBool(false, "compress", "enabled") { // {{{
		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "compress", "allowed_groups")

		c := &middleware.CompressConfig{
			MinSize:      x.Conf.GetInt("compress", "min_size"),
			GzipLevel:    x.Conf.GetInt("compress", "gzip_level"),
			DeflateLevel: x.Conf.GetInt("compress", "deflate_level"),
		}

		x.UseHttpMiddleware(middleware.Compress(c), allowed_groups...)

		x.Info("Load http middleware: ", "Compress")
	} // }}}

	// 加载 http log 中间件
	if x.Conf.GetDefBool(false, "http_log", "enabled") && x.Logger != nil { // {{{
		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "http_log", "allowed_groups")

		c := &middleware.LogConfig{
			InfoLogName:    x.Conf.GetString("http_log", "info_level_name"),
			WarnLogName:    x.Conf.GetString("http_log", "warn_level_name"),
			ErrorLogName:   x.Conf.GetString("http_log", "error_level_name"),
			CheckMethod:    x.Conf.GetStringSlice("http_log", "method"),
			CheckExcept:    x.Conf.GetStringSlice("http_log", "except"),
			CheckReqMethod: x.Conf.GetStringSlice("http_log", "req_method"),
			CheckReqExcept: x.Conf.GetStringSlice("http_log", "req_except"),
			CheckResMethod: x.Conf.GetStringSlice("http_log", "res_method"),
			CheckResExcept: x.Conf.GetStringSlice("http_log", "res_except"),
			Logger:         x.Logger,
		}

		x.UseHttpMiddleware(middleware.HttpLog(c), allowed_groups...)

		x.Info("Load http middleware: ", "HttpLog")
	} // }}}
} // }}}

// 加载 rpc 中间件
func (this *Nyx) useRpcMiddlewares() { // {{{
	// 加载 rpc auth 中间件
	if x.Conf.GetDefBool(false, "auth", "rpc_check", "enabled") { // {{{
		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "auth", "rpc_check", "allowed_groups")

		confAuthApp := map[string]string{}
		confAuthAppRpcAllow := map[string][]string{}
		confAuthAppRpcForbid := map[string][]string{}

		for _, v := range x.Conf.GetMapSlice("auth", "app") {
			appid := x.AsString(v["appid"])
			confAuthApp[appid] = x.AsString(v["secret"])

			confAuthAppRpcAllow[appid] = x.AsStringSlice(v["rpc_allow"])
			confAuthAppRpcForbid[appid] = x.AsStringSlice(v["rpc_forbid"])
		}

		c := &middleware.AuthConfig{
			AppSecrets:  confAuthApp,
			CheckTTL:    x.Conf.GetDefInt(3600, "auth", "rpc_check", "ttl"),
			CheckNonce:  x.Conf.GetDefBool(false, "auth", "rpc_check", "check_nonce"),
			CheckMethod: x.Conf.GetStringSlice("auth", "rpc_check", "method"),
			CheckExcept: x.Conf.GetStringSlice("auth", "rpc_check", "except"),
			CheckAllow:  confAuthAppRpcAllow,
			CheckForbid: confAuthAppRpcForbid,
			LocalCache:  x.LocalCache,
		}

		x.UseRpcMiddleware(middleware.RpcAuth(c), allowed_groups...)

		x.Info("Load rpc middleware: ", "RpcAuth")
	} // }}}

	// 加载 rpc log 中间件
	if x.Conf.GetDefBool(false, "rpc_log", "enabled") && x.Logger != nil { // {{{
		allowed_groups := x.Conf.GetDefStringSlice([]string{}, "rpc_log", "allowed_groups")

		c := &middleware.LogConfig{
			InfoLogName:    x.Conf.GetString("rpc_log", "info_level_name"),
			WarnLogName:    x.Conf.GetString("rpc_log", "warn_level_name"),
			ErrorLogName:   x.Conf.GetString("rpc_log", "error_level_name"),
			CheckMethod:    x.Conf.GetStringSlice("rpc_log", "method"),
			CheckExcept:    x.Conf.GetStringSlice("rpc_log", "except"),
			CheckReqMethod: x.Conf.GetStringSlice("rpc_log", "req_method"),
			CheckReqExcept: x.Conf.GetStringSlice("rpc_log", "req_except"),
			CheckResMethod: x.Conf.GetStringSlice("rpc_log", "res_method"),
			CheckResExcept: x.Conf.GetStringSlice("rpc_log", "res_except"),
			Logger:         x.Logger,
		}

		x.UseRpcMiddleware(middleware.RpcLog(c), allowed_groups...)

		x.Info("Load rpc middleware: ", "RpcLog")
	} // }}}

} // }}}

// 缓存配置文件变量
func (this *Nyx) cacheConf() { // {{{
	x.ConfEnvMode = x.Conf.GetString("env_mode")
	x.ConfGuidKey = x.Conf.GetDefString("guid", "guid_key")
	x.ConfLangKey = x.Conf.GetDefString("lang", "lang_key")
	x.ConfTemplateEnabled = x.Conf.GetDefBool(false, "http_server", "template", "enabled")
	x.ConfTemplateRoot = x.Conf.GetDefString("../templates", "http_server", "template", "root")
	x.ConfTemplateRecursionLimit = x.Conf.GetDefInt(3, "http_server", "template", "recursion_limit")
	x.ConfMaxPostSize = int64(x.Conf.GetDefInt(32, "http_server", "max_post_size") << 20)
	x.ConfHttpLogOmitParams = x.Conf.GetStringSlice("http_log", "omit_params")
	x.ConfRpcLogOmitParams = x.Conf.GetStringSlice("rpc_log", "omit_params")
	x.ConfDefaultController = strings.ToLower(x.Conf.GetDefString("index", "default_controller"))
	x.ConfDefaultAction = strings.ToLower(x.Conf.GetDefString("index", "default_action"))

	this.parseRouter()
	this.parseErrMsg()
} // }}}

// 解析转换url路由配置
func (this *Nyx) parseRouter() { // {{{
	//prefix := strings.Trim(x.Conf.GetString("url_route", "prefix"), " \r\t\v/")
	routes := x.Conf.GetMapSlice("url_route")

	//[map[group:user handler:Hello/GetUserInfo method:GET path:/info] map[handler:Hello/GetUserInfo path:/users/:id]]
	url_routes := map[string]map[string]any{}
	url_param_routes := map[int]map[int]map[string]map[string]any{}
	url_prefix := []map[string]string{}

	for _, route := range routes { // {{{
		if !x.AsBool(route["enabled"], true) {
			continue
		}

		prefix := ""
		if _prefix, ok := route["prefix"]; ok {
			prefix = strings.Trim(x.AsString(_prefix), " \r\t\v/")
		}

		if _group_rules, ok := route["group_rule"]; ok {
			group_rules, ok := _group_rules.([]any)
			if !ok {
				panic("路由配置有误: group_rule")
			}

			for _, _group_rule := range group_rules {
				group_rule, ok := _group_rule.(map[string]any)
				if !ok {
					panic("路由配置有误: group_rule")
				}

				_from, ok := group_rule["from"]
				if !ok {
					panic("路由配置有误: group_rule->from")
				}

				_to, ok := group_rule["to"]
				if !ok {
					panic("路由配置有误: group_rule->to")
				}

				from := strings.Trim(x.AsString(_from), " \r\t\v/")
				to := strings.Trim(x.AsString(_to), " \r\t\v/")

				if prefix != "" {
					from = x.Concat(prefix, "/", from)
					from = strings.Trim(from, "/")
				}

				//过滤空值
				if from != "" || to != "" {
					url_prefix = append(url_prefix, map[string]string{"from": strings.ToLower(from), "to": strings.ToLower(to)})
				}
			}
		}

		_path_rules, ok := route["path_rule"]
		if !ok {
			continue
		}

		path_rules, ok := _path_rules.([]any)
		if !ok {
			panic("路由配置有误: path_rule")
		}

		for _, _path_rule := range path_rules {
			path_rule, ok := _path_rule.(map[string]any)
			if !ok {
				panic("路由配置有误: path_rule")
			}

			var group, controller, action string
			handler := strings.Trim(x.AsString(path_rule["to"]), " \r\t\v/")
			if handler != "" {
				handler = strings.ToLower(handler)
				group, controller, action = x.ParseUri(handler)
			}

			path := strings.Trim(x.AsString(path_rule["from"]), " \r\t\v/")
			//if path_rule["group"] != "" {
			//	path = x.Concat(path_rule["group"], "/", path)
			//}

			if prefix != "" {
				path = x.Concat(prefix, "/", path)
			}

			//转换为小写(路由路径不区分大小写)
			path = strings.ToLower(path)

			path_num := strings.Count(path, "/") + 1
			var params []string
			for {
				idx := strings.LastIndex(path, "@")
				if idx == -1 {
					break
				}

				params = append(params, path[idx+1:])
				path = strings.Trim(path[:idx], " \r\t\v/")
			}

			methods := map[string]string{}
			if _method, ok := path_rule["method"]; ok {
				if mtd, ok := _method.(string); ok {
					methods[mtd] = mtd
				} else if mtds, ok := _method.([]any); ok {
					for _, _mtd := range mtds {
						mtd := x.AsString(_mtd)
						methods[mtd] = mtd
					}
				}
			}

			if len(params) > 0 {
				if url_param_routes[path_num] == nil {
					url_param_routes[path_num] = map[int]map[string]map[string]any{}
				}
				if url_param_routes[path_num][len(params)] == nil {
					url_param_routes[path_num][len(params)] = map[string]map[string]any{}
				}

				url_param_routes[path_num][len(params)][path] = map[string]any{
					"group":      group,
					"controller": controller,
					"action":     action,
					"method":     methods,
					"params":     params,
				}
			} else {
				url_routes[path] = map[string]any{
					"group":      group,
					"controller": controller,
					"action":     action,
					"method":     methods,
				}
			}
		}
	} // }}}

	x.UrlRoutes = url_routes
	x.UrlParamRoutes = url_param_routes
	x.UrlPrefix = url_prefix
} // }}}

// 解析 err_msg
func (this *Nyx) parseErrMsg() { // {{{
	x.DEFAULT_LANG = x.Conf.GetDefString("CN", "default_lang")

	x.ErrMapRo = x.MapMerge(x.ErrMapRo, x.ErrMap)

	err_msg := x.Conf.GetMap("err_msg")
	for code, msgs := range err_msg {
		x.ErrMapRo[x.AsInt32(code)] = x.AsStringMap(msgs)
	}
} // }}}

// 初始化日志
func (this *Nyx) useLogger() error { // {{{
	log_enabled := x.Conf.GetDefBool(true, "log", "enabled")
	if !log_enabled {
		return nil
	}

	log_rule := x.Conf.GetMap("log", "file_rule")
	log_level_rule := x.Conf.GetMap("log", "file_level_rule")

	def_path := x.AsString(log_rule["path"])
	if !filepath.IsAbs(def_path) {
		def_path = filepath.Join(x.AppRoot, def_path)
	}

	file_rule := &log.LogFileRule{
		Path:           def_path,
		NamingFormat:   x.AsString(log_rule["naming_format"]),
		BufferSize:     x.AsInt(log_rule["buffer_size"]),
		FileSize:       x.AsInt(log_rule["file_size"]),
		Compress:       x.AsBool(log_rule["compress"]),
		CompressBefore: x.AsInt(log_rule["compress_before"]),
		Remove:         x.AsBool(log_rule["remove"]),
		RemoveBefore:   x.AsInt(log_rule["remove_before"]),
	}

	file_level_rule := map[string]*log.LogFileRule{}
	for level_name, rule := range log_level_rule {
		node, _ := x.GetNode(rule, "path")
		logpath := x.AsString(node, file_rule.Path)

		if !filepath.IsAbs(logpath) {
			logpath = filepath.Join(x.AppRoot, logpath)
		}

		namingFormatNode, _ := x.GetNode(rule, "naming_format")
		bufferSizeNode, _ := x.GetNode(rule, "buffer_size")
		fileSizeNode, _ := x.GetNode(rule, "file_size")
		compressNode, _ := x.GetNode(rule, "compress")
		compressBeforeNode, _ := x.GetNode(rule, "compress_before")
		removeNode, _ := x.GetNode(rule, "remove")
		removeBeforeNode, _ := x.GetNode(rule, "remove_before")

		file_level_rule[level_name] = &log.LogFileRule{
			Path:           logpath,
			NamingFormat:   x.AsString(namingFormatNode, file_rule.NamingFormat),
			BufferSize:     x.AsInt(bufferSizeNode, file_rule.BufferSize),
			FileSize:       x.AsInt(fileSizeNode, file_rule.FileSize),
			Compress:       x.AsBool(compressNode, file_rule.Compress),
			CompressBefore: x.AsInt(compressBeforeNode, file_rule.CompressBefore),
			Remove:         x.AsBool(removeNode, file_rule.Remove),
			RemoveBefore:   x.AsInt(removeBeforeNode, file_rule.RemoveBefore),
		}
	}

	log_options := &log.LogOptions{
		QueueSize:     x.Conf.GetDefInt(1024, "log", "queue_size"),
		Level:         x.Conf.GetDefInt(0x0F, "log", "level"),
		TraceFile:     x.Conf.GetDefBool(false, "log", "trace_file"),
		UseQueue:      x.Conf.GetDefBool(true, "log", "use_queue"),
		BulkSize:      x.Conf.GetDefInt(32, "log", "bulk_size"),
		FileEnabled:   x.Conf.GetDefBool(true, "log", "file_enabled"),
		FileRule:      file_rule,
		FileLevelRule: file_level_rule,
		ShowLevel:     x.Conf.GetDefBool(true, "log", "show_level"),
		Prefix:        x.Conf.GetString("log", "prefix"),
		TimeFormat:    x.Conf.GetString("log", "time_format"),
	}

	var err error
	x.Logger, err = log.NewLogger(log_options)
	if err != nil {
		return err
	}

	if x.Debug {
		x.Logger.SetDebug(true)
		x.Logger.SetLevel(log.LevelAll)
	}

	return nil
} // }}}

// 初始化本地缓存
func (this *Nyx) useLocalCache() { // {{{
	localcache_enabled := x.Conf.GetDefBool(false, "localcache", "enabled")
	localcache_size := x.Conf.GetDefInt(100*1024*1024, "localcache", "size") //100M
	if localcache_enabled {
		x.LocalCache = cache.NewLocalCache(localcache_size)
		if x.Logger != nil {
			x.LocalCache.WithLogger(x.Logger)
		}

		x.Info("LocalCache Init, size:", localcache_size)
	}
} // }}}

// 添加http 方法对应的controller实例, 支持分组; 默认url路径: controller/action, 分组时路径: group/controller/action
func AddApi(c interface{}, group ...string) {
	x.AddApi(c, group...)
}

// 添加rpc 方法对应的controller实例
func AddRpc(c interface{}, group ...string) {
	x.AddRpc(c, group...)
}

// 添加cli 方法对应的controller实例
func AddCli(c interface{}) {
	x.AddCli(c)
}

// 添加 grpc ServerOption
func AddGrpcServerOption(o ...grpc.ServerOption) {
	x.AddGrpcServerOption(o...)
}

func (this *Nyx) RunTcp() {
	this.run(SERVER_TCP)
}

func (this *Nyx) RunWs() {
	this.run(SERVER_WS)
}

func (this *Nyx) RunRpc() {
	this.run(SERVER_RPC)
}

func (this *Nyx) RunHttp() {
	this.run(SERVER_HTTP)
}

func (this *Nyx) RunCli() {
	this.run(SERVER_CLI)
}

// 支持命令行参数 -m 指定运行模式
func (this *Nyx) Run() {
	this.run(this.Mode...)
}

func (this *Nyx) run(modes ...string) { // {{{
	defer func() {
		x.Redis.Close()
		x.DB.Close()
		if x.LocalCache != nil {
			x.LocalCache.Clear()
		}
		if x.Logger != nil {
			x.Logger.Close()
		}

		this.removePidFile()

		x.Warn("======= Server Exit: " + x.DateTime() + " " + x.TIME_ZONE + " ======")
	}()

	if len(modes) == 0 {
		x.Warn("Error: ", "未指定运行模式")
		os.Exit(0)
	}

	var monitor_port string
	var wg sync.WaitGroup
	for _, mode := range modes { // {{{

		switch mode {
		case "http":
			this.useHttpMiddlewares()

			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewHttpServer(
					x.Conf.GetString("http_sever", "addr"),
					x.Conf.GetDefInt(80, "http_server", "port"),
					x.Conf.GetDefInt(60000, "http_server", "read_timeout"),
					x.Conf.GetDefInt(60000, "http_server", "write_timeout"),
					x.Conf.GetDefBool(true, "http_server", "use_graceful"),
					x.Conf.GetBool("http_server", "pprof_enable"),
					x.Conf.GetBool("http_server", "static_files", "enabled"),
					x.Conf.GetDefString("static", "http_server", "static_files", "path"),
					x.Conf.GetDefString("../www", "http_server", "static_files", "root"),
				).Run()
			}()

		case "rpc":
			this.useRpcMiddlewares()
			monitor_port = x.Conf.GetString("rpc_server", "monitor_port")

			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("rpc_server", "port")
				if p <= 0 {
					panic("请先指定 rpc  服务端口号!")
				}
				x.NewRpcServer(x.Conf.GetString("rpc_server", "addr"), p, x.Conf.GetInt("rpc_server", "timeout"), x.Conf.GetDefBool(true, "rpc_server", "use_graceful")).Run()
			}()

		case "tcp":
			monitor_port = x.Conf.GetString("tcp_server", "monitor_port")
			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("tcp_server", "port")
				if p <= 0 {
					panic("请先指定 tcp  服务端口号!")
				}
				x.NewTcpServer(x.Conf.GetString("tcp_server", "addr"), p, x.Conf.GetDefBool(true, "tcp_server", "use_graceful")).Run()
			}()

		case "ws":
			monitor_port = x.Conf.GetString("ws_server", "monitor_port")
			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("ws_server", "port")
				if p <= 0 {
					panic("请先指定 ws  服务端口号!")
				}
				x.NewWsServer(x.Conf.GetString("ws_server", "addr"), p, x.Conf.GetInt("ws_server", "read_timeout"), x.Conf.GetInt("ws_server", "write_timeout"), x.Conf.GetDefBool(true, "ws_server", "use_graceful")).Run()
			}()

		case "cli":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewCliServer(this.cliPath, this.cliParams).Run()
			}()

		default:
			x.Warn("Error: ", "未指定正确的运行模式")
			os.Exit(0)
		}

		x.Success("======= " + mode + " Server Start: " + x.DateTime() + " " + x.TIME_ZONE + " ======")

	} // }}}

	if monitor_port != "" {
		go x.RunMonitor(monitor_port)
	}

	wg.Wait()

} // }}}

// 生成pid文件
func (this *Nyx) genPidFile() { // {{{
	pid := os.Getpid()
	ioutil.WriteFile(x.Conf.GetDefString("./pid", "app_pid_file"), x.AsBytes(pid), 0777)

	x.Info("Pid: ", pid)
} // }}}

// 删除pid文件
func (this *Nyx) removePidFile() {
	os.Remove(x.Conf.GetDefString("./pid", "app_pid_file"))
}
