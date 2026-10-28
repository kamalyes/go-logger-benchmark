/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 09:52:36
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 09:52:36
 * @FilePath: \go-logger-benchmark\bench_disabled_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 级别短路场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// ══════════════════════════════════════════════════════════════════════════════
// Disabled：级别短路（WARN 阈值下打 Debug，考察各库的短路路径开销）
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_Disabled(b *testing.B) { benchSequential(b, logDisabledGoLogger) }
func BenchmarkZap_Disabled(b *testing.B)      { benchSequential(b, logDisabledZap) }
func BenchmarkZerolog_Disabled(b *testing.B)  { benchSequential(b, logDisabledZerolog) }
func BenchmarkSlog_Disabled(b *testing.B)     { benchSequential(b, logDisabledSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// Disabled_KV10：级别短路但带 5 对字段
//
// 上面的 Disabled 只带 1 对，四库看起来都"几乎免费"，于是很容易得出
// "把级别调高就没成本了"的结论。字段加到 5 对后结论要修正：参数在**调用点**
// 就构造完了，级别判断再早也省不掉，短路路径的耗时随字段数上涨
// （本机实测四库都涨 1.3~2.5 倍）。
//
// 更值得看的是分配。同样是"本该啥也不做"的路径，go-logger / zerolog / slog
// 都是 0 allocs/op，而 zap 的强类型 Field 即使级别不通过也照样构造并落堆
// （本机实测 320 B/op、1 alloc/op）——被关掉的 Debug 日志里只有它在持续产生垃圾。
//
// 注：本组最初的注释断言"变参装箱省不掉这 10 次堆分配"，实测推翻了它：
// 装箱确实发生，但常量与小整数不落堆、[]any 也未逃逸，分配是 0。
// 留此一笔，提醒这类成本结论必须测而不是推。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_Disabled_KV10(b *testing.B) { benchSequential(b, logDisabledKV10GoLogger) }
func BenchmarkZap_Disabled_KV10(b *testing.B)      { benchSequential(b, logDisabledKV10Zap) }
func BenchmarkZerolog_Disabled_KV10(b *testing.B)  { benchSequential(b, logDisabledKV10Zerolog) }
func BenchmarkSlog_Disabled_KV10(b *testing.B)     { benchSequential(b, logDisabledKV10Slog) }
