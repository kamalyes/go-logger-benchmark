<div align="center">

# go-logger-benchmark

**[go-logger] 与 [zap]、[zerolog]、[slog] 的性能对比**

[go-logger]: https://github.com/kamalyes/go-logger
[zap]: https://github.com/uber-go/zap
[zerolog]: https://github.com/rs/zerolog
[slog]: https://pkg.go.dev/log/slog

[![Benchmark](https://github.com/kamalyes/go-logger-benchmark/actions/workflows/benchmark.yml/badge.svg)](https://github.com/kamalyes/go-logger-benchmark/actions/workflows/benchmark.yml)

[English](./README.md)

</div>

---

## 📊 性能对比图表

### 延迟 — 串行

![Latency Sequential](./benchmarks/latency.svg)

### 延迟 — 并行

![Latency Parallel](./benchmarks/latency_parallel.svg)

### 内存分配 — 串行

![Allocs Sequential](./benchmarks/allocs.svg)

### Zap Core vs Sugared — 强类型字段收益

![Zap Core vs Sugared](./benchmarks/zap_core_vs_sugared.svg)

> 图表由 GitHub Actions 在每次推送时自动生成。

---

## 🧪 测试场景

所有场景统一口径：JSON 格式 + Info 级别 + `io.Discard` 输出，各库按各自惯用 API 记录等价日志。

| 场景 | 说明 |
|------|------|
| KV10 | 5 对（10 个）混合类型字段：3 个 string + 2 个 int |
| KV10_Caller | KV10 基础上开启 caller（输出源码位置） |
| MsgOnly | 纯消息（无字段） |
| Disabled | 级别短路：WARN 阈值下打 Debug |

每个场景均覆盖串行与并行（`-Parallel` 后缀），另含 Zap Sugared 对照组用于度量强类型 Field API 的收益。

---

## 🚀 快速开始

```bash
# 运行所有基准测试
go test -run='^$' -bench=. -benchmem -count=3 -timeout=30m ./... | tee benchmark_output.txt

# 生成图表和报告
go run ./bootstrap/report

# 解析真实基准测试输出
go test -run='^$' -bench=. -benchmem -count=1 -timeout=10m ./... > benchmark_output.txt
go run ./bootstrap/report -parse benchmark_output.txt
```

---

## 📁 项目结构

```bash
go-logger-benchmark/
├── bench_kv_test.go               # KV 字段场景基准测试
├── bench_msg_test.go              # 纯消息场景基准测试
├── bench_disabled_test.go         # 级别短路场景基准测试
├── loggers.go                     # 四库对齐的测试装置
├── bootstrap/
│   └── report/
│       ├── main.go                # 报告和 SVG 图表生成器
│       └── main_test.go           # 报告解析器单元测试
├── benchmarks/
│   ├── latency.svg                # 串行延迟图
│   ├── latency_parallel.svg       # 并行延迟图
│   ├── allocs.svg                 # 内存分配图
│   ├── zap_core_vs_sugared.svg    # Zap Core vs Sugared 图
│   └── *.json                     # 原始基准数据
├── .github/workflows/
│   └── benchmark.yml              # CI 自动基准测试工作流
├── BENCHMARKS.md                  # 详细数据表格
└── go.mod
```

---

## ⚙️ CI 自动化

[benchmark 工作流](./.github/workflows/benchmark.yml) 自动运行：

1. 每次推送到 `master` 时（仅代码变更）
2. [go-logger] 发布新 tag 时通过 `upstream-updated` 事件触发

每次运行以 3 次迭代对指定 go-logger ref 执行 `go test -bench`，生成 SVG 图表并以单一 amend 提交回仓库。

---

## 📋 完整数据

详见 [BENCHMARKS.md](./BENCHMARKS.md)，包含所有场景的详细对比数据表格。

---

## 📝 许可证

本项目采用 MIT 许可证。
