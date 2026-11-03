# 基准的规范跑法定义在这里，而不是散在 CI 的工作流步骤里
#
# 以前这条命令只写在 .github/workflows/benchmark.yml 的一个 step 里，于是
# 「这些数是怎么跑出来的」只能去读工作流，而工作流里的参数一改，默认值就跟着变
# 现在它只有一处定义：本地跑和 CI 引用的是同一个文件
#
# [EN] The canonical way these benchmarks are run lives here, not scattered through CI steps
# It used to exist only as a step in .github/workflows/benchmark.yml, so answering "how were
# these numbers produced" meant reading the workflow, and editing the parameters there
# silently changed the defaults
# There is one definition now, shared by a local run and by what CI references

# BENCH_FLAGS 规范参数
# count=3 是为了看得见轮间波动 —— 单轮数字不可信，本项目实测过同一格波动到 20%
BENCH_FLAGS ?= -run='^$$' -bench=. -benchmem -count=3 -timeout=30m
BENCH_OUT   ?= benchmark_output.txt

.PHONY: bench report bench-smoke clean

# bench 规范跑法，产出 BENCHMARKS.md 与 benchmarks/ 下的数据
#
# 只在固定的那台物理机上跑：共享 CI runner 是 2-4 vCPU、与别的作业共享物理机、
# CPU 型号还会换，它的数字和上一次的不可比，提交上去会让报告长期显示一组
# 互相不可比的值
#
# [EN] The canonical run, producing BENCHMARKS.md and the data under benchmarks/
# Only on one fixed physical machine: a shared CI runner has two to four vCPUs, shares a
# host with other jobs and changes CPU model between runs, so its numbers cannot be
# compared with the previous ones and committing them leaves the report showing mutually
# incomparable values
bench:
	go test $(BENCH_FLAGS) ./... | tee $(BENCH_OUT)
	$(MAKE) report

# report 从已有输出重新生成报告，不重跑基准
# [EN] Regenerates the report from an existing output without re-running the benchmarks
report:
	go run ./bootstrap/report -parse $(BENCH_OUT)

# bench-smoke CI 用的正确性闸门：只验基准能编译、能跑到结束、生成器能解析输出
#
# 刻意不复用 BENCH_FLAGS：这一档的数字不采信，参数就不该和规范跑法长得一样，
# 否则下一个人会以为报告里的数是这么跑出来的
#
# [EN] The correctness gate CI uses: the benchmarks compile, run to completion and the
# generator parses the output
# Deliberately does not reuse BENCH_FLAGS, since this run's numbers are not trusted and
# its parameters should not look like the canonical ones, or the next reader will assume
# the report was produced this way
bench-smoke:
	go test -run='^$$' -bench=. -benchmem -benchtime=10x -count=1 -timeout=20m ./... | tee $(BENCH_OUT)
	go run ./bootstrap/report -parse $(BENCH_OUT)
	git --no-pager diff --stat

clean:
	rm -f $(BENCH_OUT)
