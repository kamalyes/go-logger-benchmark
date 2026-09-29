/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 11:02:17
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 11:02:17
 * @FilePath: \go-logger-benchmark\bootstrap\report\main_test.go
 * @Description: 测试报告解析器单元测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBenchLineNormalizesGoBenchmarkName(t *testing.T) {
	line := "BenchmarkGoLogger_KV10-20  20865500  364.00 ns/op  0 B/op  0 allocs/op"

	name, nsPerOp, bPerOp, allocsOp, ok := parseBenchLine(line)

	if !ok {
		t.Fatal("expected benchmark line to parse")
	}
	if name != "BenchmarkGoLogger_KV10" {
		t.Fatalf("name = %q, want %q", name, "BenchmarkGoLogger_KV10")
	}
	if nsPerOp != 364.00 {
		t.Fatalf("nsPerOp = %v, want 364.00", nsPerOp)
	}
	if bPerOp != 0 {
		t.Fatalf("bPerOp = %d, want 0", bPerOp)
	}
	if allocsOp != 0 {
		t.Fatalf("allocsOp = %d, want 0", allocsOp)
	}
}

func TestParseBenchmarkFileUnifiedAggregatesNormalizedNames(t *testing.T) {
	content := `goos: windows
BenchmarkGoLogger_KV10-20  100  10 ns/op  0 B/op  0 allocs/op
BenchmarkGoLogger_KV10-16  100  20 ns/op  2 B/op  2 allocs/op
panic: later benchmark failed
`
	filename := filepath.Join(t.TempDir(), "benchmark_output.txt")
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	results := parseBenchmarkFileUnified(filename)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1: %#v", len(results), results)
	}
	got := results[0]
	if got.Name != "BenchmarkGoLogger_KV10" {
		t.Fatalf("Name = %q, want %q", got.Name, "BenchmarkGoLogger_KV10")
	}
	if got.NsPerOp != 15 {
		t.Fatalf("NsPerOp = %v, want 15", got.NsPerOp)
	}
	if got.BPerOp != 1 {
		t.Fatalf("BPerOp = %d, want 1", got.BPerOp)
	}
	if got.AllocsOp != 1 {
		t.Fatalf("AllocsOp = %d, want 1", got.AllocsOp)
	}
}

func TestSplitByLibrariesTrimsPrefixesAndAveragesMatchingSuffixes(t *testing.T) {
	all := []benchResult{
		{Name: "BenchmarkGoLogger_KV10", NsPerOp: 10, BPerOp: 0, AllocsOp: 0},
		{Name: "BenchmarkGoLogger_KV10", NsPerOp: 20, BPerOp: 2, AllocsOp: 2},
		{Name: "BenchmarkZap_KV10", NsPerOp: 30, BPerOp: 3, AllocsOp: 3},
		{Name: "BenchmarkZerolog_KV10", NsPerOp: 40, BPerOp: 4, AllocsOp: 4},
		{Name: "BenchmarkSlog_KV10", NsPerOp: 50, BPerOp: 5, AllocsOp: 5},
	}

	goLogger, zap, zerolog, slog := splitByLibraries(all)

	if len(goLogger) != 1 || len(zap) != 1 || len(zerolog) != 1 || len(slog) != 1 {
		t.Fatalf("split sizes = %d/%d/%d/%d, want 1/1/1/1",
			len(goLogger), len(zap), len(zerolog), len(slog))
	}
	if goLogger[0].Name != "KV10" || goLogger[0].NsPerOp != 15 || goLogger[0].BPerOp != 1 || goLogger[0].AllocsOp != 1 {
		t.Fatalf("goLogger[0] = %#v, want name KV10 avg 15/1/1", goLogger[0])
	}
	if zap[0].Name != "KV10" || zap[0].NsPerOp != 30 {
		t.Fatalf("zap[0] = %#v, want name KV10 ns 30", zap[0])
	}
	if zerolog[0].Name != "KV10" || zerolog[0].NsPerOp != 40 {
		t.Fatalf("zerolog[0] = %#v, want name KV10 ns 40", zerolog[0])
	}
	if slog[0].Name != "KV10" || slog[0].NsPerOp != 50 {
		t.Fatalf("slog[0] = %#v, want name KV10 ns 50", slog[0])
	}
}

func TestBuildComparisonsJoinsScenariosAndPicksWinner(t *testing.T) {
	goLogger := []benchResult{{Name: "KV10", NsPerOp: 364}, {Name: "KV10_Parallel", NsPerOp: 900}}
	zap := []benchResult{{Name: "KV10", NsPerOp: 861}, {Name: "KV10_Parallel", NsPerOp: 800}, {Name: "KV10_Sugared", NsPerOp: 1500}}
	zerolog := []benchResult{{Name: "KV10", NsPerOp: 312}, {Name: "KV10_Parallel", NsPerOp: 700}}
	slog := []benchResult{{Name: "KV10", NsPerOp: 242}, {Name: "KV10_Parallel", NsPerOp: 2000}}

	rows := buildComparisons(goLogger, zap, zerolog, slog)

	// KV10_Sugared 仅 zap 存在，四库联表必须排除该场景
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2: %#v", len(rows), rows)
	}
	if rows[0].Scenario != "KV10" || rows[0].Winner != "slog" {
		t.Fatalf("rows[0] = %#v, want scenario KV10 winner slog", rows[0])
	}
	if rows[1].Scenario != "KV10_Parallel" || rows[1].Winner != "zerolog" {
		t.Fatalf("rows[1] = %#v, want scenario KV10_Parallel winner zerolog", rows[1])
	}
}

func TestFastestLibraryTieKeepsFixedOrderDeterminism(t *testing.T) {
	ns := map[string]float64{"slog": 100, "zerolog": 100, "zap": 200, "go-logger": 300}

	if got := fastestLibrary(ns); got != "zerolog" {
		t.Fatalf("fastestLibrary = %q, want zerolog (libraryOrder 并列先到者胜)", got)
	}
}

func TestBuildSugaredVsCoreComparisonsPairsCoreWithSugared(t *testing.T) {
	zap := []benchResult{
		{Name: "KV10", NsPerOp: 861},
		{Name: "KV10_Sugared", NsPerOp: 1722},
		{Name: "MsgOnly", NsPerOp: 205},
	}

	rows := buildSugaredVsCoreComparisons(zap)

	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1: %#v", len(rows), rows)
	}
	got := rows[0]
	if got.Scenario != "KV10" {
		t.Fatalf("Scenario = %q, want KV10", got.Scenario)
	}
	if got.CoreNs != 861 || got.SugaredNs != 1722 {
		t.Fatalf("ns pair = %v/%v, want 861/1722", got.CoreNs, got.SugaredNs)
	}
	if got.Ratio != 2 {
		t.Fatalf("Ratio = %v, want 2", got.Ratio)
	}
	if got.Speedup != "Core faster" {
		t.Fatalf("Speedup = %q, want Core faster", got.Speedup)
	}
}

func TestFilterParallelSplitsSequentialAndParallelRows(t *testing.T) {
	comparisons := []comparisonRow{
		{Scenario: "KV10"},
		{Scenario: "KV10_Parallel"},
		{Scenario: "MsgOnly"},
		{Scenario: "MsgOnly_Parallel"},
	}
	allocs := []allocRow{
		{Scenario: "KV10"},
		{Scenario: "KV10_Parallel"},
	}

	if got := filterParallel(comparisons, false); len(got) != 2 || got[0].Scenario != "KV10" || got[1].Scenario != "MsgOnly" {
		t.Fatalf("sequential comparisons = %#v", got)
	}
	if got := filterParallel(comparisons, true); len(got) != 2 || got[0].Scenario != "KV10_Parallel" {
		t.Fatalf("parallel comparisons = %#v", got)
	}
	if got := filterAllocParallel(allocs, false); len(got) != 1 || got[0].Scenario != "KV10" {
		t.Fatalf("sequential allocs = %#v", got)
	}
	if got := filterAllocParallel(allocs, true); len(got) != 1 || got[0].Scenario != "KV10_Parallel" {
		t.Fatalf("parallel allocs = %#v", got)
	}
}
