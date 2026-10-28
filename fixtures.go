/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-logger-benchmark\fixtures.go
 * @Description: 扩展场景测试装置 — 输出目标、子 Logger、类型化值、text 格式
 *
 * io.Discard 基线只能测出格式化成本，而生产里 85~97% 的单条成本在写出路径上，
 * 子 Logger 与 error 值又是服务端最常见的两种用法，故单列这几组场景
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	gologger "github.com/kamalyes/go-logger"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ============================================================================
// 公共测试数据
// ============================================================================

// benchErr 典型的下游错误（结构化日志里最常见的非原始类型）
var benchErr = errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

// benchDur 典型的耗时值（time.Duration 是具名 int64，各库的处理方式不同）
const benchDur = 1500 * time.Microsecond

// benchPayload 失控的大字段：考察各库在单条日志体量异常时的行为
var benchPayload = strings.Repeat("x", 64*1024)

// benchChildFields 子 Logger 的常驻字段（服务名、实例、区域等每条日志都要带的身份信息）
var benchChildFields = []struct {
	key string
	val string
}{
	{"service", "order-svc"},
	{"instance", "order-svc-7d9f-abcde"},
	{"region", "cn-shanghai"},
	{"env", "prod"},
}

// ============================================================================
// 并发安全的缓冲写出
// ============================================================================

// lockedWriter 给非并发安全的 writer（如 bufio.Writer）加锁
//
// zap 自带 BufferedWriteSyncer、go-logger 自带异步批量管道，而 zerolog 与 slog
// 需要调用方自备缓冲与并发保护——这一差异本身就是对比的一部分
type lockedWriter struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func newLockedWriter(w io.Writer, size int) *lockedWriter {
	return &lockedWriter{w: bufio.NewWriterSize(w, size)}
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func (l *lockedWriter) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Flush()
}

// sinkBufferSize 缓冲写出的统一缓冲区大小，保证四库口径一致
const sinkBufferSize = 64 * 1024

// ============================================================================
// 输出目标装置
// ============================================================================

// sinkSet 同一输出目标下的四库 Logger
type sinkSet struct {
	goLogger *gologger.Logger
	zap      *zap.Logger
	zerolog  zerolog.Logger
	slog     *slog.Logger
	flush    func()
}

// newRawFileSinks 四库均直写 *os.File（无缓冲，每条一次 write 系统调用）
func newRawFileSinks(file *os.File) sinkSet {
	return sinkSet{
		goLogger: baselineGoLogger(file, false),
		zap: zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(file),
			zapcore.InfoLevel,
		)),
		zerolog: zerolog.New(file).With().Timestamp().Logger(),
		slog:    slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})),
		flush:   func() {},
	}
}

// newBufferedFileSinks 四库均按各自推荐方式做缓冲写出
//
//	go-logger  内建异步批量管道（队列 + 字节阈值 + 定时刷）
//	zap        BufferedWriteSyncer（自带锁）
//	zerolog    调用方自备 bufio + 锁
//	slog       调用方自备 bufio + 锁
func newBufferedFileSinks(file *os.File) sinkSet {
	glWriter := gologger.NewConsoleWriter(gologger.WithConsoleOutput(file))
	zapSyncer := &zapcore.BufferedWriteSyncer{WS: zapcore.AddSync(file), Size: sinkBufferSize}
	zeroWriter := newLockedWriter(file, sinkBufferSize)
	slogWriter := newLockedWriter(file, sinkBufferSize)

	return sinkSet{
		goLogger: baselineGoLogger(glWriter, false),
		zap: zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapSyncer,
			zapcore.InfoLevel,
		)),
		zerolog: zerolog.New(zeroWriter).With().Timestamp().Logger(),
		slog:    slog.New(slog.NewJSONHandler(slogWriter, &slog.HandlerOptions{Level: slog.LevelInfo})),
		flush: func() {
			_ = glWriter.Close()
			_ = zapSyncer.Stop()
			_ = zeroWriter.Flush()
			_ = slogWriter.Flush()
		},
	}
}

// ============================================================================
// 子 Logger 装置（各库的 With 语义）
// ============================================================================

// childGoLogger 构造带常驻字段的 go-logger 子 Logger
func childGoLogger(base *gologger.Logger) gologger.ILogger {
	var child gologger.ILogger = base
	for _, f := range benchChildFields {
		child = child.WithField(f.key, f.val)
	}
	return child
}

// childZap 构造带常驻字段的 zap 子 Logger
func childZap(base *zap.Logger) *zap.Logger {
	fields := make([]zap.Field, 0, len(benchChildFields))
	for _, f := range benchChildFields {
		fields = append(fields, zap.String(f.key, f.val))
	}
	return base.With(fields...)
}

// childZerolog 构造带常驻字段的 zerolog 子 Logger
func childZerolog(base zerolog.Logger) zerolog.Logger {
	ctx := base.With()
	for _, f := range benchChildFields {
		ctx = ctx.Str(f.key, f.val)
	}
	return ctx.Logger()
}

// childSlog 构造带常驻字段的 slog 子 Logger
func childSlog(base *slog.Logger) *slog.Logger {
	attrs := make([]any, 0, len(benchChildFields)*2)
	for _, f := range benchChildFields {
		attrs = append(attrs, f.key, f.val)
	}
	return base.With(attrs...)
}

// 预先构造好的子 Logger（稳态场景用：构造一次，打多条）
var (
	glChild   = childGoLogger(gl)
	zlChild   = childZap(zl)
	zeloChild = childZerolog(zelog)
	slogChild = childSlog(slogL)
)

// ============================================================================
// text / console 格式装置
// ============================================================================

var (
	// glText go-logger 文本格式
	glText = gologger.NewLogger().
		WithFormat(gologger.FormatText).
		WithOutput(io.Discard).
		WithShowCaller(false).
		WithColorful(false).
		WithSanitize(gologger.SanitizeOff).
		WithMaxLineBytes(0)

	// zlText zap console 编码器
	zlText = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zapcore.InfoLevel,
	))

	// zeloText zerolog ConsoleWriter（官方的人读格式）
	zeloText = zerolog.New(zerolog.ConsoleWriter{Out: io.Discard, NoColor: true}).
			With().Timestamp().Logger()

	// slogText slog TextHandler
	slogText = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
)

// ============================================================================
// 场景日志函数
// ============================================================================

// ---------- Child：子 Logger 稳态（构造一次，打多条） ----------

func logChildGoLogger() { glChild.InfoMsg(benchMsg) }
func logChildZap()      { zlChild.Info(benchMsg) }
func logChildZerolog()  { zeloChild.Info().Msg(benchMsg) }
func logChildSlog()     { slogChild.Info(benchMsg) }

// ---------- ChildBuild：每次重新构造子 Logger 并打一条 ----------
// 服务端每请求构造一次子 Logger 的写法，考察 With 的构造成本

func logChildBuildGoLogger() { childGoLogger(gl).InfoMsg(benchMsg) }
func logChildBuildZap()      { childZap(zl).Info(benchMsg) }
func logChildBuildZerolog() {
	// zerolog.Logger 是值类型，Info() 为指针方法，需先落到可取址的变量
	child := childZerolog(zelog)
	child.Info().Msg(benchMsg)
}
func logChildBuildSlog() { childSlog(slogL).Info(benchMsg) }

// ---------- TypedValues：error + Duration + int ----------
// error 是结构化日志里最常见的非原始类型，Duration 是具名 int64，
// 两者都容易落进各库的 fmt 兜底分支

func logTypedGoLogger() {
	gl.ErrorKV(benchMsg, "err", benchErr, "cost", benchDur, "retry", 3)
}

func logTypedZap() {
	zl.Error(benchMsg, zap.Error(benchErr), zap.Duration("cost", benchDur), zap.Int("retry", 3))
}

func logTypedZerolog() {
	zelog.Error().Err(benchErr).Dur("cost", benchDur).Int("retry", 3).Msg(benchMsg)
}

func logTypedSlog() {
	slogL.Error(benchMsg, "err", benchErr, "cost", benchDur, "retry", 3)
}

// ---------- Text：文本/控制台格式 ----------

func logTextGoLogger() {
	glText.InfoKV(benchMsg, "uid", 10086, "app", "gameai", "region", "cn")
}

func logTextZap() {
	zlText.Info(benchMsg, zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("region", "cn"))
}

func logTextZerolog() {
	zeloText.Info().Int("uid", 10086).Str("app", "gameai").Str("region", "cn").Msg(benchMsg)
}

func logTextSlog() {
	slogText.Info(benchMsg, "uid", 10086, "app", "gameai", "region", "cn")
}

// ---------- BigPayload：64KB 单字段 ----------
// 考察单条日志体量异常时的行为；go-logger 默认的行长上限在此场景下生效，
// 故这里用的是关闭上限的基线装置，与其他库口径一致

func logBigGoLogger() { gl.InfoKV(benchMsg, "payload", benchPayload) }
func logBigZap()      { zl.Info(benchMsg, zap.String("payload", benchPayload)) }
func logBigZerolog()  { zelog.Info().Str("payload", benchPayload).Msg(benchMsg) }
func logBigSlog()     { slogL.Info(benchMsg, "payload", benchPayload) }
