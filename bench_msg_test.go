/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 09:36:16
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 09:36:16
 * @FilePath: \go-logger-benchmark\bench_msg_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 纯消息场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// ══════════════════════════════════════════════════════════════════════════════
// MsgOnly：纯消息（无字段）
// ══════════════════════════════════════════════════════════════════════════════

// ─── Sequential ───

func BenchmarkGoLogger_MsgOnly(b *testing.B) { benchSequential(b, logMsgGoLogger) }
func BenchmarkZap_MsgOnly(b *testing.B)      { benchSequential(b, logMsgZap) }
func BenchmarkZerolog_MsgOnly(b *testing.B)  { benchSequential(b, logMsgZerolog) }
func BenchmarkSlog_MsgOnly(b *testing.B)     { benchSequential(b, logMsgSlog) }

// ─── Parallel ───

func BenchmarkGoLogger_MsgOnly_Parallel(b *testing.B) { benchParallel(b, logMsgGoLogger) }
func BenchmarkZap_MsgOnly_Parallel(b *testing.B)      { benchParallel(b, logMsgZap) }
func BenchmarkZerolog_MsgOnly_Parallel(b *testing.B)  { benchParallel(b, logMsgZerolog) }
func BenchmarkSlog_MsgOnly_Parallel(b *testing.B)     { benchParallel(b, logMsgSlog) }

// ══════════════════════════════════════════════════════════════════════════════
// MsgOnly_Caller：caller 的固定成本
//
// KV10_Caller 把 caller 成本与 5 对字段的编码成本混在一起，看不出 caller
// 自己值多少。挂到最省的纯消息场景上，runtime 栈回溯的固定开销才单独显形
// ——与同组的 MsgOnly 相减即是 caller 的净成本。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkGoLogger_MsgOnly_Caller(b *testing.B) { benchSequential(b, logMsgGoLoggerCaller) }
func BenchmarkZap_MsgOnly_Caller(b *testing.B)      { benchSequential(b, logMsgZapCaller) }
func BenchmarkZerolog_MsgOnly_Caller(b *testing.B)  { benchSequential(b, logMsgZerologCaller) }
func BenchmarkSlog_MsgOnly_Caller(b *testing.B)     { benchSequential(b, logMsgSlogCaller) }
