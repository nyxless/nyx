package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"github.com/nyxless/nyx/x"
	"io"
	"net/http"
	"strings"
	"sync"
)

type CompressConfig struct {
	MinSize      int //压缩数据大小阈值
	GzipLevel    int // gzip压缩等级
	DeflateLevel int // deflate压缩等级
}

// 支持 gzip | deflate 压缩，小数据不压缩
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

type bufferedResponseWriter struct {
	http.ResponseWriter
	buffer *bytes.Buffer
	status int
	header http.Header
}

func (w *bufferedResponseWriter) Write(b []byte) (int, error) {
	if w.buffer == nil {
		w.buffer = bytes.NewBuffer(nil)
	}
	return w.buffer.Write(b)
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *bufferedResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func handleCompress(w http.ResponseWriter, r *http.Request, next http.Handler, minSize int, encoding string, pool *sync.Pool) { // {{{

	w.Header().Add("Vary", "Accept-Encoding")

	buffered := &bufferedResponseWriter{ResponseWriter: w}
	next.ServeHTTP(buffered, r)

	for k, v := range buffered.header {
		w.Header()[k] = v
	}

	// 检查是否需要压缩
	if buffered.buffer == nil || buffered.buffer.Len() < minSize {
		// 数据太小，直接写入
		if buffered.status != 0 {
			w.WriteHeader(buffered.status)
		}
		if buffered.buffer != nil {
			w.Write(buffered.buffer.Bytes())
		}
		return
	}

	// 从对象池获取压缩器
	var compressedWriter io.WriteCloser

	if encoding == "gzip" {
		gz := pool.Get().(*gzip.Writer)
		gz.Reset(w)
		compressedWriter = gz

		defer func() {
			gz.Close()
			pool.Put(gz)
		}()
	} else { // deflate
		fl := pool.Get().(*flate.Writer)
		fl.Reset(w)
		compressedWriter = fl

		defer func() {
			fl.Close()
			pool.Put(fl)
		}()
	}

	w.Header().Set("Content-Encoding", encoding)

	if buffered.status != 0 {
		w.WriteHeader(buffered.status)
	}
	compressedWriter.Write(buffered.buffer.Bytes())
} // }}}
