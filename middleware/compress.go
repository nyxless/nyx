package middleware

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/nyxless/nyx/x"
)

type CompressConfig struct {
	MinSize      int //压缩数据大小阈值
	GzipLevel    int // gzip压缩等级
	DeflateLevel int // deflate压缩等级
}

// 支持 gzip | deflate 压缩，小数据不压缩；数据超过阈值后切换为流式压缩，支持 Flush
func Compress(config *CompressConfig) x.HttpMiddleware {
	gzipLevel := validateGzipLevel(config.GzipLevel)
	deflateLevel := validateDeflateLevel(config.DeflateLevel)
	minSize := config.MinSize
	if minSize <= 0 {
		minSize = 1024 // 默认1KB
	}

	gzipPool := createGzipPool(gzipLevel)
	deflatePool := createDeflatePool(deflateLevel)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acceptEncoding := r.Header.Get("Accept-Encoding")

			if strings.Contains(acceptEncoding, "gzip") {
				handleCompress(w, r, next, minSize, "gzip", gzipPool)
			} else if strings.Contains(acceptEncoding, "deflate") {
				handleCompress(w, r, next, minSize, "deflate", deflatePool)
			} else {
				// 不支持压缩
				w.Header().Add("Vary", "Accept-Encoding")
				next.ServeHTTP(w, r)
			}
		})
	}
}

// 验证gzip压缩等级
func validateGzipLevel(level int) int {
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		return gzip.DefaultCompression
	}
	return level
}

// 验证deflate压缩等级
func validateDeflateLevel(level int) int {
	if level < flate.BestSpeed || level > flate.BestCompression {
		return flate.DefaultCompression
	}
	return level
}

// 创建gzip对象池
func createGzipPool(level int) *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, level)
			return w
		},
	}
}

// 创建deflate对象池
func createDeflatePool(level int) *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			w, _ := flate.NewWriter(nil, level)
			return w
		},
	}
}

// flusherWriter 抽象 gzip.Writer / flate.Writer 共有的 Flush 能力
type flusherWriter interface {
	io.WriteCloser
	Flush() error
}

type bufferedResponseWriter struct {
	http.ResponseWriter
	buffer   *bytes.Buffer
	status   int
	header   http.Header
	encoding string
	pool     *sync.Pool
	minSize  int

	writer    flusherWriter // 切换压缩后的写出口
	headerSet bool          // 是否已写出响应头
}

func newBufferedResponseWriter(w http.ResponseWriter, minSize int, encoding string, pool *sync.Pool) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		ResponseWriter: w,
		buffer:         bytes.NewBuffer(nil),
		header:         make(http.Header),
		minSize:        minSize,
		encoding:       encoding,
		pool:           pool,
	}
}

func (w *bufferedResponseWriter) Write(b []byte) (int, error) {
	// 已进入压缩模式，直接写压缩器
	if w.writer != nil {
		return w.writer.Write(b)
	}

	n, err := w.buffer.Write(b)
	if err != nil {
		return n, err
	}

	// 超过阈值，切换到压缩模式
	if w.buffer.Len() >= w.minSize {
		if err := w.switchToCompress(); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

// switchToCompress 把已缓冲数据交给压缩器，并设置响应头，切换到流式压缩模式
func (w *bufferedResponseWriter) switchToCompress() error {
	// 拷贝外层响应头到真实 writer
	for k, v := range w.header {
		w.ResponseWriter.Header()[k] = v
	}

	w.ResponseWriter.Header().Set("Content-Encoding", w.encoding)

	// 从对象池获取压缩器
	var cw flusherWriter
	if w.encoding == "gzip" {
		gz := w.pool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter)
		cw = gz
	} else {
		fl := w.pool.Get().(*flate.Writer)
		fl.Reset(w.ResponseWriter)
		cw = fl
	}
	w.writer = cw

	if w.status != 0 {
		w.ResponseWriter.WriteHeader(w.status)
	}
	w.headerSet = true

	// 把已缓冲数据写入压缩器
	if w.buffer.Len() > 0 {
		if _, err := cw.Write(w.buffer.Bytes()); err != nil {
			return err
		}
		w.buffer.Reset()
	}

	return nil
}

// finish 结束响应：小数据直接写，大数据关闭压缩器
func (w *bufferedResponseWriter) finish() {
	// 未切换压缩，原样输出
	if w.writer == nil {
		// 把 header 拷回真实 writer
		for k, v := range w.header {
			w.ResponseWriter.Header()[k] = v
		}
		if w.status != 0 {
			w.ResponseWriter.WriteHeader(w.status)
		}
		if w.buffer.Len() > 0 {
			w.ResponseWriter.Write(w.buffer.Bytes())
		}
		return
	}

	// 已压缩模式，收尾
	w.writer.Close()
	if w.encoding == "gzip" {
		w.pool.Put(w.writer.(*gzip.Writer))
	} else {
		w.pool.Put(w.writer.(*flate.Writer))
	}
	w.writer = nil
}

// Flush 支持流式推送。若尚未切换压缩且 buffer 未达阈值，会强制切换压缩后 flush。
func (w *bufferedResponseWriter) Flush() {
	if w.writer == nil {
		// 还没进入压缩模式：强制切换（放弃"小数据不压缩"的判断，因为要实时推送）
		if err := w.switchToCompress(); err != nil {
			return
		}
	}

	w.writer.Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack 透传底层 Hijacker，供 WebSocket / 连接接管使用。
func (w *bufferedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func handleCompress(w http.ResponseWriter, r *http.Request, next http.Handler, minSize int, encoding string, pool *sync.Pool) { // {{{

	w.Header().Add("Vary", "Accept-Encoding")

	buffered := newBufferedResponseWriter(w, minSize, encoding, pool)
	next.ServeHTTP(buffered, r)
	buffered.finish()
} // }}}
