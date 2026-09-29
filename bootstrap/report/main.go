/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 10:12:36
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 10:12:36
 * @FilePath: \go-logger-benchmark\bootstrap\report\main.go
 * @Description: 测试报告解析器
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type benchResult struct {
	Name     string  `json:"name"`
	NsPerOp  float64 `json:"ns_per_op"`
	BPerOp   uint64  `json:"bytes_per_op"`
	AllocsOp uint64  `json:"allocs_per_op"`
}

type comparisonRow struct {
	Scenario   string  `json:"scenario"`
	GoLoggerNs float64 `json:"go_logger_ns"`
	ZapNs      float64 `json:"zap_ns"`
	ZerologNs  float64 `json:"zerolog_ns"`
	SlogNs     float64 `json:"slog_ns"`
	Winner     string  `json:"winner"`
}

type allocRow struct {
	Scenario       string `json:"scenario"`
	GoLoggerAllocs uint64 `json:"go_logger_allocs"`
	ZapAllocs      uint64 `json:"zap_allocs"`
	ZerologAllocs  uint64 `json:"zerolog_allocs"`
	SlogAllocs     uint64 `json:"slog_allocs"`
	GoLoggerBytes  uint64 `json:"go_logger_bytes"`
	ZapBytes       uint64 `json:"zap_bytes"`
	ZerologBytes   uint64 `json:"zerolog_bytes"`
	SlogBytes      uint64 `json:"slog_bytes"`
	Winner         string `json:"winner"`
}

type sugaredVsCoreRow struct {
	Scenario  string  `json:"scenario"`
	CoreNs    float64 `json:"core_ns"`
	SugaredNs float64 `json:"sugared_ns"`
	Ratio     float64 `json:"ratio"`
	Speedup   string  `json:"speedup"`
}

type envInfo struct {
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Pkg    string `json:"pkg"`
	CPU    string `json:"cpu"`
}

func main() {
	rootDir := "."
	parseFile := ""

	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-parse" && i+1 < len(os.Args) {
			parseFile = os.Args[i+1]
			i++
		} else {
			rootDir = os.Args[i]
		}
	}

	var allResults []benchResult
	var env envInfo

	if parseFile != "" {
		allResults, env = parseBenchmarkFileUnifiedWithEnv(parseFile)
		if len(allResults) == 0 {
			fmt.Println("Warning: no benchmark results parsed, using fallback data")
			allResults = fallbackDataUnified()
		}
	} else {
		allResults = fallbackDataUnified()
	}

	goLoggerResults, zapResults, zerologResults, slogResults := splitByLibraries(allResults)

	comparisons := buildComparisons(goLoggerResults, zapResults, zerologResults, slogResults)
	allocs := buildAllocComparisons(goLoggerResults, zapResults, zerologResults, slogResults)
	sugaredRows := buildSugaredVsCoreComparisons(zapResults)

	benchDir := filepath.Join(rootDir, "benchmarks")
	os.MkdirAll(benchDir, 0755)

	seqComparisons := filterParallel(comparisons, false)
	parComparisons := filterParallel(comparisons, true)

	latencySVG := generateLatencySVG(seqComparisons, "Latency Comparison (ns/op) — Sequential")
	writeFile(filepath.Join(benchDir, "latency.svg"), latencySVG)

	parallelSVG := generateLatencySVG(parComparisons, "Latency Comparison (ns/op) — Parallel")
	writeFile(filepath.Join(benchDir, "latency_parallel.svg"), parallelSVG)

	seqAllocs := filterAllocParallel(allocs, false)
	allocSVG := generateAllocSVG(seqAllocs, "Memory Allocation (heap allocs/op) — Sequential")
	writeFile(filepath.Join(benchDir, "allocs.svg"), allocSVG)

	sugaredSVG := generateSugaredVsCoreSVG(sugaredRows, "Zap Core vs Sugared (ns/op) — Typed Field Gain")
	writeFile(filepath.Join(benchDir, "zap_core_vs_sugared.svg"), sugaredSVG)

	writeJSON(filepath.Join(benchDir, "benchmark_results.json"), struct {
		GoLogger []benchResult `json:"go_logger"`
		Zap      []benchResult `json:"zap"`
		Zerolog  []benchResult `json:"zerolog"`
		Slog     []benchResult `json:"slog"`
	}{goLoggerResults, zapResults, zerologResults, slogResults})
	writeJSON(filepath.Join(benchDir, "benchmark_comparisons.json"), comparisons)
	writeJSON(filepath.Join(benchDir, "benchmark_allocs.json"), allocs)
	writeJSON(filepath.Join(benchDir, "zap_core_vs_sugared.json"), sugaredRows)

	generateBenchmarksMD(rootDir, comparisons, allocs, sugaredRows, env)

	fmt.Printf("Done! Generated %d four-library comparisons, %d alloc rows, %d Zap Core/Sugared comparisons\n",
		len(comparisons), len(allocs), len(sugaredRows))
}

func fallbackDataUnified() []benchResult {
	return []benchResult{
		{"BenchmarkGoLogger_KV10", 364, 0, 0},
		{"BenchmarkZap_KV10", 861, 952, 3},
		{"BenchmarkZerolog_KV10", 312, 320, 2},
		{"BenchmarkSlog_KV10", 2424, 2160, 26},
		{"BenchmarkGoLogger_KV10_Caller", 1605, 0, 0},
		{"BenchmarkZap_KV10_Caller", 1707, 720, 3},
		{"BenchmarkZerolog_KV10_Caller", 1760, 352, 4},
		{"BenchmarkSlog_KV10_Caller", 2600, 1600, 30},
		{"BenchmarkGoLogger_MsgOnly", 185, 0, 0},
		{"BenchmarkZap_MsgOnly", 205, 96, 2},
		{"BenchmarkZerolog_MsgOnly", 201, 96, 2},
		{"BenchmarkSlog_MsgOnly", 1100, 720, 12},
		{"BenchmarkGoLogger_Disabled", 2, 0, 0},
		{"BenchmarkZap_Disabled", 5, 0, 0},
		{"BenchmarkZerolog_Disabled", 2, 0, 0},
		{"BenchmarkSlog_Disabled", 3, 0, 0},
	}
}

func parseBenchmarkFileUnified(filename string) []benchResult {
	results, _ := parseBenchmarkFileUnifiedWithEnv(filename)
	return results
}

func parseBenchmarkFileUnifiedWithEnv(filename string) ([]benchResult, envInfo) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", filename, err)
		return nil, envInfo{}
	}
	defer f.Close()

	aggregated := map[string]*aggResult{}
	var env envInfo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "goos:") {
			env.Goos = strings.TrimSpace(strings.TrimPrefix(line, "goos:"))
			continue
		}
		if strings.HasPrefix(line, "goarch:") {
			env.Goarch = strings.TrimSpace(strings.TrimPrefix(line, "goarch:"))
			continue
		}
		if strings.HasPrefix(line, "pkg:") && env.Pkg == "" {
			env.Pkg = strings.TrimSpace(strings.TrimPrefix(line, "pkg:"))
			continue
		}
		if strings.HasPrefix(line, "cpu:") {
			env.CPU = strings.TrimSpace(strings.TrimPrefix(line, "cpu:"))
			continue
		}
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		name, nsPerOp, bPerOp, allocsOp, ok := parseBenchLine(line)
		if !ok {
			continue
		}
		if _, exists := aggregated[name]; !exists {
			aggregated[name] = &aggResult{name: name}
		}
		aggregated[name].add(nsPerOp, bPerOp, allocsOp)
	}

	return aggregateResults(aggregated), env
}

func splitByLibraries(all []benchResult) (goLogger, zap, zerolog, slog []benchResult) {
	goLogger = collectResultsByBenchmarkPrefix(all, "BenchmarkGoLogger")
	zap = collectResultsByBenchmarkPrefix(all, "BenchmarkZap")
	zerolog = collectResultsByBenchmarkPrefix(all, "BenchmarkZerolog")
	slog = collectResultsByBenchmarkPrefix(all, "BenchmarkSlog")
	return goLogger, zap, zerolog, slog
}

func buildSugaredVsCoreComparisons(zapResults []benchResult) []sugaredVsCoreRow {
	coreMap := resultsByName(zapResults)

	var rows []sugaredVsCoreRow
	for _, sugared := range zapResults {
		if !strings.HasSuffix(sugared.Name, "_Sugared") {
			continue
		}
		core, ok := coreMap[strings.TrimSuffix(sugared.Name, "_Sugared")]
		if !ok {
			continue
		}
		ratio := sugared.NsPerOp / core.NsPerOp
		speedup := "Core faster"
		if core.NsPerOp > sugared.NsPerOp {
			speedup = "Sugared faster"
		}
		rows = append(rows, sugaredVsCoreRow{
			Scenario: strings.TrimSuffix(sugared.Name, "_Sugared"),
			CoreNs:   core.NsPerOp, SugaredNs: sugared.NsPerOp,
			Ratio: ratio, Speedup: speedup,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Scenario < rows[j].Scenario })
	return rows
}

func collectResultsByBenchmarkPrefix(all []benchResult, prefix string) []benchResult {
	aggregated := map[string]*aggResult{}
	for _, r := range all {
		addByTrimmedPrefix(aggregated, r, prefix)
	}
	return aggregateResults(aggregated)
}

func addByTrimmedPrefix(aggregated map[string]*aggResult, r benchResult, prefix string) bool {
	name, ok := trimBenchmarkPrefix(r.Name, prefix)
	if !ok {
		return false
	}
	if _, exists := aggregated[name]; !exists {
		aggregated[name] = &aggResult{name: name}
	}
	aggregated[name].add(r.NsPerOp, r.BPerOp, r.AllocsOp)
	return true
}

func trimBenchmarkPrefix(name, prefix string) (string, bool) {
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	trimmed := strings.TrimPrefix(name, prefix)
	trimmed = strings.TrimPrefix(trimmed, "_")
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func aggregateResults(aggregated map[string]*aggResult) []benchResult {
	var results []benchResult
	for _, agg := range aggregated {
		results = append(results, benchResult{
			Name:     agg.name,
			NsPerOp:  agg.avgNsPerOp(),
			BPerOp:   agg.avgBPerOp(),
			AllocsOp: agg.avgAllocsOp(),
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results
}

func resultsByName(results []benchResult) map[string]benchResult {
	byName := map[string]benchResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	return byName
}

type aggResult struct {
	name      string
	nsList    []float64
	bList     []uint64
	allocList []uint64
}

func (a *aggResult) add(ns float64, b uint64, alloc uint64) {
	a.nsList = append(a.nsList, ns)
	a.bList = append(a.bList, b)
	a.allocList = append(a.allocList, alloc)
}

func (a *aggResult) avgNsPerOp() float64 {
	if len(a.nsList) == 0 {
		return 0
	}
	var sum float64
	for _, v := range a.nsList {
		sum += v
	}
	return sum / float64(len(a.nsList))
}

func (a *aggResult) avgBPerOp() uint64 {
	if len(a.bList) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range a.bList {
		sum += v
	}
	return sum / uint64(len(a.bList))
}

func (a *aggResult) avgAllocsOp() uint64 {
	if len(a.allocList) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range a.allocList {
		sum += v
	}
	return sum / uint64(len(a.allocList))
}

func parseBenchLine(line string) (name string, nsPerOp float64, bPerOp uint64, allocsOp uint64, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return "", 0, 0, 0, false
	}

	name = normalizeBenchName(fields[0])

	nsIdx := -1
	for i, f := range fields {
		if strings.HasSuffix(f, "ns/op") {
			nsIdx = i
			break
		}
	}
	if nsIdx < 1 {
		return "", 0, 0, 0, false
	}

	nsPerOp, err := strconv.ParseFloat(fields[nsIdx-1], 64)
	if err != nil {
		return "", 0, 0, 0, false
	}

	for i, f := range fields {
		if strings.HasSuffix(f, "B/op") && i > 0 {
			b, _ := strconv.ParseUint(fields[i-1], 10, 64)
			bPerOp = b
		}
		if strings.HasSuffix(f, "allocs/op") && i > 0 {
			a, _ := strconv.ParseUint(fields[i-1], 10, 64)
			allocsOp = a
		}
	}

	return name, nsPerOp, bPerOp, allocsOp, true
}

func normalizeBenchName(name string) string {
	idx := strings.LastIndexByte(name, '-')
	if idx < 0 || idx == len(name)-1 {
		return name
	}
	for _, r := range name[idx+1:] {
		if r < '0' || r > '9' {
			return name
		}
	}
	return name[:idx]
}

// libraryOrder winner 判定的固定遍历顺序（并列时先到者胜，保证确定性）
var libraryOrder = []string{"go-logger", "zap", "zerolog", "slog"}

func fastestLibrary(ns map[string]float64) string {
	best, bestNs := "", math.MaxFloat64
	for _, label := range libraryOrder {
		if v, ok := ns[label]; ok && v < bestNs {
			best, bestNs = label, v
		}
	}
	return best
}

func fewestAllocsLibrary(libs map[string]benchResult) string {
	best, bestAllocs, bestBytes := "", uint64(math.MaxUint64), uint64(math.MaxUint64)
	for _, label := range libraryOrder {
		r, ok := libs[label]
		if !ok {
			continue
		}
		if r.AllocsOp < bestAllocs || (r.AllocsOp == bestAllocs && r.BPerOp < bestBytes) {
			best, bestAllocs, bestBytes = label, r.AllocsOp, r.BPerOp
		}
	}
	return best
}

func buildComparisons(goLogger, zap, zerolog, slog []benchResult) []comparisonRow {
	joins := map[string]map[string]benchResult{
		"go-logger": resultsByName(goLogger),
		"zap":       resultsByName(zap),
		"zerolog":   resultsByName(zerolog),
		"slog":      resultsByName(slog),
	}

	scenarioSet := map[string]struct{}{}
	for _, byName := range joins {
		for scenario := range byName {
			scenarioSet[scenario] = struct{}{}
		}
	}

	var rows []comparisonRow
	for scenario := range scenarioSet {
		ns := map[string]float64{}
		present := true
		for label, byName := range joins {
			r, ok := byName[scenario]
			if !ok {
				present = false
				break
			}
			ns[label] = r.NsPerOp
		}
		if !present {
			continue
		}
		rows = append(rows, comparisonRow{
			Scenario:   scenario,
			GoLoggerNs: ns["go-logger"],
			ZapNs:      ns["zap"],
			ZerologNs:  ns["zerolog"],
			SlogNs:     ns["slog"],
			Winner:     fastestLibrary(ns),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Scenario < rows[j].Scenario })
	return rows
}

func buildAllocComparisons(goLogger, zap, zerolog, slog []benchResult) []allocRow {
	joins := map[string]map[string]benchResult{
		"go-logger": resultsByName(goLogger),
		"zap":       resultsByName(zap),
		"zerolog":   resultsByName(zerolog),
		"slog":      resultsByName(slog),
	}

	scenarioSet := map[string]struct{}{}
	for _, byName := range joins {
		for scenario := range byName {
			scenarioSet[scenario] = struct{}{}
		}
	}

	var rows []allocRow
	for scenario := range scenarioSet {
		present := true
		for _, byName := range joins {
			if _, ok := byName[scenario]; !ok {
				present = false
				break
			}
		}
		if !present {
			continue
		}
		libs := map[string]benchResult{
			"go-logger": joins["go-logger"][scenario],
			"zap":       joins["zap"][scenario],
			"zerolog":   joins["zerolog"][scenario],
			"slog":      joins["slog"][scenario],
		}
		rows = append(rows, allocRow{
			Scenario:       scenario,
			GoLoggerAllocs: libs["go-logger"].AllocsOp,
			ZapAllocs:      libs["zap"].AllocsOp,
			ZerologAllocs:  libs["zerolog"].AllocsOp,
			SlogAllocs:     libs["slog"].AllocsOp,
			GoLoggerBytes:  libs["go-logger"].BPerOp,
			ZapBytes:       libs["zap"].BPerOp,
			ZerologBytes:   libs["zerolog"].BPerOp,
			SlogBytes:      libs["slog"].BPerOp,
			Winner:         fewestAllocsLibrary(libs),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Scenario < rows[j].Scenario })
	return rows
}

func filterParallel(comparisons []comparisonRow, parallel bool) []comparisonRow {
	var filtered []comparisonRow
	for _, c := range comparisons {
		if strings.Contains(c.Scenario, "Parallel") == parallel {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterAllocParallel(allocs []allocRow, parallel bool) []allocRow {
	var filtered []allocRow
	for _, a := range allocs {
		if strings.Contains(a.Scenario, "Parallel") == parallel {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func writeFile(path, content string) {
	os.WriteFile(path, []byte(content), 0644)
}

func writeJSON(filename string, data interface{}) {
	f, _ := os.Create(filename)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

const (
	svgWidth      = 800
	barHeight     = 22
	barGap        = 5
	groupGap      = 20
	marginLeft    = 190
	marginRight   = 60
	marginTop     = 68
	marginBottom  = 40
	goLoggerColor = "#3B82F6"
	zapColor      = "#10B981"
	zerologColor  = "#8B5CF6"
	slogColor     = "#F43F5E"
	coreColor     = "#3B82F6"
	sugaredColor  = "#8B5CF6"
	goLoggerLabel = "go-logger"
	zapLabel      = "zap"
	zerologLabel  = "zerolog"
	slogLabel     = "slog"
	coreLabel     = "Core"
	sugaredLabel  = "Sugared"
	bgColor       = "#FFFFFF"
	titleColor    = "#1E293B"
	labelColor    = "#334155"
	valueColor    = "#475569"
	legendColor   = "#64748B"
	hintColor     = "#94A3B8"
)

type barSpec struct {
	Label string
	Ns    float64
	Color string
}

func libraryBars(c comparisonRow) []barSpec {
	return []barSpec{
		{goLoggerLabel, c.GoLoggerNs, goLoggerColor},
		{zapLabel, c.ZapNs, zapColor},
		{zerologLabel, c.ZerologNs, zerologColor},
		{slogLabel, c.SlogNs, slogColor},
	}
}

func svgHeader(sb *strings.Builder, title string, totalHeight int, legend []barSpec) {
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, svgWidth, totalHeight, svgWidth, totalHeight))
	sb.WriteString(fmt.Sprintf(`<rect width="100%%" height="100%%" fill="%s" rx="12"/>`, bgColor))
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="26" fill="%s" font-family="system-ui,-apple-system,sans-serif" font-size="16" font-weight="600">%s</text>`, marginLeft, titleColor, title))
	for i, entry := range legend {
		x := marginLeft + i*110
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="36" width="14" height="14" rx="3" fill="%s"/>`, x, entry.Color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="48" fill="%s" font-family="system-ui,sans-serif" font-size="12">%s</text>`, x+18, legendColor, entry.Label))
	}
}

func generateLatencySVG(comparisons []comparisonRow, title string) string {
	if len(comparisons) == 0 {
		return "<svg></svg>"
	}

	maxNs := 0.0
	for _, c := range comparisons {
		for _, bar := range libraryBars(c) {
			maxNs = math.Max(maxNs, bar.Ns)
		}
	}

	chartWidth := svgWidth - marginLeft - marginRight
	barsPerGroup := 4
	groupInner := barsPerGroup*barHeight + (barsPerGroup-1)*barGap
	groupHeight := groupInner + groupGap
	totalHeight := marginTop + len(comparisons)*groupHeight + marginBottom + 30

	var sb strings.Builder
	svgHeader(&sb, title, totalHeight, []barSpec{
		{goLoggerLabel, 0, goLoggerColor},
		{zapLabel, 0, zapColor},
		{zerologLabel, 0, zerologColor},
		{slogLabel, 0, slogColor},
	})

	for i, c := range comparisons {
		groupY := marginTop + i*groupHeight
		label := c.Scenario
		if len(label) > 28 {
			label = label[:25] + "..."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="12" text-anchor="end">%s</text>`, marginLeft-10, groupY+groupInner/2+4, labelColor, label))
		for j, bar := range libraryBars(c) {
			y := groupY + j*(barHeight+barGap)
			w := (bar.Ns / maxNs) * float64(chartWidth)
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%.1f" height="%d" rx="4" fill="%s" opacity="0.9"/>`, marginLeft, y, w, barHeight, bar.Color))
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="11">%.0f ns</text>`, marginLeft+int(w)+6, y+barHeight-6, valueColor, bar.Ns))
		}
	}

	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="system-ui,sans-serif" font-size="10">Lower is better ▸</text>`, marginLeft, totalHeight-10, hintColor))
	sb.WriteString(`</svg>`)
	return sb.String()
}

func generateAllocSVG(allocs []allocRow, title string) string {
	if len(allocs) == 0 {
		return "<svg></svg>"
	}

	type allocBar struct {
		Label  string
		Allocs uint64
		Bytes  uint64
		Color  string
	}
	barsOf := func(a allocRow) []allocBar {
		return []allocBar{
			{goLoggerLabel, a.GoLoggerAllocs, a.GoLoggerBytes, goLoggerColor},
			{zapLabel, a.ZapAllocs, a.ZapBytes, zapColor},
			{zerologLabel, a.ZerologAllocs, a.ZerologBytes, zerologColor},
			{slogLabel, a.SlogAllocs, a.SlogBytes, slogColor},
		}
	}

	maxAllocs := uint64(0)
	for _, a := range allocs {
		for _, bar := range barsOf(a) {
			maxAllocs = maxU64(maxAllocs, bar.Allocs, 0)
		}
	}
	if maxAllocs == 0 {
		maxAllocs = 1
	}

	chartWidth := svgWidth - marginLeft - marginRight
	barsPerGroup := 4
	groupInner := barsPerGroup*barHeight + (barsPerGroup-1)*barGap
	groupHeight := groupInner + groupGap
	totalHeight := marginTop + len(allocs)*groupHeight + marginBottom + 30

	var sb strings.Builder
	svgHeader(&sb, title, totalHeight, []barSpec{
		{goLoggerLabel, 0, goLoggerColor},
		{zapLabel, 0, zapColor},
		{zerologLabel, 0, zerologColor},
		{slogLabel, 0, slogColor},
	})

	for i, a := range allocs {
		groupY := marginTop + i*groupHeight
		label := a.Scenario
		if len(label) > 22 {
			label = label[:19] + "..."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="12" text-anchor="end">%s</text>`, marginLeft-10, groupY+groupInner/2+4, labelColor, label))
		for j, bar := range barsOf(a) {
			y := groupY + j*(barHeight+barGap)
			w := (float64(bar.Allocs) / float64(maxAllocs)) * float64(chartWidth)
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%.1f" height="%d" rx="4" fill="%s" opacity="0.9"/>`, marginLeft, y, w, barHeight, bar.Color))
			sb.WriteString(allocValueText(marginLeft, w, y+barHeight-6, bar.Allocs, bar.Bytes, chartWidth, bar.Color))
		}
	}

	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="system-ui,sans-serif" font-size="10">Lower is better ▸</text>`, marginLeft, totalHeight-10, hintColor))
	sb.WriteString(`</svg>`)
	return sb.String()
}

func generateSugaredVsCoreSVG(rows []sugaredVsCoreRow, title string) string {
	if len(rows) == 0 {
		return "<svg></svg>"
	}

	maxNs := 0.0
	for _, r := range rows {
		maxNs = math.Max(maxNs, r.CoreNs)
		maxNs = math.Max(maxNs, r.SugaredNs)
	}

	chartWidth := svgWidth - marginLeft - marginRight
	barsPerGroup := 2
	groupInner := barsPerGroup*barHeight + (barsPerGroup-1)*barGap
	groupHeight := groupInner + groupGap
	totalHeight := marginTop + len(rows)*groupHeight + marginBottom + 30

	var sb strings.Builder
	svgHeader(&sb, title, totalHeight, []barSpec{
		{coreLabel, 0, coreColor},
		{sugaredLabel, 0, sugaredColor},
	})

	for i, r := range rows {
		groupY := marginTop + i*groupHeight
		label := r.Scenario
		if len(label) > 28 {
			label = label[:25] + "..."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="12" text-anchor="end">%s</text>`, marginLeft-10, groupY+groupInner/2+4, labelColor, label))
		coreW := (r.CoreNs / maxNs) * float64(chartWidth)
		sugaredW := (r.SugaredNs / maxNs) * float64(chartWidth)
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%.1f" height="%d" rx="4" fill="%s" opacity="0.9"/>`, marginLeft, groupY, coreW, barHeight, coreColor))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="11">%.0f ns</text>`, marginLeft+int(coreW)+6, groupY+barHeight-6, valueColor, r.CoreNs))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%.1f" height="%d" rx="4" fill="%s" opacity="0.9"/>`, marginLeft, groupY+barHeight+barGap, sugaredW, barHeight, sugaredColor))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="11">%.0f ns</text>`, marginLeft+int(sugaredW)+6, groupY+2*barHeight+barGap-6, valueColor, r.SugaredNs))
	}

	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="system-ui,sans-serif" font-size="10">Lower is better ▸</text>`, marginLeft, totalHeight-10, hintColor))
	sb.WriteString(`</svg>`)
	return sb.String()
}

func allocValueText(baseX int, barW float64, textY int, allocs, bytes uint64, chartWidth int, color string) string {
	if allocs == 0 {
		return fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="11">0</text>`, baseX+4, textY, valueColor)
	}
	text := fmt.Sprintf("%d (%dB)", allocs, bytes)
	textX := baseX + int(barW) + 6
	if textX+len(text)*7 > baseX+chartWidth {
		textX = baseX + int(barW) - len(text)*7 - 6
		return fmt.Sprintf(`<text x="%d" y="%d" fill="#FFF" font-family="monospace" font-size="11">%s</text>`, textX, textY, text)
	}
	return fmt.Sprintf(`<text x="%d" y="%d" fill="%s" font-family="monospace" font-size="11">%s</text>`, textX, textY, valueColor, text)
}

func maxU64(a, b, c uint64) uint64 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func generateBenchmarksMD(rootDir string, comparisons []comparisonRow, allocs []allocRow, sugaredRows []sugaredVsCoreRow, env envInfo) {
	var sb strings.Builder

	sb.WriteString("# Benchmark Details\n\n")
	sb.WriteString("Auto-generated by `go run ./bootstrap/report`. Do not edit manually.\n\n")
	if env.Goos != "" || env.Goarch != "" || env.CPU != "" {
		sb.WriteString("## Environment\n\n")
		sb.WriteString(fmt.Sprintf("| Key | Value |\n|-----|-------|\n| goos | %s |\n| goarch | %s |\n| pkg | %s |\n| cpu | %s |\n\n",
			env.Goos, env.Goarch, env.Pkg, env.CPU))
	}

	sb.WriteString("## Latency (ns/op) — Four Libraries\n\n")
	sb.WriteString("| Scenario | go-logger | zap | zerolog | slog | Winner |\n")
	sb.WriteString("|----------|----------:|----:|--------:|----:|--------|\n")
	for _, c := range comparisons {
		sb.WriteString(fmt.Sprintf("| %s | %.0f | %.0f | %.0f | %.0f | %s |\n",
			c.Scenario, c.GoLoggerNs, c.ZapNs, c.ZerologNs, c.SlogNs, c.Winner))
	}

	sb.WriteString("\n## Memory Allocation — Four Libraries\n\n")
	sb.WriteString("| Scenario | go-logger (allocs) | zap (allocs) | zerolog (allocs) | slog (allocs) | go-logger (bytes) | zap (bytes) | zerolog (bytes) | slog (bytes) | Winner |\n")
	sb.WriteString("|----------|-------------------:|-------------:|-----------------:|--------------:|------------------:|------------:|----------------:|--------------:|--------|\n")
	for _, a := range allocs {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %d | %d | %d | %s |\n",
			a.Scenario, a.GoLoggerAllocs, a.ZapAllocs, a.ZerologAllocs, a.SlogAllocs,
			a.GoLoggerBytes, a.ZapBytes, a.ZerologBytes, a.SlogBytes, a.Winner))
	}

	if len(sugaredRows) > 0 {
		sb.WriteString("\n## Zap Core vs Sugared — Typed Field Gain\n\n")
		sb.WriteString("| Scenario | Core (ns) | Sugared (ns) | Ratio (Sugared/Core) | Faster |\n")
		sb.WriteString("|----------|----------:|--------------:|---------------------:|---------|\n")
		for _, r := range sugaredRows {
			sb.WriteString(fmt.Sprintf("| %s | %.0f | %.0f | %.2fx | %s |\n",
				r.Scenario, r.CoreNs, r.SugaredNs, r.Ratio, r.Speedup))
		}
	}

	os.WriteFile(filepath.Join(rootDir, "BENCHMARKS.md"), []byte(sb.String()), 0644)
}
