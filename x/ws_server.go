package x

import (
	"bufio"
	"fmt"
	"github.com/nyxless/nyx/x/endless"
	"golang.org/x/net/websocket"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

var (
	defaultWsMaxHeaderBytes = 0 //0时, 将使用默认配置DefaultMaxHeaderBytes(1M)
	defaultWsPath           = "/"

	defaultWsHandler = func(conn net.Conn) { // {{{

		defer conn.Close()

		for {
			reader := bufio.NewReader(conn)
			var buf [128]byte
			n, err := reader.Read(buf[:])
			if err != nil {
				fmt.Println("read from client failed, err: ", err)
				break
			}
			recvStr := string(buf[:n])
			fmt.Println("Received msg from Client：", conn.RemoteAddr(), recvStr)
			conn.Write([]byte(recvStr))
		}
	} // }}}
)

// 设置MaxHeaderBytes, 单位M
func SetWsMaxHeaderBytes(m int) { // {{{
	if m > 0 {
		defaultWsMaxHeaderBytes = m << 20
	}
} // }}}

// 设置路由及回调处理
func SetWsHandler(p string, h func(net.Conn)) { // {{{
	if p != "" {
		defaultWsPath = p
	}

	defaultWsHandler = h
} // }}}

func NewWsServer(addr string, port, rtimeout, wtimeout int, useGraceful bool) *WsServer {
	server := &WsServer{
		addr:           addr,
		port:           port,
		readTimeout:    rtimeout,
		writeTimeout:   wtimeout,
		useGraceful:    useGraceful,
		maxHeaderBytes: defaultWsMaxHeaderBytes,
		path:           defaultWsPath,
		handler:        defaultWsHandler,
	}

	return server
}

// WebsocketServer
type WsServer struct {
	addr           string
	port           int
	readTimeout    int //毫秒
	writeTimeout   int //毫秒
	useGraceful    bool
	maxHeaderBytes int
	path           string
	handler        func(net.Conn)
}

func (w *WsServer) Run() { // {{{
	if w.handler == nil {
		return
	}

	addr := fmt.Sprintf("%s:%d", w.addr, w.port)

	log.Println("WebsocketServer Listen", addr)

	mux := http.NewServeMux()
	mux.Handle(w.path, websocket.Handler(func(conn *websocket.Conn) {
		w.process(conn)
	}))

	rtimeout := time.Duration(w.readTimeout) * time.Millisecond
	wtimeout := time.Duration(w.writeTimeout) * time.Millisecond

	if w.useGraceful {
		log.Println(endless.ListenAndServe(addr, mux, rtimeout, wtimeout, w.maxHeaderBytes))
	} else {
		httpServer := &http.Server{
			Addr:           addr,
			Handler:        mux,
			ReadTimeout:    rtimeout,
			WriteTimeout:   wtimeout,
			MaxHeaderBytes: w.maxHeaderBytes,
		}

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Println("websocket Listen error:", err)
		}

		httpServer.Serve(NewTCPKeepAliveListener(ln.(*net.TCPListener), time.Minute*3))

	}
} // }}}

type tcpKeepAliveListener struct {
	*net.TCPListener
	keepAliveDuration time.Duration
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) { // {{{
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(ln.keepAliveDuration)
	return tc, nil
} // }}}

func NewTCPKeepAliveListener(ln *net.TCPListener, d time.Duration) net.Listener { // {{{
	return &tcpKeepAliveListener{
		TCPListener:       ln,
		keepAliveDuration: d,
	}
} // }}}

func (w *WsServer) process(conn net.Conn) { // {{{
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

			fmt.Println("ServeWebsocket: ", errmsg)
		}
	}()

	defer conn.Close() // 关闭连接

	w.handler(conn)
} // }}}
