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

格式化类场景统一口径：JSON 格式 + Info 级别 + `io.Discard` 输出，各库按各自惯用 API 记录等价日志。
`Sink*` 场景例外——它们必须写真实文件才有意义（见下）。

| 场景 | 说明 |
|------|------|
| KV10 | 5 对（10 个）混合类型字段：3 个 string + 2 个 int |
| **KV2** / **KV20** | 1 对 / 10 对字段。与 KV10 合起来是三个点，可读出每字段的边际成本 |
| KV10_Caller | KV10 基础上开启 caller（输出源码位置） |
| MsgOnly | 纯消息，无字段 |
| **MsgOnly_Caller** | 纯消息 + caller。与 MsgOnly 相减即 caller 的净固定成本 |
| Disabled | 级别短路：WARN 阈值下打 Debug，考察短路路径开销 |
| **Disabled_KV10** | 级别短路但带 5 对字段，检验「调高级别就没成本」这个假设 |
| **Context** | trace_id 透传。go-logger 自带提取，其余三库由调用方自备 helper |
| **EscapeHeavy** | 需转义的字段值（嵌套 JSON + 换行 + 中文），区别于其余场景的干净 ASCII |
| **ObjectField** | 结构体字段。各库都无专用分支，只能走 reflect / encoding/json |
| **Child** | 子 Logger 稳态：绑定 4 个常驻字段后构造一次、打多条 |
| **ChildBuild** | 每次重新构造子 Logger 并打一条（每请求一个子 Logger 的写法） |
| **TypedValues** | error + `time.Duration` + int：业务里最常见、却最容易落进 fmt 兜底的类型 |
| **Text** | 文本 / 控制台格式（各库的文本编码路径与 JSON 完全独立） |
| **BigPayload** | 单字段 64KB：单条体量异常时的行为 |
| **SinkRaw** | 四库均直写 `*os.File`，无缓冲，每条一次 write 系统调用 |
| **SinkBuffered** | 四库各按自己推荐的方式做缓冲写出 |

### 为什么字段规模要测三个点

单点数据没法外推。"固定成本 + 每字段成本"这个结构至少要两点才能分离，而只给
KV10（5 对）的话，想按自己的字段数估算就只能瞎乘。KV2 / KV10 / KV20 三点给出斜率，
顺带能看出哪些库的每字段成本明显非线性。

三点一摊开，单看 KV10 会漏掉的结论就出来了：go-logger 的优势主要在**固定成本**，
而每字段的边际成本与 zerolog 接近——字段越多两者越靠近。只看 5 对那一行，会把这条
斜率差异误当成恒定的倍数关系。

### `Context` 这组的口径不对等，而且故意不抹平

go-logger 的 `*Context` 系列自带 trace_id 提取；zap / zerolog / slog 没有这一层，
业务侧必须自己写 helper 再把字段显式挂上去——装置里的 `traceIDFromCtx` 就是那个 helper。
本场景对比的是"把 trace_id 落进日志行"的总代价，而不是某个 API 的调用开销。

不对等之处写在明处：该 helper 只读一次 `ctx.Value`，而 go-logger 的内置提取会先探一次
OTel span 再回落到 `ctx.Value`。**本场景里 go-logger 做的是严格更多的工作，数字对它不利。**
抹平这点反而失真，因为那不是两边各自的默认行为。

### `EscapeHeavy` 补的是什么洞

其余场景的字段值都是干净 ASCII，整段可以 memmove；而真实字段里嵌 JSON、带换行、含中文
都很常见，必须逐字节改写。两条路径的成本不可互推——只测前者，等于只测了最省的那一档。

这一组也是本次新增场景里 **go-logger 唯一明确落后的一项**：转义密集时 zerolog 更快，
且 go-logger 这条路径上还有一次堆分配而 zerolog 是零分配。只报对自己有利的场景没有意义，
而失分项恰好是下一步优化该去的地方。

### `Disabled_KV10` 修正的常见假设

只带 1 对字段时四库都"几乎免费"，于是很容易得出「把级别调高就没成本了」。字段加到 5 对后
结论要修正：参数在**调用点**就构造完了，级别判断再早也省不掉，短路路径的耗时随字段数上涨。

更值得看的是分配。同样是"本该啥也不做"的路径，go-logger / zerolog / slog 都是 0 allocs/op，
而 zap 的强类型 `Field` 即使级别不通过也照样构造并落堆——**被关掉的 Debug 日志里只有它在持续
产生垃圾**，这在高频 Debug 的代码路径上会变成真实的 GC 压力。

> 这一组最初的说明断言"变参装箱省不掉那几次堆分配"，实测把它推翻了：装箱确实发生，但常量与
> 小整数不落堆、`[]any` 也没逃逸，分配是 0。成本结论必须测，不能推。

### 为什么必须有 `Sink*` 这组

其余场景输出到 `io.Discard`，测的是纯格式化成本。但生产里日志要真的落盘：裸 `*os.File`
每条一次系统调用再加一把互斥，这部分能占到单条总成本的 85~97%。实测直写文件时四库的
差距从 io.Discard 下的 3~5 倍收敛到 1.4 倍以内——**只看 `io.Discard` 的数字去选库或调优，
会把全部精力投在不到 15% 的那段上**。

缓冲口径的差异本身也是对比的一部分：zap 自带 `BufferedWriteSyncer`、go-logger 自带异步
批量管道，而 zerolog 与 slog 需要调用方自备 `bufio` 与并发保护。

> `Sink*` 结果受文件系统影响，跨机器不可直接比较，同机器内的相对关系才有意义。

### 关于 go-logger 独有能力

脱敏、行长上限、服务身份字段是 zap / zerolog / slog 都不具备的能力。把它们算进四库对比
等于让 go-logger 多做一份别人不做的工作，数字会失真——所以**四库场景里这些功能一律关闭**，
其真实代价由 `bench_feature_test.go` 单独测量，报告中独立成表。

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
├── bench_child_test.go            # 子 Logger（With）场景基准测试
├── bench_value_test.go            # 值类型 / text / 大字段场景基准测试
├── bench_shape_test.go            # 调用形状场景（字段规模 / ctx / 转义 / 对象字段）
├── bench_sink_test.go             # 输出目标（直写 / 缓冲）场景基准测试
├── bench_feature_test.go          # go-logger 独有能力的开启代价
├── loggers.go                     # 四库对齐的测试装置
├── fixtures.go                    # 扩展场景测试装置（输出目标 / 子 Logger / 值类型）
├── fixtures_shape.go              # 调用形状测试装置（字段规模 / ctx / 转义 / 对象）
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
