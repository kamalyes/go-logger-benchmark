/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 09:23:36
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 09:23:36
 * @FilePath: \go-logger-benchmark\bench_kv_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — KV 字段场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// benchSequential 串行执行场景日志函数
func benchSequential(b *testing.B, log func()) {
	b.Helper()
	b.ReportAllocs()
	for range b.N {
		log()
	}
}

// benchParallel 并行执行场景日志函数（压输出锁竞争与并发分配）
func benchParallel(b *testing.B, log func()) {
	b.Helper()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			log()
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// KV10：5 对（10 个）混合类型字段
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_KV10(b *testing.B) { benchSequential(b, logKV10GoLogger) }
func BenchmarkZap_KV10(b *testing.B)      { benchSequential(b, logKV10Zap) }
func BenchmarkZerolog_KV10(b *testing.B)  { benchSequential(b, logKV10Zerolog) }
func BenchmarkSlog_KV10(b *testing.B)     { benchSequential(b, logKV10Slog) }

// ─── Parallel ───

func BenchmarkGoLogger_KV10_Parallel(b *testing.B) { benchParallel(b, logKV10GoLogger) }
func BenchmarkZap_KV10_Parallel(b *testing.B)      { benchParallel(b, logKV10Zap) }
func BenchmarkZerolog_KV10_Parallel(b *testing.B)  { benchParallel(b, logKV10Zerolog) }
func BenchmarkSlog_KV10_Parallel(b *testing.B)     { benchParallel(b, logKV10Slog) }

// ─── Caller ───

func BenchmarkGoLogger_KV10_Caller(b *testing.B) { benchSequential(b, logKV10GoLoggerCaller) }
func BenchmarkZap_KV10_Caller(b *testing.B)      { benchSequential(b, logKV10ZapCaller) }
func BenchmarkZerolog_KV10_Caller(b *testing.B)  { benchSequential(b, logKV10ZerologCaller) }
func BenchmarkSlog_KV10_Caller(b *testing.B)     { benchSequential(b, logKV10SlogCaller) }

// ─── Zap 独有：Sugared（KV 装箱 API 与 Core 强类型 Field 的差距） ───

func BenchmarkZap_KV10_Sugared(b *testing.B) { benchSequential(b, logKV10ZapSugared) }
