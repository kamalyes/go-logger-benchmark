/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-10-05 17:50:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-10-05 17:50:00
 * @FilePath: \go-logger-benchmark\bench_shape_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 调用形状场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// ══════════════════════════════════════════════════════════════════════════════
// KV2 / KV20：字段规模的两个端点
//
// 与已有的 KV10（5 对）合起来构成 1 / 5 / 10 对三个点。单点数据无法外推——
// 一条直线至少要两点，而"固定成本 + 每字段成本"这个结构也只有多点才能分离。
// 按自己的字段数估算时，用斜率比用某一个点准得多。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_KV2(b *testing.B) { benchSequential(b, logKV2GoLogger) }
func BenchmarkZap_KV2(b *testing.B)      { benchSequential(b, logKV2Zap) }
func BenchmarkZerolog_KV2(b *testing.B)  { benchSequential(b, logKV2Zerolog) }
func BenchmarkSlog_KV2(b *testing.B)     { benchSequential(b, logKV2Slog) }

func BenchmarkGoLogger_KV20(b *testing.B) { benchSequential(b, logKV20GoLogger) }
func BenchmarkZap_KV20(b *testing.B)      { benchSequential(b, logKV20Zap) }
func BenchmarkZerolog_KV20(b *testing.B)  { benchSequential(b, logKV20Zerolog) }
func BenchmarkSlog_KV20(b *testing.B)     { benchSequential(b, logKV20Slog) }

// ══════════════════════════════════════════════════════════════════════════════
// Context：trace_id 透传
//
// 服务端几乎每条日志都要带 trace_id，这是本表里最接近真实调用形状的一组。
// go-logger 的 *Context 系列自带提取；其余三库没有这一层，业务侧得自己写
// helper 再把字段显式挂上去（装置里的 traceIDFromCtx 就是那个 helper）。
//
// 口径不对等之处已在装置注释里写明：go-logger 的内置提取会先探一次 OTel
// span 再回落 ctx.Value，而 helper 只读一次 ctx.Value——本场景 go-logger
// 做的是严格更多的工作。抹平这点反而失真，故保留并在此标注。
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_Context(b *testing.B) { benchSequential(b, logCtxGoLogger) }
func BenchmarkZap_Context(b *testing.B)      { benchSequential(b, logCtxZap) }
func BenchmarkZerolog_Context(b *testing.B)  { benchSequential(b, logCtxZerolog) }
func BenchmarkSlog_Context(b *testing.B)     { benchSequential(b, logCtxSlog) }

// ─── Parallel ───
// trace_id 透传的真实现场是并发请求，顺序数字不足以代表。

func BenchmarkGoLogger_Context_Parallel(b *testing.B) { benchParallel(b, logCtxGoLogger) }
func BenchmarkZap_Context_Parallel(b *testing.B)      { benchParallel(b, logCtxZap) }
func BenchmarkZerolog_Context_Parallel(b *testing.B)  { benchParallel(b, logCtxZerolog) }
func BenchmarkSlog_Context_Parallel(b *testing.B)     { benchParallel(b, logCtxSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// EscapeHeavy：需转义的字段值
//
// 其余所有场景的值都是干净 ASCII，整段可以 memmove；而真实字段里嵌 JSON、
// 带换行、含中文都很常见，必须逐字节改写。两条路径的成本不可互推，而基准
// 只测前者的话，恰好测的是最省的那一档。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_EscapeHeavy(b *testing.B) { benchSequential(b, logEscapeGoLogger) }
func BenchmarkZap_EscapeHeavy(b *testing.B)      { benchSequential(b, logEscapeZap) }
func BenchmarkZerolog_EscapeHeavy(b *testing.B)  { benchSequential(b, logEscapeZerolog) }
func BenchmarkSlog_EscapeHeavy(b *testing.B)     { benchSequential(b, logEscapeSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// ObjectField：结构体字段
//
// TypedValues 覆盖的是 error / Duration——各库都有专门分支的类型。结构体
// 没有任何库有专用分支，只能走 reflect 或 encoding/json，成本比原始类型高
// 一到两个数量级。把 DTO 整个丢进日志是日常写法，这一组量化它的代价。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_ObjectField(b *testing.B) { benchSequential(b, logObjGoLogger) }
func BenchmarkZap_ObjectField(b *testing.B)      { benchSequential(b, logObjZap) }
func BenchmarkZerolog_ObjectField(b *testing.B)  { benchSequential(b, logObjZerolog) }
func BenchmarkSlog_ObjectField(b *testing.B)     { benchSequential(b, logObjSlog) }
