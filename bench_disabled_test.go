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
