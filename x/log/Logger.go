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
	level          LogLevel                     // 日志级别
	writer         io.Writer                    // 日志输出位置
	writers        map[io.Writer]struct{}       // 当使用 MultiWriter 时，保存所有 writer
	writer_accepts map[io.Writer]map[string]int // 当使用 MultiWriter 时， 不同writer接受的日志级别，未指定则全接受
	encoder        Encoder                      //输出格式化
	//buffer         *RingBuffer                  // 日志缓冲区
	stopChan      chan struct{}  // 停止信号
	wg            sync.WaitGroup // 等待日志写完
	mu            sync.Mutex
	timeFormat    string        // 时间格式
	useSync       bool          //是否使用异步队列
	flushInterval time.Duration //自动刷新 bufio 时间间隔，避免buffed size过小时无法刷新的问题
	prefix        string        //前缀
	bulk          *Bulk
	queue         chan *Bulk
	queueSize     int
	debug         bool
	printFileName bool // 是否打印文件名
}

var (
	DefaultWriter        io.Writer     = os.Stdout
	DefaultEncoder       Encoder       = &TextEncoder{}
	DefaultLogLevel      LogLevel      = LevelAll
	DefaultTimeFormat    string        = "2006-01-02 15:04:05.000"
	DefaultFlushInterval time.Duration = 3 * time.Second
	DefaultQueueSize     int           = 1024
	DefaultTraceDepth    int           = 1
)

func NewLogger(queue_sizes ...int) *Logger { // {{{
	var queue_size int
	if len(queue_sizes) > 0 {
		queue_size = queue_sizes[0]
	}

	if queue_size <= 0 {
		queue_size = DefaultQueueSize
	}

	logger := &Logger{
		level:          DefaultLogLevel,
		writer:         DefaultWriter,
		writers:        make(map[io.Writer]struct{}),
		writer_accepts: make(map[io.Writer]map[string]int),
		encoder:        DefaultEncoder,
		//buffer:         NewRingBuffer(ring_size),
		timeFormat:    DefaultTimeFormat,
		stopChan:      make(chan struct{}),
		useSync:       true,
		flushInterval: DefaultFlushInterval,
		bulk:          GetBulk(),
		queue:         make(chan *Bulk, queue_size),
		queueSize:     queue_size,
	}
	//logger.SetWriter(DefaultWriter)

	//后台处理日志
	logger.wg.Add(1)
	go logger.processLogs()

	return logger
} // }}}

func (l *Logger) processLogs() { // {{{
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopChan:
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
		if l.debug {
			fmt.Println("log queue size:", l.queueSize, len(l.queue))
		}
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
func (l *Logger) UseSync(use bool) {
	l.useSync = use
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
func (l *Logger) PrintFileName(b bool) {
	l.printFileName = b
}

func (l *Logger) log(level_name string, args ...any) { // {{{
	entry := GetEntry()
	entry.Time = time.Now()
	entry.Level = level_name
	entry.Args = args
	entry.Formated = false

	l.write(entry)
} // }}}

func (l *Logger) logf(level_name, msg string, args ...any) { // {{{
	entry := GetEntry()
	entry.Time = time.Now()
	entry.Level = level_name
	entry.Msg = msg
	entry.Args = args
	entry.Formated = true

	l.write(entry)
} // }}}

// 实现 io.Writer 接口, 可以供标准库log使用
func (l *Logger) Write(p []byte) (int, error) { // {{{
	entry := GetEntry()
	entry.Time = time.Now()
	entry.Level = LevelInfo.Name()
	entry.Msg = string(p)

	l.write(entry)

	return len(p), nil
} // }}}

func (l *Logger) write(entry *Entry) { // {{{
	if l.printFileName {
		entry.File = l.traceFile()
	}

	if l.useSync {
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

		encoded, err := l.encoder.Encode(entry)
		if err != nil {
			fmt.Println(err)
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
				if _, ok = la[entry.Level]; !ok {
					continue
				}
			}

			w.Write(line)
		}
	}
} // }}}

func (l *Logger) check(level LogLevel) bool {
	return l.level&level != 0
}

func (l *Logger) traceFile() string { // {{{
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

	l.printFileName = true
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

	l.printFileName = true
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
			l.writer_accepts[w] = map[string]int{}
			for _, level_name := range accept_level_names {
				l.writer_accepts[w][level_name] = 1
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

	//fmt.Println("log close")
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
