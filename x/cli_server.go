package x

import (
	"fmt"
	"net/url"
	"reflect"
	"runtime/debug"
	"strings"
)

var (
	defaultClis = [][]any{}
)

// 添加cli 方法对应的controller实例, 支持分组
func AddCli(c any, groups ...string) { // {{{
	group := ""
	if len(groups) > 0 {
		group = groups[0]
	}

	defaultClis = append(defaultClis, []any{c, group})
} // }}}

func NewCliServer(uri, params string) *CliServer {
	server := &CliServer{
		uri:       uri,
		params:    params,
		routMap:   make(map[string]map[string]reflect.Type),
		methodMap: make(map[string]map[string]int),
	}

	for _, v := range defaultClis {
		server.AddController(v[0], AsString(v[1]))
	}

	return server
}

type CliServer struct {
	uri       string
	params    string
	routMap   map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	methodMap map[string]map[string]int          //key:controller: {key:method value:method_index}
}

func (this *CliServer) AddController(c any, group ...string) {
	this.addController(c, group...)
}

func (this *CliServer) Run() {
	if len(this.routMap) == 0 {
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	this.serveCli()
}

func (this *CliServer) serveCli() {
	defer func() {
		if err := recover(); err != nil {
			var errmsg string
			switch errinfo := err.(type) {
			case *Error:
				errmsg = errinfo.GetMessage()
			case error:
				errmsg = errinfo.Error()
				fmt.Println(errmsg)
				debug.PrintStack()
			default:
				errmsg = fmt.Sprint(errinfo)
			}

			fmt.Println("ServeCli: ", errmsg)
		}
	}()

	uri := strings.Trim(this.uri, " \r\t\v/")
	idx := strings.LastIndex(uri, "/")

	Interceptor(idx > 0, ERR_METHOD_INVALID, this.uri)

	uri = strings.ToLower(uri)

	controller_name := uri[:idx]
	action_name := uri[idx+1:]

	var group string
	if idx = strings.LastIndex(controller_name, "/"); idx > 0 {
		group = controller_name[:idx]
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

	Interceptor(canhandler, ERR_METHOD_INVALID, uri)

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

	m := map[string][]string{}
	var err error
	if len(this.params) > 1 {
		m, err = url.ParseQuery(this.params)
		if nil != err {
			fmt.Println("params parse error")
			return
		}
	}

	in = make([]reflect.Value, 4)
	in[0] = reflect.ValueOf(m)
	in[1] = reflect.ValueOf(controller_name)
	in[2] = reflect.ValueOf(action_name)
	in[3] = reflect.ValueOf(group)
	method = vc.Method(this.methodMap[controller_name]["PrepareCli"])
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name]["Init"])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name][action_name])
	method.Call(in)

	return
}

func (this *CliServer) addController(c any, group ...string) {
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), "Controller")
	if len(group) > 0 && group[0] != "" {
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
}
