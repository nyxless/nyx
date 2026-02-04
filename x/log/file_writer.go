package log

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	DefaultBufferSize     int   = 4096              // 默认缓冲区大小 4k
	DefaultFileSize       int64 = 500 * 1024 * 1024 // 默认单个日志文件大小 500MB
	DefaultCompress       bool  = false             // 默认不压缩历史文件
	DefaultCompressBefore int   = 60 * 24           // 压缩24小时前修改的文件，单位分钟
	DefaultRemove         bool  = false             // 默认不删除历史文件
	DefaultRemoveBefore   int   = 60 * 24 * 7       // 删除7天前修改的文件，单位分钟
)

type FileWriter struct {
	file          *os.File      // 当前日志文件
	writer        *bufio.Writer // 缓冲
	mu            sync.Mutex
	logPath       string    // 日志目录
	namingFormat  string    //时间模板，用于文件名
	fileName      string    // 当前写入的文件名
	currentSize   int64     // 当前日志文件大小（字节）
	lastFlushTime time.Time //上次刷新缓存时间
	options       *Options  // 配置选项
}

type Options struct {
	maxBufferSize  int   // 缓冲区大小（字节）
	maxFileSize    int64 // 单个日志文件最大大小（字节）
	compress       bool  // 是否压缩旧日志
	compressBefore int   // 压缩N分钟前修改旧日志, 单位分钟
	remove         bool  // 是否删除旧文件
	removeBefore   int   // 删除N分钟前修改旧日志, 单位分钟
}

type FuncOption func(r *Options)

// 设置配置 maxBufferSize
func WithBufferSize(i int) FuncOption { // {{{
	return func(o *Options) {
		o.maxBufferSize = i
	}
} // }}}

// 设置配置 maxFileSize
func WithFileSize(i int) FuncOption { // {{{
	return func(o *Options) {
		o.maxFileSize = int64(i)
	}
} // }}}

// 设置配置 compress
func WithCompress(b bool) FuncOption { // {{{
	return func(o *Options) {
		o.compress = b
	}
} // }}}

// 设置配置 compressBefore
func WithCompressBefore(i int) FuncOption { // {{{
	return func(o *Options) {
		o.compressBefore = i
	}
} // }}}

// 设置配置 remove
func WithRemove(b bool) FuncOption { // {{{
	return func(o *Options) {
		o.remove = b
	}
} // }}}

// 设置配置 removeBefore
func WithRemoveBefore(i int) FuncOption { // {{{
	return func(o *Options) {
		o.removeBefore = i
	}
} // }}}

func NewFileWriter(logPath, namingFormat string, opts ...FuncOption) (*FileWriter, error) { // {{{
	options := &Options{
		maxBufferSize:  DefaultBufferSize,
		maxFileSize:    DefaultFileSize,
		compress:       DefaultCompress,
		compressBefore: DefaultCompressBefore,
		remove:         DefaultRemove,
		removeBefore:   DefaultRemoveBefore,
	}

	for _, opt := range opts {
		opt(options)
	}

	fw := &FileWriter{
		logPath:       logPath,
		namingFormat:  namingFormat,
		lastFlushTime: time.Now(),
		options:       options,
	}

	// 初始化日志文件
	//if err := fw.rotateLog(); err != nil {
	//	return nil, err
	//}

	return fw, nil
} // }}}

// Write 实现 io.Writer 接口
func (fw *FileWriter) Write(p []byte) (int, error) { // {{{
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// 检查是否需要轮转
	if err := fw.checkRotate(); err != nil {
		return 0, err
	}

	before_size := fw.writer.Buffered()
	// 写入缓冲区
	n, err := fw.writer.Write(p)
	if err != nil {
		return n, err
	}
	after_size := fw.writer.Buffered()
	if before_size+n > after_size {
		fw.lastFlushTime = time.Now()
	}

	// 更新当前文件大小
	fw.currentSize += int64(n)
	return n, nil
} // }}}

// 定时自动刷新缓存
func (fw *FileWriter) AutoFlush(interval time.Duration) { // {{{
	now := time.Now()
	if now.Sub(fw.lastFlushTime) >= interval && fw.writer != nil && fw.writer.Buffered() > 0 {
		fw.writer.Flush()
		fw.lastFlushTime = now
		//fmt.Println("auto flush")
	}
} // }}}

// Close 关闭日志写入器
func (fw *FileWriter) Close() error { // {{{
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.writer != nil {
		// 刷新缓冲区
		if err := fw.writer.Flush(); err != nil {
			return err
		}
	}

	if fw.file != nil {
		// 关闭文件
		return fw.file.Close()
	}

	return nil
} // }}}

// 检查是否需要轮转（时间或大小）
func (fw *FileWriter) checkRotate() error {
	// 获取当前时间对应的文件名
	if fw.getCurrentName() != filepath.Base(fw.fileName) {
		return fw.rotateLog() // 时间轮转
	}

	// 检查文件大小
	if fw.currentSize >= fw.options.maxFileSize {
		return fw.rotateLog() // 大小轮转
	}

	return nil
}

// 根据namingFormat生成时间标识文件名
func (fw *FileWriter) getCurrentName() string {
	now := time.Now()
	return now.Format(fw.namingFormat)
}

// rotateLog 执行日志轮转
func (fw *FileWriter) rotateLog() error { // {{{
	// 刷新并关闭当前文件
	if fw.writer != nil {
		if err := fw.writer.Flush(); err != nil {
			return err
		}
	}

	if fw.file != nil {
		if err := fw.file.Close(); err != nil {
			return err
		}
	}

	// 生成新文件名
	fw.fileName = fw.genFilename()

	// 确保日志目录存在
	if err := os.MkdirAll(path.Dir(fw.fileName), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// 创建新文件
	file, err := os.OpenFile(fw.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// 更新状态
	fw.file = file
	fw.writer = bufio.NewWriterSize(file, fw.options.maxBufferSize)
	if fileInfo, err := file.Stat(); err == nil {
		fw.currentSize = fileInfo.Size()
	} else {
		fw.currentSize = 0
	}

	if fw.options.compress || fw.options.remove {
		go fw.cleanOldFile()
	}

	return nil
} // }}}

// 生成带路径带序号的文件名
func (fw *FileWriter) genFilename() string { // {{{
	filename := filepath.Join(fw.logPath, fw.getCurrentName())

	info, err := os.Stat(filename)
	if err != nil {
		return filename
	}

	// 如果当前文件未超大小，继续写入
	if info.Size() < fw.options.maxFileSize {
		return filename
	}

	// 查找已有文件序号
	maxSeq := 0
	if matches, _ := filepath.Glob(filename + ".*"); matches != nil {
		for _, match := range matches {
			seq := extractSequence(match, filename)
			if seq > maxSeq {
				maxSeq = seq
			}
		}
	}

	seqfile := fmt.Sprintf("%s.%d", fw.fileName, maxSeq+1)
	os.Rename(filename, seqfile)

	return filename
} // }}}

// 从文件名提取序号
func extractSequence(match, filename string) int {
	trimmed := strings.TrimPrefix(match, filename+".")
	trimmed = strings.TrimSuffix(trimmed, ".gz")

	var seq int
	fmt.Sscanf(trimmed, "%d", &seq)
	return seq
}

func (fw *FileWriter) cleanOldFile() { // {{{
	// 删除旧日志（如果启用）
	if fw.options.remove {
		fw.removeFiles()
	}

	// 压缩旧日志（如果启用）
	if fw.options.compress {
		fw.compressFiles()
	}
} // }}}

func (fw *FileWriter) removeFiles() error { // {{{
	cutoff := time.Now().Add(-(time.Duration(fw.options.removeBefore) * time.Minute))

	err := filepath.Walk(fw.logPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if matched, err := filepath.Match(maskTime(filepath.Base(fw.namingFormat))+".*", filepath.Base(path)); err == nil && matched {
			if !info.IsDir() && info.ModTime().Before(cutoff) {
				if fw.file != nil && path == fw.file.Name() {
					fmt.Println("skip current file")
					return nil
				}

				if err := os.Remove(path); err != nil {
					fmt.Printf("删除失败: %s, 错误: %v\n", path, err)
				} else {
					fmt.Printf("已删除: %s\n", path)
				}
			}
		}
		return nil
	})

	return err
} // }}}

func (fw *FileWriter) compressFiles() error { // {{{
	cutoff := time.Now().Add(-(time.Duration(fw.options.removeBefore) * time.Minute))

	err := filepath.Walk(fw.logPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if matched, err := filepath.Match(maskTime(filepath.Base(fw.namingFormat)), filepath.Base(path)); err == nil && matched {
			if !info.IsDir() && info.ModTime().Before(cutoff) {
				fw.compressFile(path)
			}
		}
		return nil
	})

	return err
} // }}}

// 压缩日志文件
func (fw *FileWriter) compressFile(filename string) { // {{{
	// 跳过正在写入的文件和已压缩文件
	if strings.HasSuffix(filename, ".gz") {
		return
	}

	if fw.file != nil && filename == fw.file.Name() {
		return
	}

	// 打开原文件
	src, err := os.Open(filename)
	if err != nil {
		return
	}
	defer src.Close()

	// 创建压缩文件
	dst, err := os.Create(filename + ".gz")
	if err != nil {
		return
	}
	defer dst.Close()

	// 执行压缩
	gz := gzip.NewWriter(dst)
	defer gz.Close()

	if _, err := io.Copy(gz, src); err == nil {
		os.Remove(filename) // 压缩成功后删除原文件
	}
} // }}}

// 替换时间模板为模糊匹配字符串
func maskTime(s string) string { // {{{
	time_vals := []string{"2006", "15", "01", "02", "04", "05", "06"}

	res := s
	for _, v := range time_vals {
		res = strings.ReplaceAll(res, v, "*") //strings.Repeat("*", len(v)))
	}

	return res
} // }}}
