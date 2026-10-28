/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-logger-benchmark\bench_value_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 值类型与大字段场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// ══════════════════════════════════════════════════════════════════════════════
// TypedValues：error + time.Duration + int
//
// KV10 只用 string 与 int，是各库都有专门分支的类型。实际业务里 error 才是最常见
// 的非原始类型，而 time.Duration 是具名 int64——两者都容易落进 fmt 兜底分支，
// 各付一次堆分配。这一组专门把这类"真实但容易被基准忽略"的类型拎出来对比。
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_TypedValues(b *testing.B) { benchSequential(b, logTypedGoLogger) }
func BenchmarkZap_TypedValues(b *testing.B)      { benchSequential(b, logTypedZap) }
func BenchmarkZerolog_TypedValues(b *testing.B)  { benchSequential(b, logTypedZerolog) }
func BenchmarkSlog_TypedValues(b *testing.B)     { benchSequential(b, logTypedSlog) }

// ─── Parallel ───

func BenchmarkGoLogger_TypedValues_Parallel(b *testing.B) { benchParallel(b, logTypedGoLogger) }
func BenchmarkZap_TypedValues_Parallel(b *testing.B)      { benchParallel(b, logTypedZap) }
func BenchmarkZerolog_TypedValues_Parallel(b *testing.B)  { benchParallel(b, logTypedZerolog) }
func BenchmarkSlog_TypedValues_Parallel(b *testing.B)     { benchParallel(b, logTypedSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// Text：文本 / 控制台格式
//
// 其余场景都是 JSON。开发期与容器日志里文本格式同样常用，而各库的文本编码路径
// 与 JSON 完全独立（zerolog 的 ConsoleWriter 甚至要先编码成 JSON 再转排版），
// 成本结构和 JSON 不可互推。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_Text(b *testing.B) { benchSequential(b, logTextGoLogger) }
func BenchmarkZap_Text(b *testing.B)      { benchSequential(b, logTextZap) }
func BenchmarkZerolog_Text(b *testing.B)  { benchSequential(b, logTextZerolog) }
func BenchmarkSlog_Text(b *testing.B)     { benchSequential(b, logTextSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// BigPayload：单字段 64KB
//
// 考察单条日志体量异常时的行为——一个 bug 或一次恶意输入就能让某个字段变成
// 几十上百 KB。这一组用的是关闭行长上限的 go-logger 基线装置，与其他三库
// 口径一致；上限开启后的实际表现见 bench_feature_test.go。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_BigPayload(b *testing.B) { benchSequential(b, logBigGoLogger) }
func BenchmarkZap_BigPayload(b *testing.B)      { benchSequential(b, logBigZap) }
func BenchmarkZerolog_BigPayload(b *testing.B)  { benchSequential(b, logBigZerolog) }
func BenchmarkSlog_BigPayload(b *testing.B)     { benchSequential(b, logBigSlog) }
