package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogLevel int

// Log levels
const (
	LevelNone   LogLevel = 0x00
	LevelCustom LogLevel = 0x01 //自定义级别
	LevelFatal  LogLevel = 0x02
	LevelError  LogLevel = 0x04
	LevelWarn   LogLevel = 0x08
	LevelNotice LogLevel = 0x10
	LevelInfo   LogLevel = 0x20
	LevelDebug  LogLevel = 0x40
	LevelAll    LogLevel = 0xFF
)

func (ll LogLevel) Name() string {
	switch ll {
	case LevelCustom:
		return "CUSTOM"
	case LevelFatal:
		return "FATAL"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelNotice:
		return "NOTICE"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	default:
		return ""
	}
}

type Logger struct {
	level          LogLevel                          // 日志级别
	writer         io.Writer                         // 日志输出位置
	writers        map[io.Writer]struct{}            // 当使用 MultiWriter 时，保存所有 writer
	writer_accepts map[io.Writer]map[string]struct{} // 当使用 MultiWriter 时， 不同writer接受的日志级别，未指定则全接受
	encoder        Encoder                           //输出格式化
	//buffer         *RingBuffer                  // 日志缓冲区
	stopChan      chan struct{}  // 停止信号
	wg            sync.WaitGroup // 等待日志写完
	mu            sync.Mutex
	timeFormat    string        // 时间格式
	useQueue      bool          //是否使用异步队列
	flushInterval time.Duration //自动刷新 bufio 时间间隔，避免buffed size过小时无法刷新的问题
	prefix        string        //前缀
	bulk          *Bulk
	queue         chan *Bulk
	queueSize     int
	debug         bool
	traceFile     bool // 是否打印文件名
	showLevel     bool //是否显示级别名称
}

var (
	DefaultWriter        io.Writer     = os.Stdout
	DefaultEncoder       Encoder       = &TextEncoder{}
	DefaultLogLevel      LogLevel      = LevelAll
	DefaultTimeFormat    string        = "" //空时使用时间戳
	DefaultFlushInterval time.Duration = 3 * time.Second
	DefaultQueueSize     int           = 1024
	DefaultTraceDepth    int           = 1
	DefaultPrefix        string        = ""
)

type LogOptions struct {
	QueueSize     int
	Level         int
	BulkSize      int
	TraceFile     bool
	ShowLevel     bool
	UseQueue      bool
	FileEnabled   bool
	FileRule      *LogFileRule
	FileLevelRule map[string]*LogFileRule
	Prefix        string
	TimeFormat    string
}

type LogFileRule struct {
	FileSize       int
	BufferSize     int
	CompressBefore int
	RemoveBefore   int
	Compress       bool
	Remove         bool
	Path           string
	NamingFormat   string
}

func NewLogger(opts ...*LogOptions) (*Logger, error) { // {{{
	use_queue := true
	trace_file := false
	show_level := false
	file_enabled := false
	var file_rule *LogFileRule
	var file_level_rule map[string]*LogFileRule

	if len(opts) > 0 {
		opt := opts[0]

		if opt.QueueSize > 0 {
			DefaultQueueSize = opt.QueueSize
		}

		if opt.BulkSize > 0 {
			DefaultBulkSize = opt.BulkSize
		}

		if opt.Level > 0 {
			DefaultLogLevel = LogLevel(opt.Level)
		}

		if opt.TimeFormat != "" {
			DefaultTimeFormat = opt.TimeFormat
		}

		if opt.Prefix != "" {
			DefaultPrefix = opt.Prefix
		}

		trace_file = opt.TraceFile
		show_level = opt.ShowLevel
		use_queue = opt.UseQueue
		file_rule = opt.FileRule
		file_level_rule = opt.FileLevelRule
		file_enabled = opt.FileEnabled
	}

	logger := &Logger{
		level:          DefaultLogLevel,
		writer:         DefaultWriter,
		writers:        make(map[io.Writer]struct{}),
		writer_accepts: make(map[io.Writer]map[string]struct{}),
		encoder:        DefaultEncoder,
		//buffer:         NewRingBuffer(ring_size),
		timeFormat:    DefaultTimeFormat,
		stopChan:      make(chan struct{}),
		useQueue:      use_queue,
		flushInterval: DefaultFlushInterval,
		bulk:          GetBulk(),
		queue:         make(chan *Bulk, DefaultQueueSize),
		queueSize:     DefaultQueueSize,
		traceFile:     trace_file,
		showLevel:     show_level,
		prefix:        DefaultPrefix,
	}
	//logger.SetWriter(DefaultWriter)

	if file_enabled {
		err := logger.UseFileWriter(file_rule, file_level_rule)
		if err != nil {
			return nil, err
		}
	}

	//后台处理日志
	logger.wg.Add(1)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("logger err:", err)
			}
		}()

		logger.processLogs()
	}()

	return logger, nil
} // }}}

func (l *Logger) UseFileWriter(file_rule *LogFileRule, file_level_rule map[string]*LogFileRule) error { // {{{
	writers := map[string]io.Writer{}
	writer_levels := map[io.Writer][]string{}
	conf := map[string]string{}

	var err error
	for level_name, rule := range file_level_rule { // {{{
		key := rule.Path + rule.NamingFormat
		conf[level_name] = key
		writer, ok := writers[key]
		if !ok {
			var writer_opts []FuncOption

			if rule.BufferSize > 0 {
				writer_opts = append(writer_opts, WithBufferSize(rule.BufferSize))
			}

			if rule.FileSize > 0 {
				writer_opts = append(writer_opts, WithFileSize(rule.FileSize))
			}

			writer_opts = append(writer_opts, WithCompress(rule.Compress))
			if rule.Compress {
				writer_opts = append(writer_opts, WithCompressBefore(rule.CompressBefore))
			}

			writer_opts = append(writer_opts, WithRemove(rule.Remove))
			if rule.Remove {
				writer_opts = append(writer_opts, WithRemoveBefore(rule.RemoveBefore))
			}

			writer, err = NewFileWriter(rule.Path, rule.NamingFormat, writer_opts...)
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

	def_key := file_rule.Path + file_rule.NamingFormat
	for _, level_name := range []string{"FATAL", "ERROR", "WARN", "NOTICE", "INFO", "DEBUG"} {
		if _, ok := conf[level_name]; !ok {
			conf[level_name] = def_key
			writer, ok := writers[def_key]
			if !ok {
				var writer_opts []FuncOption

				if file_rule.BufferSize > 0 {
					writer_opts = append(writer_opts, WithBufferSize(file_rule.BufferSize))
				}

				if file_rule.FileSize > 0 {
					writer_opts = append(writer_opts, WithFileSize(file_rule.FileSize))
				}

				writer_opts = append(writer_opts, WithCompress(file_rule.Compress))
				if file_rule.Compress {
					writer_opts = append(writer_opts, WithCompressBefore(file_rule.CompressBefore))
				}

				writer_opts = append(writer_opts, WithRemove(file_rule.Remove))
				if file_rule.Remove {
					writer_opts = append(writer_opts, WithRemoveBefore(file_rule.RemoveBefore))
				}

				writer, err = NewFileWriter(file_rule.Path, file_rule.NamingFormat, writer_opts...)
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

	l.RemoveWriter(DefaultWriter)
	for writer, level_names := range writer_levels {
		l.AddWriter(writer, level_names...)
	}

	return nil
} // }}}

func (l *Logger) processLogs() { // {{{
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopChan:
			// 处理队列中未处理完的数据
			for len(l.queue) > 0 {
				bulk := <-l.queue
				l.prepareWrite(bulk.GetEntrys()...)
				PutBulk(bulk)
			}

			// 处理未进入队列的数据
			l.prepareWrite(l.bulk.GetEntrys()...)
			PutBulk(l.bulk)
			l.bulk = nil
			return
		case bulk := <-l.queue:
			l.prepareWrite(bulk.GetEntrys()...)
			PutBulk(bulk)
		case <-ticker.C:
			// 定时刷新缓冲区
			l.autoFlush()
		}
	}
} // }}}

func (l *Logger) autoFlush() { // {{{
	l.mu.Lock()
	defer l.mu.Unlock()
	//l.buffer.Put(l.bulk)

	select {
	case l.queue <- l.bulk:
		l.bulk = GetBulk()

		//fmt.Println("log queue size:", l.queueSize, len(l.queue))
	default:
		fmt.Println("log channel full, queue size:", l.queueSize, len(l.queue))
		return
	}

	for w := range l.writers {
		if fw, ok := w.(*FileWriter); ok {
			fw.AutoFlush(l.flushInterval)
		}
	}

} // }}}

func (l *Logger) SetDebug(d bool) {
	l.debug = d
}

// 开启/关闭队列
func (l *Logger) UseQueue(use bool) {
	l.useQueue = use
}

// 设置日志级别
func (l *Logger) SetLevel(lvl LogLevel) { // {{{
	l.level = lvl
} // }}}

// 指定前缀
func (l *Logger) SetPrefix(p string) { // {{{
	l.prefix = p
} // }}}

// 开启/关闭 打印文件名
func (l *Logger) TraceFile(b bool) {
	l.traceFile = b
}

func (l *Logger) log(level_name string, args ...any) { // {{{
	entry := GetEntry(false, l.timeFormat, level_name, "", args...)
	l.write(entry)
} // }}}

func (l *Logger) logf(level_name, msg string, args ...any) { // {{{
	entry := GetEntry(true, l.timeFormat, level_name, msg, args...)
	l.write(entry)
} // }}}

// 实现 io.Writer 接口, 可以供标准库log使用
func (l *Logger) Write(p []byte) (int, error) { // {{{
	entry := GetEntry(false, l.timeFormat, LevelInfo.Name(), string(p))
	l.write(entry)

	return len(p), nil
} // }}}

func (l *Logger) write(entry *Entry) { // {{{
	if l.traceFile {
		entry.File = l.traceCallerFile()
	}

	if l.useQueue {
		//阻塞写入，为保证性能需要合理设置缓冲区
		//l.buffer.Put(entry)
		l.mu.Lock()
		defer l.mu.Unlock()

		l.bulk.Append(entry)

		if l.bulk.IsFull() {
			//l.buffer.Put(l.bulk)
			select {
			case l.queue <- l.bulk:
				l.bulk = GetBulk()
			default: // 队列写满后尝试直接写
				l.prepareWrite(l.bulk.GetEntrys()...)
				PutBulk(l.bulk)
				l.bulk = GetBulk()

				//	fmt.Println("log channel full, queue size:", l.queueSize, len(l.queue))
			}
		}

	} else {
		l.prepareWrite(entry)
	}
} // }}}

// entry编码处理后转到 writer
func (l *Logger) prepareWrite(entrys ...*Entry) { // {{{
	if len(l.writers) == 0 {
		l.SetWriter(DefaultWriter)
	}

	for _, entry := range entrys {
		defer PutEntry(entry) // 归还对象池

		entryLevel := entry.Level
		if !l.showLevel {
			entry.Level = ""
		}

		encoded, err := l.encoder.Encode(entry)
		if err != nil {
			fmt.Println("logger err:", err)
			return
		}

		var line []byte
		if l.prefix != "" {
			line = append(line, []byte(l.prefix)...)
		}

		line = append(line, encoded...)
		line = append(line, '\n')

		for w := range l.writers {
			//if lw, ok := w.(LevelWriter); ok && !lw.Accepts(entry.Level) {
			//	continue // 跳过不接受的级别
			//}

			if la, ok := l.writer_accepts[w]; ok { //未设置则全接受
				if _, ok = la[entryLevel]; !ok {
					continue
				}
			}

			_, err := w.Write(line)
			if err != nil {
				fmt.Println("logger err:", err)
			}
		}
	}
} // }}}

func (l *Logger) check(level LogLevel) bool {
	return l.level&level != 0
}

func (l *Logger) traceCallerFile() string { // {{{
	depth := DefaultTraceDepth
	if _, file, line, ok := runtime.Caller(3 + depth); ok {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}

		return file + ":" + strconv.Itoa(line)
	}

	return ""
} // }}}

func (l *Logger) Fatal(args ...any) { // {{{
	if !l.check(LevelFatal) {
		return
	}

	l.log(LevelFatal.Name(), args...)
} // }}}

func (l *Logger) Error(args ...any) { // {{{
	if !l.check(LevelError) {
		return
	}

	l.log(LevelError.Name(), args...)
} // }}}

func (l *Logger) Warn(args ...any) { // {{{
	if !l.check(LevelWarn) {
		return
	}

	l.log(LevelWarn.Name(), args...)
} // }}}

func (l *Logger) Info(args ...any) { // {{{
	if !l.check(LevelInfo) {
		return
	}

	l.log(LevelInfo.Name(), args...)
} // }}}

func (l *Logger) Notice(args ...any) { // {{{
	if !l.check(LevelNotice) {
		return
	}

	l.log(LevelNotice.Name(), args...)
} // }}}

func (l *Logger) Debug(args ...any) { // {{{
	if !l.check(LevelDebug) {
		return
	}

	l.traceFile = true
	l.log(LevelDebug.Name(), args...)
} // }}}

// Printf-style logging
func (l *Logger) Fatalf(msg string, args ...any) { // {{{
	if !l.check(LevelFatal) {
		return
	}

	l.logf(LevelFatal.Name(), msg, args...)
} // }}}

func (l *Logger) Errorf(msg string, args ...any) { // {{{
	if !l.check(LevelError) {
		return
	}

	l.logf(LevelError.Name(), msg, args...)
} // }}}

func (l *Logger) Warnf(msg string, args ...any) { // {{{
	if !l.check(LevelWarn) {
		return
	}

	l.logf(LevelWarn.Name(), msg, args...)
} // }}}

func (l *Logger) Noticef(msg string, args ...any) { // {{{
	if !l.check(LevelNotice) {
		return
	}

	l.logf(LevelNotice.Name(), msg, args...)
} // }}}

func (l *Logger) Infof(msg string, args ...any) { // {{{
	if !l.check(LevelInfo) {
		return
	}

	l.logf(LevelInfo.Name(), msg, args...)
} // }}}

func (l *Logger) Debugf(msg string, args ...any) { // {{{
	if !l.check(LevelDebug) {
		return
	}

	l.traceFile = true
	l.logf(LevelDebug.Name(), msg, args...)
} // }}}

// 自定义级别日志
func (l *Logger) Log(level_name string, args ...any) { // {{{
	if !l.check(LevelCustom) {
		return
	}

	l.log(level_name, args...)
} // }}}

func (l *Logger) Logf(level_name, msg string, args ...any) { // {{{
	if !l.check(LevelCustom) {
		return
	}

	l.logf(level_name, msg, args...)
} // }}}

// 只使用指定的 writer (关闭其他writer)
func (l *Logger) SetWriter(w io.Writer) { // {{{
	l.writer = w
	l.writers = map[io.Writer]struct{}{w: struct{}{}}
} // }}}

// 增加新的 writer, 将使用 MultiWriter, 同时指定接收的日志级别名称，不在范围内的不写入此 writer
// 未指定级别名则表示全接受, 预设级别使用 LogLevel.Name(), 自定义级别使用自定义的名称
func (l *Logger) AddWriter(w io.Writer, accept_level_names ...string) { // {{{
	if _, exists := l.writers[w]; !exists {
		l.writers[w] = struct{}{}
		if len(accept_level_names) > 0 {
			l.writer_accepts[w] = map[string]struct{}{}
			for _, level_name := range accept_level_names {
				l.writer_accepts[w][level_name] = struct{}{}
			}
		}
		l.updateMultiWriter()
	}
} // }}}

func (l *Logger) RemoveWriter(w io.Writer) { // {{{
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.writers[w]; exists {
		delete(l.writers, w)
		l.updateMultiWriter()
	}
} // }}}

func (l *Logger) updateMultiWriter() { // {{{
	writers := make([]io.Writer, 0, len(l.writers))
	for w := range l.writers {
		writers = append(writers, w)
	}

	l.writer = io.MultiWriter(writers...)
} // }}}

// 关闭所有 writers ( 保留Stdout和Stderr)
func (l *Logger) Close() { // {{{
	if l.stopChan == nil {
		return // 已经关闭
	}

	fmt.Println("logger close")
	//l.buffer.Close()

	close(l.stopChan)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	for w := range l.writers {
		if closer, ok := w.(io.Closer); ok && w != os.Stdout && w != os.Stderr {
			closer.Close()
		}
	}

	l.writers = make(map[io.Writer]struct{})
	l.updateMultiWriter()

	return
} // }}}
