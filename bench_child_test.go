/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-logger-benchmark\bench_child_test.go
 * @Description: go-logger vs 主流日志库性能基准测试 — 子 Logger（With）场景
 *
 * 每个场景按 go-logger → zap → zerolog → slog 紧挨排列，便于上下对比查阅
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import "testing"

// ══════════════════════════════════════════════════════════════════════════════
// 子 Logger：服务端最常见的两种用法，成本结构完全不同
//
//	Child       构造一次、打多条（把 request_id 等绑在子 Logger 上贯穿整个请求）
//	            考察常驻字段是否被预编码：预编码后每条只是一次 memcpy，
//	            否则每条都要重新遍历并编码一遍全部常驻字段
//
//	ChildBuild  每次重新构造子 Logger 并打一条（每请求一个子 Logger 的写法）
//	            考察 With 自身的构造成本，与上面恰好是一对相反的取舍
//
// 字段口径统一为 4 个 string（service / instance / region / env，即服务身份信息）
// ══════════════════════════════════════════════════════════════════════════════

// ─── Child：稳态（构造一次，打多条） ───

func BenchmarkGoLogger_Child(b *testing.B) { benchSequential(b, logChildGoLogger) }
func BenchmarkZap_Child(b *testing.B)      { benchSequential(b, logChildZap) }
func BenchmarkZerolog_Child(b *testing.B)  { benchSequential(b, logChildZerolog) }
func BenchmarkSlog_Child(b *testing.B)     { benchSequential(b, logChildSlog) }

// ─── Child Parallel ───

func BenchmarkGoLogger_Child_Parallel(b *testing.B) { benchParallel(b, logChildGoLogger) }
func BenchmarkZap_Child_Parallel(b *testing.B)      { benchParallel(b, logChildZap) }
func BenchmarkZerolog_Child_Parallel(b *testing.B)  { benchParallel(b, logChildZerolog) }
func BenchmarkSlog_Child_Parallel(b *testing.B)     { benchParallel(b, logChildSlog) }

// ─── ChildBuild：每次重新构造 ───

func BenchmarkGoLogger_ChildBuild(b *testing.B) { benchSequential(b, logChildBuildGoLogger) }
func BenchmarkZap_ChildBuild(b *testing.B)      { benchSequential(b, logChildBuildZap) }
func BenchmarkZerolog_ChildBuild(b *testing.B)  { benchSequential(b, logChildBuildZerolog) }
func BenchmarkSlog_ChildBuild(b *testing.B)     { benchSequential(b, logChildBuildSlog) }
