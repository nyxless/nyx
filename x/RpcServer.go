package x

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/endless"
	"github.com/nyxless/nyx/x/pb"
	"google.golang.org/grpc"
	"log"
	"net"
	"reflect"
	"time"
	//"runtime"
	"runtime/debug"
	"strings"
)

// 执行路由方法的函数
type RouteRpcFunc func(map[string]any, context.Context) map[string]any

var (
	defaultRpcs              = [][]any{}
	defaultRouteRpcFuncs     = map[string]map[string]RouteRpcFunc{}
	defaultGrpcServerOptions = []grpc.ServerOption{}
)

// 添加rpc 方法对应的controller实例, 支持分组
func AddRpc(c any, groups ...string) { // {{{
	group := ""
	if len(groups) > 0 {
		group = groups[0]
	}

	defaultRpcs = append(defaultRpcs, []any{c, group})
} // }}}

// 添加 route 调用方法, 在脚手架生成代码时使用
func AddRouteRpcFunc(controller_name, action_name string, f RouteRpcFunc) { // {{{
	if defaultRouteRpcFuncs[controller_name] == nil {
		defaultRouteRpcFuncs[controller_name] = map[string]RouteRpcFunc{}
	}

	defaultRouteRpcFuncs[controller_name][action_name] = f
} // }}}

// 添加 grpc ServerOption
func AddGrpcServerOption(o ...grpc.ServerOption) { // {{{
	defaultGrpcServerOptions = append(defaultGrpcServerOptions, o...)
} // }}}

func NewRpcServer(addr string, port, timeout int, useGraceful bool) *RpcServer { // {{{
	if timeout <= 0 {
		timeout = 3
	}

	AddGrpcServerOption(grpc.ConnectionTimeout(time.Duration(timeout) * time.Second))

	server := &RpcServer{
		addr:        addr,
		port:        port,
		useGraceful: useGraceful,
		handler: &rpcHandler{
			routMap:    make(map[string]map[string]reflect.Type),
			methodMap:  make(map[string]map[string]int),
			routeFuncs: defaultRouteRpcFuncs,
		},
	}

	for _, v := range defaultRpcs {
		server.AddController(v[0], AsString(v[1]))
	}

	return server
} // }}}

type RpcServer struct {
	addr        string
	port        int
	useGraceful bool
	handler     *rpcHandler
}

func (this *RpcServer) AddController(c any, group ...string) {
	this.handler.addController(c, group...)
}

func (this *RpcServer) Run() { // {{{
	if len(this.handler.routMap) == 0 || len(this.handler.routeFuncs) == 0 {
		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	rpcServer := grpc.NewServer(defaultGrpcServerOptions...)
	pb.RegisterNYXRpcServer(rpcServer, this.handler)

	addr := fmt.Sprintf("%s:%d", this.addr, this.port)
	Println("RpcServer Listen", addr)

	if this.useGraceful {
		log.Println(endless.ListenAndServeTcp(addr, "", rpcServer))
	} else {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", this.port))

		if err != nil {
			panic(err)
		}
		rpcServer.Serve(lis)
	}
} // }}}

type rpcHandler struct { // {{{
	pb.UnimplementedNYXRpcServer
	routMap    map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	methodMap  map[string]map[string]int          //key:controller: {key:method value:method_index}
	routeFuncs map[string]map[string]RouteRpcFunc //controller.action 函数缓存, 替换反射调用
} // }}}

func (this *rpcHandler) Call(ctx context.Context, in *pb.Request) (*pb.Reply, error) { // {{{
	method := in.Method

	params := map[string]interface{}{}
	for k, v := range in.Keys {
		if in.Types[k] == "BYTES" {
			params[v] = in.Values[k]
		} else {
			params[v] = string(in.Values[k])
		}
	}

	res := this.Serve(method, params, ctx)

	return this.buildReply(res), nil
} // }}}

func (this *rpcHandler) Serve(requesturi string, params map[string]any, ctx context.Context) (res map[string]any) { // {{{
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

			res = map[string]interface{}{
				"code": ERR_SYSTEM.GetCode(),
				"msg":  errmsg,
			}

			log.Println("ServeRpc: ", errmsg)
		}
	}()

	uri := strings.Trim(requesturi, " \r\t\v/")
	idx := strings.LastIndex(uri, "/")

	Interceptor(idx > 0, ERR_METHOD_INVALID, uri)

	uri = strings.ToLower(uri)

	controller_name := uri[:idx]
	action_name := uri[idx+1:]

	// 先尝试执行预生成函数代码
	if cf, ok := this.routeFuncs[controller_name]; ok {
		if f, ok := cf[action_name]; ok {
			return f(params, ctx)
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

	Interceptor(canhandler, ERR_METHOD_INVALID, uri)

	vc := reflect.New(controllerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if err := recover(); err != nil {
			in = []reflect.Value{reflect.ValueOf(err)}
			method := vc.Method(this.methodMap[controller_name]["RenderError"])
			method.Call(in)

			in = make([]reflect.Value, 0)
			method = vc.Method(this.methodMap[controller_name]["GetRpcContent"])
			ret := method.Call(in)
			res = ret[0].Interface().(map[string]any) //res = ret[0].Bytes()
		}
	}()

	in = make([]reflect.Value, 4)
	in[0] = reflect.ValueOf(params)
	in[1] = reflect.ValueOf(ctx)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	method = vc.Method(this.methodMap[controller_name]["PrepareRpc"])
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name]["Init"])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name][action_name])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(this.methodMap[controller_name]["GetRpcContent"])
	ret := method.Call(in)
	res = ret[0].Interface().(map[string]any)

	return
} // }}}

func (this *rpcHandler) addController(c any, group ...string) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), CONTROLLER_SUFFIX)
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
	methodPrepare, _ := rt.MethodByName("PrepareRpc")
	methodInit, _ := rt.MethodByName("Init")
	methodGetRpcContent, _ := rt.MethodByName("GetRpcContent")
	this.methodMap[controller_name]["RenderError"] = methodRenderError.Index
	this.methodMap[controller_name]["PrepareRpc"] = methodPrepare.Index
	this.methodMap[controller_name]["Init"] = methodInit.Index
	this.methodMap[controller_name]["GetRpcContent"] = methodGetRpcContent.Index

} // }}}

func (this *rpcHandler) buildReply(res map[string]interface{}) *pb.Reply { // {{{
	var reply_data *pb.ReplyData
	keys := map[int32]string{}
	types := map[int32]string{}
	values := map[int32][]byte{}

	if data, ok := res["data"].(map[string]interface{}); ok {
		var i int32
		for k, v := range data {
			keys[i] = k
			if v != nil {
				if val, ok := v.([]byte); ok {
					types[i] = "BYTES"
					values[i] = val
				} else {
					typ := reflect.TypeOf(v).Kind()

					switch typ {
					case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
						types[i] = "JSON"
						values[i] = JsonEncodeToBytes(v)
					default:
						values[i] = AsBytes(v)
					}
				}
			}

			i++
		}
		reply_data = &pb.ReplyData{Keys: keys, Types: types, Values: values}
	}

	return &pb.Reply{
		Code:    AsInt32(res["code"]),
		Consume: AsInt32(res["consume"]),
		Time:    AsInt64(res["time"]),
		Msg:     AsString(res["msg"]),
		Data:    reply_data,
	}
} // }}}
