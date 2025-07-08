package nyx

import (
	"flag"
	"fmt"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyx/x/endless"
	"github.com/nyxless/nyx/x/log"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SYSTEM_VERSION = "1.0"

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
	logo := `
       _/      _/  _/      _/  _/      _/   
      _/_/    _/    _/  _/      _/  _/
     _/  _/  _/      _/          _/
    _/    _/_/      _/        _/  _/
   _/      _/      _/      _/      _/    ` + "Version: " + SYSTEM_VERSION + ` 
-------------------------------------------------------
`
	x.Printf("\033[1;31;33m%s\033[0m\n", logo)

	this.envInit()
	this.genPidFile()
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
		x.Println("Error: ", err)
		os.Exit(0)
	}

	if path.Base(app_root) == "bin" {
		app_root = path.Dir(app_root)
	}

	x.AppRoot = app_root //服务根路径

	config, err := x.NewConfig(config_file, x.AppRoot+"/conf/app.conf", "../conf/app.conf", "./conf/app.conf", "app.conf") //按顺序寻找配置文件
	if nil != err {
		x.Println("Error: ", err)
		os.Exit(0)
	}

	x.Println("Config Init: ", config.ConfigFile)

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

	// 缓存配置文件变量(框架中用到的)
	this.cacheConf()

	err = this.useLogger()
	if nil != err {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	x.Println("run cmd: ", os.Args)
	x.Println("time: ", time.Now().Format("2006-01-02 15:04:05"))
} // }}}

// 缓存配置文件变量
func (this *Nyx) cacheConf() { // {{{
	x.Conf_default_controller = strings.ToLower(x.Conf.GetDefString("index", "default_controller"))
	x.Conf_default_action = strings.ToLower(x.Conf.GetDefString("index", "default_action"))

	x.Conf_env_mode = x.Conf.GetString("env_mode")
	x.Conf_access_log_enabled = x.Conf.GetDefBool(true, "access_log", "enabled")
	x.Conf_access_log_success_level_name = x.Conf.GetString("access_log", "success_level_name")
	x.Conf_access_log_error_level_name = x.Conf.GetString("access_log", "error_level_name")
	x.Conf_access_log_omit_params = x.Conf.GetStringSlice("access_log", "omit_params")
	x.Conf_template_enabled = x.Conf.GetBool("template", "enabled")
	x.Conf_max_post_size = int64(x.Conf.GetDefInt(32, "max_post_size") << 20)

	x.Conf_auth_api_check_enabled = x.Conf.GetDefBool(false, "auth", "api_check", "enabled")
	x.Conf_auth_rpc_check_enabled = x.Conf.GetDefBool(true, "auth", "rpc_check", "enabled")
	x.Conf_auth_api_check_ttl = x.Conf.GetDefInt(3600, "auth", "api_check", "ttl")
	x.Conf_auth_rpc_check_ttl = x.Conf.GetDefInt(3600, "auth", "rpc_check", "ttl")
	x.Conf_auth_api_check_method = x.Conf.GetStringSlice("auth", "api_check", "method")
	x.Conf_auth_api_check_except = x.Conf.GetStringSlice("auth", "api_check", "except")
	x.Conf_auth_rpc_check_method = x.Conf.GetStringSlice("auth", "rpc_check", "method")
	x.Conf_auth_rpc_check_except = x.Conf.GetStringSlice("auth", "rpc_check", "except")
	x.Conf_auth_app = x.Conf.GetMapsSlice("auth", "app")

	x.ConfAuthApiCheckMethod = map[string]struct{}{}
	x.ConfAuthApiCheckExcept = map[string]struct{}{}
	x.ConfAuthRpcCheckMethod = map[string]struct{}{}
	x.ConfAuthRpcCheckExcept = map[string]struct{}{}
	x.ConfAuthApp = map[string]string{}

	for _, v := range x.Conf_auth_api_check_method {
		x.ConfAuthApiCheckMethod[strings.ToLower(v)] = struct{}{}
	}

	for _, v := range x.Conf_auth_api_check_except {
		x.ConfAuthApiCheckExcept[strings.ToLower(v)] = struct{}{}
	}

	for _, v := range x.Conf_auth_rpc_check_method {
		x.ConfAuthRpcCheckMethod[strings.ToLower(v)] = struct{}{}
	}

	for _, v := range x.Conf_auth_rpc_check_except {
		x.ConfAuthRpcCheckExcept[strings.ToLower(v)] = struct{}{}
	}

	for _, v := range x.Conf_auth_app {
		x.ConfAuthApp[v["appid"]] = v["secret"]
	}

	this.parseRouter()
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
					url_prefix = append(url_prefix, map[string]string{"from": from, "to": to})
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

func (this *Nyx) useLogger() error { // {{{
	log_enabled := x.Conf.GetDefBool(true, "log_enabled")
	if !log_enabled {
		return nil
	}

	log_queue_size := x.Conf.GetInt("log_queue_size")
	x.Logger = log.NewLogger(log_queue_size)

	log_level := x.Conf.GetInt("log_level")
	if x.Debug {
		x.Logger.SetDebug(true)
		log_level = 0xFF
	}
	x.Logger.SetLevel(log.LogLevel(log_level))
	log_print_filename := x.Conf.GetDefBool(false, "log_print_filename")
	x.Logger.PrintFileName(log_print_filename)

	log_bulk_size := x.Conf.GetInt("log_bulk_size")
	if log_bulk_size > 0 {
		log.DefaultBulkSize = log_bulk_size
	}

	log_file := x.Conf.GetBool("log_file")
	if !log_file {
		return nil
	}

	log_rule := x.Conf.GetMap("log_rule")
	log_level_rule := x.Conf.GetMap("log_level_rule")

	writers := map[string]io.Writer{}
	writer_levels := map[io.Writer][]string{}
	conf := map[string]string{}

	def_path := x.AsString(log_rule["path"])
	def_naming_format := x.AsString(log_rule["naming_format"])
	def_buffer_size := x.AsInt(log_rule["buffer_size"])
	def_file_size := x.AsInt(log_rule["file_size"])
	def_compress := x.AsBool(log_rule["compress"])
	def_compress_before := x.AsInt(log_rule["compress_before"])
	def_remove := x.AsBool(log_rule["remove"])
	def_remove_before := x.AsInt(log_rule["remove_before"])

	if !filepath.IsAbs(def_path) {
		def_path = filepath.Join(x.AppRoot, def_path)
	}

	def_key := def_path + def_naming_format

	var err error
	for level_name, rule := range log_level_rule { // {{{
		logpath := x.AsString(x.GetNode(rule, "path"), def_path)
		naming_format := x.AsString(x.GetNode(rule, "naming_format"), def_naming_format)
		buffer_size := x.AsInt(x.GetNode(rule, "buffer_size"), def_buffer_size)
		file_size := x.AsInt(x.GetNode(rule, "file_size"), def_file_size)
		compress := x.AsBool(x.GetNode(rule, "compress"), def_compress)
		compress_before := x.AsInt(x.GetNode(rule, "compress_before"), def_compress_before)
		remove := x.AsBool(x.GetNode(rule, "remove"), def_remove)
		remove_before := x.AsInt(x.GetNode(rule, "remove_before"), def_remove_before)

		if !filepath.IsAbs(logpath) {
			logpath = filepath.Join(x.AppRoot, logpath)
		}

		key := logpath + naming_format

		conf[level_name] = key
		writer, ok := writers[key]
		if !ok {
			writer, err = log.NewFileWriter(logpath, naming_format,
				log.WithBufferSize(buffer_size),
				log.WithFileSize(file_size),
				log.WithCompress(compress),
				log.WithCompressBefore(compress_before),
				log.WithRemove(remove),
				log.WithRemoveBefore(remove_before),
			)
			if err != nil {
				return err
			}
			writers[key] = writer
		}

		if writer_levels[writer] == nil {
			writer_levels[writer] = []string{}
		}

		writer_levels[writer] = append(writer_levels[writer], level_name)
	} // }}}

	for _, level_name := range []string{"FATAL", "ERROR", "WARN", "NOTICE", "INFO", "DEBUG"} {
		if _, ok := conf[level_name]; !ok {
			conf[level_name] = def_key
			writer, ok := writers[def_key]
			if !ok {
				writer, err = log.NewFileWriter(def_path, def_naming_format,
					log.WithBufferSize(def_buffer_size),
					log.WithFileSize(def_file_size),
					log.WithCompress(def_compress),
					log.WithCompressBefore(def_compress_before),
					log.WithRemove(def_remove),
					log.WithRemoveBefore(def_remove_before),
				)
				if err != nil {
					return err
				}

				writers[def_key] = writer
			}

			if writer_levels[writer] == nil {
				writer_levels[writer] = []string{}
			}

			writer_levels[writer] = append(writer_levels[writer], level_name)
		}
	}

	x.Logger.RemoveWriter(log.DefaultWriter)
	for writer, level_names := range writer_levels {
		x.Logger.AddWriter(writer, level_names...)
	}

	return nil
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

func (this *Nyx) RunWebsocket() {
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
		x.Logger.Close()

		this.removePidFile()
		x.Println("======= Server Exit at: " + x.DateTime() + " " + x.TIME_ZONE + " ======")
	}()

	if len(modes) == 0 {
		fmt.Println("Error: 未指定运行模式")
		os.Exit(0)
	}

	use_graceful := x.Conf.GetDefBool(true, "use_graceful")
	if use_graceful {
		if len(modes) > 1 {
			x.Println("多模式运行时，不支持 endless, 已自动关闭!")
		} else if x.Conf.GetString("env_mode") == "DEV" {
			endless.DevMode = true
		}
	}

	//是否监听status
	run_moniter := x.Conf.GetDefBool(false, "monitor_enable")

	var wg sync.WaitGroup
	for _, mode := range modes { // {{{

		switch mode {
		case "http":
			run_moniter = false

			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewHttpServer(
					x.Conf.GetString("http_addr"),
					x.Conf.GetDefInt(80, "http_port"),
					x.Conf.GetDefInt(60000, "http_read_timeout"),
					x.Conf.GetDefInt(60000, "http_write_timeout"),
					use_graceful,
					x.Conf.GetBool("pprof_enable"),
					x.Conf.GetBool("static_enable"),
					x.Conf.GetDefString("static", "static_path"),
					x.Conf.GetDefString("../www", "static_root"),
				).Run()
			}()

		case "rpc":
			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("rpc_port")
				if p <= 0 {
					panic("请先指定 rpc  服务端口号!")
				}
				x.NewRpcServer(x.Conf.GetString("rpc_addr"), p, x.Conf.GetInt("rpc_timeout"), x.Conf.GetDefBool(true, "use_graceful")).Run()
			}()

		case "tcp":
			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("tcp_port")
				if p <= 0 {
					panic("请先指定 tcp  服务端口号!")
				}
				x.NewTcpServer(x.Conf.GetString("tcp_addr"), p, x.Conf.GetDefBool(true, "use_graceful")).Run()
			}()

		case "ws":
			wg.Add(1)
			go func() {
				defer wg.Done()

				p := x.Conf.GetInt("ws_port")
				if p <= 0 {
					panic("请先指定 ws  服务端口号!")
				}
				x.NewWebsocketServer(x.Conf.GetString("ws_addr"), p, x.Conf.GetInt("ws_timeout"), x.Conf.GetDefBool(true, "use_graceful")).Run()
			}()

		case "cli":
			wg.Add(1)
			go func() {
				defer wg.Done()
				x.NewCliServer(this.cliPath, this.cliParams).Run()
			}()

		default:
			fmt.Println("Error: 未指定正确的运行模式")
			os.Exit(0)
		}

		x.Println("======= " + mode + " Server Start at: " + x.DateTime() + " " + x.TIME_ZONE + " ======")

	} // }}}

	monitor_port := x.Conf.GetString("monitor_port")
	if "" != monitor_port && run_moniter {
		go x.RunMonitor(monitor_port)
	}

	wg.Wait()

} // }}}

// 生成pid文件
func (this *Nyx) genPidFile() { // {{{
	pid := os.Getpid()
	pidString := strconv.Itoa(pid)
	ioutil.WriteFile(x.Conf.GetString("app_pid_file"), []byte(pidString), 0777)

	x.Println("pid: ", pidString)
} // }}}

// 删除pid文件
func (this *Nyx) removePidFile() {
	os.Remove(x.Conf.GetString("app_pid_file"))
}
