package x

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/x/endless"
	"github.com/nyxless/nyx/x/pb"
	"google.golang.org/grpc"
	"net"
	"reflect"
	"time"
	//"runtime"
	"runtime/debug"
	"strings"
)

type Stream = pb.NYXRpc_CallStreamServer

// 执行路由方法的函数
type RpcHandler func(context.Context, map[string]any, Stream) (context.Context, *ResponseData, error)

var (
	defaultRpcs              = [][]any{}
	defaultRouteRpcFuncs     = map[string]map[string]RpcHandler{}
	defaultGrpcServerOptions = []grpc.ServerOption{}
	rpcGroups                = map[string]struct{}{}
)

// 使用反射时，处理 stream 参数为空时，反射空值
var streamType reflect.Type

// 添加rpc 方法对应的controller实例, 支持分组
func AddRpc(c any, groups ...string) { // {{{
	group := ""
	if len(groups) > 0 {
		group = groups[0]
	}

	defaultRpcs = append(defaultRpcs, []any{c, group})
} // }}}

// 添加 route 调用方法, 在脚手架生成代码时使用
func AddRouteRpcFunc(controller_name, action_name string, f RpcHandler) { // {{{
	if defaultRouteRpcFuncs[controller_name] == nil {
		defaultRouteRpcFuncs[controller_name] = map[string]RpcHandler{}
	}

	defaultRouteRpcFuncs[controller_name][action_name] = f
} // }}}

// 添加 grpc ServerOption
func AddGrpcServerOption(o ...grpc.ServerOption) { // {{{
	defaultGrpcServerOptions = append(defaultGrpcServerOptions, o...)
} // }}}

func NewRpcServer(addr string, port, timeout int, useGraceful bool) *RpcServer { // {{{
	if timeout <= 0 {
		timeout = 3000
	}

	AddGrpcServerOption(grpc.ConnectionTimeout(time.Duration(timeout) * time.Millisecond))

	server := &RpcServer{
		addr:        addr,
		port:        port,
		useGraceful: useGraceful,
		handler: &grpcHandler{
			routeMap:      make(map[string]map[string]reflect.Type),
			methodMap:     make(map[string]map[string]int),
			routeFuncs:    defaultRouteRpcFuncs,
			groupHandlers: make(map[string]RpcHandler),
		},
	}

	server.handler.addControllers()
	server.handler.buildGroupHandlers()

	streamType = reflect.TypeOf((*Stream)(nil)).Elem()

	return server
} // }}}

type RpcServer struct {
	addr        string
	port        int
	useGraceful bool
	handler     *grpcHandler
}

func (rs *RpcServer) Run() { // {{{
	if len(rs.handler.routeMap) == 0 {
		Warn("rpc controller not found, pls add controller using func `AddRpc` or shell `nyx init`")

		return
	}

	//runtime.GOMAXPROCS(runtime.NumCPU())
	rpcServer := grpc.NewServer(defaultGrpcServerOptions...)
	pb.RegisterNYXRpcServer(rpcServer, rs.handler)

	addr := fmt.Sprintf("%s:%d", rs.addr, rs.port)
	Info("RpcServer Listen: ", addr)

	if rs.useGraceful {
		Info("Use graceful: ", "open")
		Warn(endless.ListenAndServeTcp(addr, "", rpcServer))
	} else {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", rs.port))

		if err != nil {
			Warn(err)
			return
		}

		rpcServer.Serve(lis)
	}
} // }}}

type grpcHandler struct { // {{{
	pb.UnimplementedNYXRpcServer
	routeMap      map[string]map[string]reflect.Type //key:controller: {key:method value:reflect.type}
	methodMap     map[string]map[string]int          //key:controller: {key:method value:method_index}
	routeFuncs    map[string]map[string]RpcHandler   //controller.action 函数缓存, 替换反射调用
	groupHandlers map[string]RpcHandler              //group buildRpcMiddlewares 后的函数缓存
} // }}}

func (g *grpcHandler) Call(ctx context.Context, req *pb.Request) (*pb.Reply, error) { // {{{
	_, res, err := g.Serve(ctx, req, nil)

	return BuildReply(res), err
} // }}}

func (g *grpcHandler) CallStream(req *pb.Request, stream Stream) error { // {{{
	ctx := stream.Context()
	_, _, err := g.Serve(ctx, req, stream)

	return err
} // }}}

func (g *grpcHandler) Serve(ctx context.Context, req *pb.Request, stream Stream) (newctx context.Context, res *ResponseData, err error) { // {{{
	newctx = ctx
	requesturi := req.Method

	params := map[string]any{}
	for _, v := range req.Data {
		params[v.Name] = BytesToData(v.Type, v.Value)
	}

	defer func() {
		if recover_err := recover(); recover_err != nil {
			var errmsg string
			switch errinfo := recover_err.(type) {
			case *Error:
				errmsg = errinfo.GetMessage()
			case error:
				errmsg = errinfo.Error()
				Warn(errmsg)
				debug.PrintStack()
			default:
				errmsg = fmt.Sprint(errinfo)
			}

			res = &ResponseData{
				Code: int32(ERR_SYSTEM.GetCode()),
				Msg:  errmsg,
			}

			Notice("ServeRpc: ", errmsg)
		}
	}()

	uri := strings.Trim(requesturi, " \r\t\v/")
	idx := strings.LastIndex(uri, "/")

	Interceptor(idx > 0, ERR_METHOD_INVALID, uri)

	uri = strings.ToLower(uri)

	group := ""
	controller_name := uri[:idx]
	action_name := uri[idx+1:]

	if idx = strings.LastIndex(controller_name, "/"); idx > 0 {
		group = controller_name[:idx]
	}

	ctx = context.WithValue(ctx, "group", group)
	ctx = context.WithValue(ctx, "controller", controller_name)
	ctx = context.WithValue(ctx, "action", action_name)

	//路由解析之后加载中间件
	if hd, ok := g.groupHandlers[group]; ok {
		return hd(ctx, params, stream)
	} else {
		return g.defaultHandler(ctx, params, stream)
	}

	return
} // }}}

func (g *grpcHandler) defaultHandler(ctx context.Context, params map[string]any, stream Stream) (newctx context.Context, res *ResponseData, err error) { // {{{
	newctx = ctx
	group := ctx.Value("group").(string)
	controller_name := ctx.Value("controller").(string)
	action_name := ctx.Value("action").(string)

	// 先尝试执行预生成函数代码
	if cf, ok := g.routeFuncs[controller_name]; ok {
		if f, ok := cf[action_name]; ok {
			return f(ctx, params, stream)
		}
	}

	canhandler := false
	var controllerType reflect.Type
	if controller_name != "" && action_name != "" {
		if routMapSub, ok := g.routeMap[controller_name]; ok {
			if controllerType, ok = routMapSub[action_name]; ok {
				canhandler = true
			}
		}
	}

	Interceptor(canhandler, ERR_METHOD_INVALID, controller_name+"/"+action_name)

	//未预生成代码，使用反射
	Info("Pre-generated code for " + controller_name + "/" + action_name + " was not found.  Reflection is used now OR U can gen code using shell `nyx init`")

	vc := reflect.New(controllerType)
	var in []reflect.Value
	var method reflect.Value

	defer func() {
		if recover_err := recover(); recover_err != nil {
			in = []reflect.Value{reflect.ValueOf(recover_err)}
			method := vc.Method(g.methodMap[controller_name]["RenderError"])
			method.Call(in)

			in = make([]reflect.Value, 0)
			method = vc.Method(g.methodMap[controller_name]["GetResponseData"])
			ret := method.Call(in)
			res, _ = ret[0].Interface().(*ResponseData) //res = ret[0].Bytes()
			err, _ = ret[1].Interface().(error)
		}

		in = make([]reflect.Value, 0)
		method = vc.Method(g.methodMap[controller_name]["Final"])
		method.Call(in)
	}()

	in = make([]reflect.Value, 6)
	in[0] = reflect.ValueOf(ctx)
	in[1] = reflect.ValueOf(params)
	in[2] = reflect.ValueOf(controller_name)
	in[3] = reflect.ValueOf(action_name)
	in[4] = reflect.ValueOf(group)
	if stream == nil {
		in[5] = reflect.Zero(streamType)
	} else {
		in[5] = reflect.ValueOf(stream)
	}

	method = vc.Method(g.methodMap[controller_name]["Prepare"])
	method.Call(in)

	//call Init method if exists
	in = make([]reflect.Value, 0)
	method = vc.Method(g.methodMap[controller_name]["Init"])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(g.methodMap[controller_name][action_name])
	method.Call(in)

	in = make([]reflect.Value, 0)
	method = vc.Method(g.methodMap[controller_name]["GetResponseData"])
	ret := method.Call(in)
	res, _ = ret[0].Interface().(*ResponseData)
	err, _ = ret[1].Interface().(error)

	return
} // }}}

// 预生成 middleware 函数缓存
func (g *grpcHandler) buildGroupHandlers() { // {{{
	g.groupHandlers = map[string]RpcHandler{}
	allGroups := rpcGroups
	allGroups[""] = struct{}{}

	for group := range allGroups {
		if len(rpcMiddlewares) > 0 {
			groupHandler := buildRpcMiddlewares(group, rpcMiddlewares...)(RpcHandler(g.defaultHandler))
			g.groupHandlers[group] = groupHandler
		} else {
			g.groupHandlers[group] = RpcHandler(g.defaultHandler)
		}
	}
} // }}}

func (g *grpcHandler) addControllers() {
	for _, v := range defaultRpcs {
		g.addController(v[0], AsString(v[1]))
	}
}

func (g *grpcHandler) addController(c any, group ...string) { // {{{
	reflectVal := reflect.ValueOf(c)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()
	controller_name := strings.TrimSuffix(ct.Name(), CONTROLLER_SUFFIX)
	if len(group) > 0 && group[0] != "" {
		rpcGroups[group[0]] = struct{}{}
		controller_name = strings.Trim(group[0], " \r\t\v/") + "/" + controller_name
	}

	controller_name = strings.ToLower(controller_name)

	if _, ok := g.routeMap[controller_name]; ok {
		return
	} else {
		g.routeMap[controller_name] = make(map[string]reflect.Type)
		g.methodMap[controller_name] = make(map[string]int)
	}
	var action_fullname string
	var action_name string
	for i := 0; i < rt.NumMethod(); i++ {
		action_fullname = rt.Method(i).Name
		if strings.HasSuffix(action_fullname, ACTION_SUFFIX) {
			action_name = strings.ToLower(strings.TrimSuffix(action_fullname, ACTION_SUFFIX))
			g.routeMap[controller_name][action_name] = ct
			g.methodMap[controller_name][action_name] = rt.Method(i).Index
		}
	}
	methodRenderError, _ := rt.MethodByName("RenderError")
	methodPrepare, _ := rt.MethodByName("Prepare")
	methodInit, _ := rt.MethodByName("Init")
	methodFinal, _ := rt.MethodByName("Final")
	methodGetResponseData, _ := rt.MethodByName("GetResponseData")
	g.methodMap[controller_name]["RenderError"] = methodRenderError.Index
	g.methodMap[controller_name]["Prepare"] = methodPrepare.Index
	g.methodMap[controller_name]["Init"] = methodInit.Index
	g.methodMap[controller_name]["Final"] = methodFinal.Index
	g.methodMap[controller_name]["GetResponseData"] = methodGetResponseData.Index

} // }}}

func BuildReply(res *ResponseData) *pb.Reply { // {{{
	data := AsMap(res.Data)
	var reply_data []*pb.Field
	for k, v := range data {
		field := &pb.Field{}
		field.Name = k
		field.Type, field.Value = DataToBytes(v)

		reply_data = append(reply_data, field)
	}

	return &pb.Reply{
		Code:    res.Code,
		Consume: res.Consume,
		Time:    res.Time,
		Msg:     res.Msg,
		Data:    reply_data,
	}
} // }}}
