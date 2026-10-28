/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-logger-benchmark\bench_sink_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 输出目标场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// ══════════════════════════════════════════════════════════════════════════════
// 输出目标：为什么必须单列这一组
//
// 其余场景一律输出到 io.Discard，测的是纯格式化成本。但生产里日志要真的落盘：
// 裸 *os.File 每条一次 write 系统调用，再加一把互斥，这部分能占到单条总成本的
// 85~97%。只看 io.Discard 的数字去选库或调优，会把全部精力投在不到 15% 的那段上。
//
// 本组测两种口径：
//   SinkRaw       四库均直写 *os.File（无缓冲）
//   SinkBuffered  四库各按自己推荐的方式做缓冲写出
//
// 缓冲口径的差异本身也是对比的一部分：zap 自带 BufferedWriteSyncer、
// go-logger 自带异步批量管道，zerolog 与 slog 需要调用方自备 bufio 与并发保护。
//
// 注意：本组结果受文件系统影响，跨机器不可直接比较，同机器内的相对关系才有意义。
// ══════════════════════════════════════════════════════════════════════════════

// benchSink 在给定输出目标上执行日志函数（串行）
func benchSink(b *testing.B, newSinks func(*os.File) sinkSet, pick func(sinkSet) func()) {
	b.Helper()
	file := openSinkFile(b)
	sinks := newSinks(file)
	b.Cleanup(sinks.flush)

	log := pick(sinks)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		log()
	}
	b.StopTimer()
	sinks.flush()
}

// benchSinkParallel 在给定输出目标上并行执行日志函数（压写出路径的锁竞争）
func benchSinkParallel(b *testing.B, newSinks func(*os.File) sinkSet, pick func(sinkSet) func()) {
	b.Helper()
	file := openSinkFile(b)
	sinks := newSinks(file)
	b.Cleanup(sinks.flush)

	log := pick(sinks)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log()
		}
	})
	b.StopTimer()
	sinks.flush()
}

// openSinkFile 在临时目录创建日志文件，基准结束时自动清理
func openSinkFile(b *testing.B) *os.File {
	b.Helper()
	file, err := os.Create(filepath.Join(b.TempDir(), "bench.log"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = file.Close() })
	return file
}

// 各库在 sinkSet 中的取用方式（统一为 5 对混合类型 KV，与 KV10 口径一致）
var (
	pickGoLogger = func(s sinkSet) func() {
		return func() {
			s.goLogger.InfoKV(benchMsg,
				"uid", 10086, "app", "gameai", "ns", "business",
				"conn", 9527, "region", "cn")
		}
	}
	pickZap = func(s sinkSet) func() {
		return func() {
			s.zap.Info(benchMsg,
				zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("ns", "business"),
				zap.Int("conn", 9527), zap.String("region", "cn"))
		}
	}
	pickZerolog = func(s sinkSet) func() {
		logger := s.zerolog
		return func() {
			logger.Info().
				Int("uid", 10086).Str("app", "gameai").Str("ns", "business").
				Int("conn", 9527).Str("region", "cn").
				Msg(benchMsg)
		}
	}
	pickSlog = func(s sinkSet) func() {
		return func() {
			s.slog.Info(benchMsg,
				"uid", 10086, "app", "gameai", "ns", "business",
				"conn", 9527, "region", "cn")
		}
	}
)

// ══════════════════════════════════════════════════════════════════════════════
// SinkRaw：直写 *os.File（无缓冲，每条一次 write 系统调用）
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_SinkRaw(b *testing.B) { benchSink(b, newRawFileSinks, pickGoLogger) }
func BenchmarkZap_SinkRaw(b *testing.B)      { benchSink(b, newRawFileSinks, pickZap) }
func BenchmarkZerolog_SinkRaw(b *testing.B)  { benchSink(b, newRawFileSinks, pickZerolog) }
func BenchmarkSlog_SinkRaw(b *testing.B)     { benchSink(b, newRawFileSinks, pickSlog) }

// ─── Parallel ───

func BenchmarkGoLogger_SinkRaw_Parallel(b *testing.B) {
	benchSinkParallel(b, newRawFileSinks, pickGoLogger)
}
func BenchmarkZap_SinkRaw_Parallel(b *testing.B) {
	benchSinkParallel(b, newRawFileSinks, pickZap)
}
func BenchmarkZerolog_SinkRaw_Parallel(b *testing.B) {
	benchSinkParallel(b, newRawFileSinks, pickZerolog)
}
func BenchmarkSlog_SinkRaw_Parallel(b *testing.B) {
	benchSinkParallel(b, newRawFileSinks, pickSlog)
}

// ══════════════════════════════════════════════════════════════════════════════
// SinkBuffered：各库按自己推荐的方式做缓冲写出（N 条合并为一次系统调用）
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_SinkBuffered(b *testing.B) { benchSink(b, newBufferedFileSinks, pickGoLogger) }
func BenchmarkZap_SinkBuffered(b *testing.B)      { benchSink(b, newBufferedFileSinks, pickZap) }
func BenchmarkZerolog_SinkBuffered(b *testing.B)  { benchSink(b, newBufferedFileSinks, pickZerolog) }
func BenchmarkSlog_SinkBuffered(b *testing.B)     { benchSink(b, newBufferedFileSinks, pickSlog) }

// ─── Parallel ───

func BenchmarkGoLogger_SinkBuffered_Parallel(b *testing.B) {
	benchSinkParallel(b, newBufferedFileSinks, pickGoLogger)
}
func BenchmarkZap_SinkBuffered_Parallel(b *testing.B) {
	benchSinkParallel(b, newBufferedFileSinks, pickZap)
}
func BenchmarkZerolog_SinkBuffered_Parallel(b *testing.B) {
	benchSinkParallel(b, newBufferedFileSinks, pickZerolog)
}
func BenchmarkSlog_SinkBuffered_Parallel(b *testing.B) {
	benchSinkParallel(b, newBufferedFileSinks, pickSlog)
}
