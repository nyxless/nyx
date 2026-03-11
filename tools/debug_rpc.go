package tools

import (
	"context"
	"fmt"
	"github.com/nyxless/nyx/controller"
	"github.com/nyxless/nyx/x"
	"github.com/nyxless/nyxc"
	"net/http"
	"strings"
	"time"
)

func getRpcClient() (*nyxc.NyxClient, error) { // {{{
	host := x.Conf.GetString("rpc_server", "addr")
	port := x.Conf.GetString("rpc_server", "port")
	return nyxc.NewNyxClient(host+":"+port, x.ConfDebugRpcAppid, x.ConfDebugRpcSecret)
} //}}}

func rpcRequest(ctx context.Context, con, act string, params x.MAP, hds x.MAPS) (x.MAP, error) { // {{{
	c, err := getRpcClient()
	if nil != err {
		return nil, err
	}

	res, err := c.Request(con+"/"+act, params, nyxc.WithContext(ctx), nyxc.WithHeaders(hds), nyxc.WithTimeout(time.Duration(x.ConfDebugRpcTimeout)*time.Second))
	if err != nil {
		return nil, err
	}

	if res.GetCode() > 0 {
		return nil, fmt.Errorf("rpc client return err[code: %d, msg: %s]", res.GetCode(), res.GetMsg())
	}

	return res.GetData(), nil
} // }}}

func DebugRpc(rw http.ResponseWriter, r *http.Request) { // {{{

	path := r.URL.Path
	uri := strings.Trim(strings.TrimPrefix(path, "/debug/rpc/"), " \r\t\v/")

	idx := strings.LastIndex(uri, "/")
	x.Interceptor(idx > 0, x.ERR_METHOD_INVALID, uri)

	uri = strings.ToLower(uri)

	group := ""
	controller_name := uri[:idx]
	action_name := uri[idx+1:]

	if idx = strings.LastIndex(controller_name, "/"); idx > 0 {
		group = controller_name[:idx]
	}

	c := &controller.HTTP{}
	c.Prepare(rw, r, controller_name, action_name, group)

	params := c.GetParams()
	hds := c.GetHeaders()
	rpc_hds := x.MAPS{}
	for k, v := range hds {
		if strings.HasPrefix(k, "rpc-") {
			rpc_hds[k] = v
			rpc_hds[strings.TrimPrefix(k, "rpc-")] = v
		}
	}
	rpc_res, err := rpcRequest(r.Context(), controller_name, action_name, params, rpc_hds)
	if err != nil {
		c.RenderError(x.NewErr(x.ERR_OTHER.GetCode(), err.Error()))
	} else {
		c.Render(rpc_res)
	}
	c.Final()
} // }}}
