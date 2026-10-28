/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-10-05 17:50:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-10-05 17:50:00
 * @FilePath: \go-logger-benchmark\fixtures_shape.go
 * @Description: 测试装置 — 调用形状：字段规模、转义密度、ctx 透传、对象字段
 *
 * 已有场景把字段规模固定在 5 对、值固定为干净 ASCII、不涉及 ctx。这三个前提
 * 恰好都是最省的那一档，于是整张表只覆盖了"最顺"的那条路径：
 *   - 固定 5 对 → 看不出每字段斜率，无法按自己的字段数外推
 *   - 干净 ASCII → 完全走转义快路径，而真实字段里嵌 JSON、换行、中文很常见
 *   - 不带 ctx  → 而 trace_id 透传几乎是服务端每条日志都要做的事
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"context"

	gologger "github.com/kamalyes/go-logger"
	"go.uber.org/zap"
)

// ============================================================================
// 公共测试数据
// ============================================================================

// benchTraceID trace_id 取值（OTel trace id 的 hex 形态）
const benchTraceID = "7b3fa9c0d5e18263"

// benchCtx 携带 trace_id 的 ctx
var benchCtx = gologger.ContextWithTraceID(context.Background(), benchTraceID)

// benchEscape 需要大量转义的字段值：嵌套 JSON（引号 + 反斜杠）、换行、制表符、中文
//
// 真实来源是把上游响应体、SQL、用户输入整段塞进一个字段。这条路径与"干净
// ASCII"完全不同——后者可以整段 memmove，前者必须逐字节改写，成本不可互推。
var benchEscape = `{"query":"SELECT * FROM orders WHERE name = \"张三\"","path":"C:\\logs\\app.log"}` +
	"\n\tretry=1 备注：\"已确认\""

// benchOrder 典型业务对象（考察各库的反射兜底路径）
type benchOrder struct {
	ID       string `json:"id"`
	UID      int64  `json:"uid"`
	Amount   int64  `json:"amount"`
	Items    int    `json:"items"`
	Paid     bool   `json:"paid"`
	Currency string `json:"currency"`
}

// benchObj 预先构造好的对象（排除构造成本，只测编码）
var benchObj = benchOrder{
	ID:       "ORD-20260929-0001",
	UID:      10086,
	Amount:   25900,
	Items:    3,
	Paid:     true,
	Currency: "CNY",
}

// traceIDFromCtx 调用方自备的 trace_id 提取
//
// go-logger 的 *Context 系列自带这一步；zap / zerolog / slog 没有对应层，
// 业务侧必须自己写这个 helper 并把字段显式挂到每条日志上——本场景对比的
// 正是"把 trace_id 落进日志行"的真实代价，而不是某个 API 的调用开销。
//
// 口径必须说清：本 helper 只读一次 ctx.Value，而 go-logger 的内置提取会先
// 探一次 OTel span 再回落到 ctx.Value。也就是说本场景里 go-logger 做的是
// 严格更多的工作，数字对它不利——但这才是两边各自的默认行为，抹平反而失真。
func traceIDFromCtx(ctx context.Context) string {
	if v := ctx.Value(gologger.ContextKeyTraceID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ============================================================================
// 场景日志函数
// ============================================================================

// ---------- KV2 / KV20：字段规模的两个端点 ----------
// 与已有的 KV10（5 对）合起来是 1 / 5 / 10 对三点，可读出每字段的边际成本。
// 类型配比与 KV10 保持 3:2 的 string:int，避免把类型差异混进规模差异。

func logKV2GoLogger() { gl.InfoKV(benchMsg, "uid", 10086) }
func logKV2Zap()      { zl.Info(benchMsg, zap.Int("uid", 10086)) }
func logKV2Zerolog()  { zelog.Info().Int("uid", 10086).Msg(benchMsg) }
func logKV2Slog()     { slogL.Info(benchMsg, "uid", 10086) }

func logKV20GoLogger() {
	gl.InfoKV(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn",
		"zone", "cn-sh-1", "shard", 7, "tenant", "acme", "ver", "v2.3.1", "seq", 880123)
}

func logKV20Zap() {
	zl.Info(benchMsg,
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("ns", "business"),
		zap.Int("conn", 9527), zap.String("region", "cn"),
		zap.String("zone", "cn-sh-1"), zap.Int("shard", 7), zap.String("tenant", "acme"),
		zap.String("ver", "v2.3.1"), zap.Int("seq", 880123))
}

func logKV20Zerolog() {
	zelog.Info().
		Int("uid", 10086).Str("app", "gameai").Str("ns", "business").
		Int("conn", 9527).Str("region", "cn").
		Str("zone", "cn-sh-1").Int("shard", 7).Str("tenant", "acme").
		Str("ver", "v2.3.1").Int("seq", 880123).
		Msg(benchMsg)
}

func logKV20Slog() {
	slogL.Info(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn",
		"zone", "cn-sh-1", "shard", 7, "tenant", "acme", "ver", "v2.3.1", "seq", 880123)
}

// ---------- Context：trace_id 透传 ----------
// 四边都产出 trace_id + 3 对业务字段，口径一致；差别在于谁来做提取。

func logCtxGoLogger() {
	gl.InfoContextKV(benchCtx, benchMsg, "uid", 10086, "app", "gameai", "region", "cn")
}

func logCtxZap() {
	zl.Info(benchMsg,
		zap.String(gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx)),
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("region", "cn"))
}

func logCtxZerolog() {
	zelog.Info().
		Str(gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx)).
		Int("uid", 10086).Str("app", "gameai").Str("region", "cn").
		Msg(benchMsg)
}

func logCtxSlog() {
	slogL.InfoContext(benchCtx, benchMsg,
		gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx),
		"uid", 10086, "app", "gameai", "region", "cn")
}

// ---------- EscapeHeavy：需转义的字段值 ----------

func logEscapeGoLogger() { gl.InfoKV(benchMsg, "body", benchEscape) }
func logEscapeZap()      { zl.Info(benchMsg, zap.String("body", benchEscape)) }
func logEscapeZerolog()  { zelog.Info().Str("body", benchEscape).Msg(benchMsg) }
func logEscapeSlog()     { slogL.Info(benchMsg, "body", benchEscape) }

// ---------- ObjectField：结构体字段（反射兜底） ----------
// 各库都没有结构体的专用分支，只能走 reflect / encoding/json，
// 这是日常把 DTO 整个丢进日志时的真实成本。

func logObjGoLogger() { gl.InfoKV(benchMsg, "order", benchObj) }
func logObjZap()      { zl.Info(benchMsg, zap.Any("order", benchObj)) }
func logObjZerolog()  { zelog.Info().Interface("order", benchObj).Msg(benchMsg) }
func logObjSlog()     { slogL.Info(benchMsg, "order", benchObj) }

// ---------- MsgOnly_Caller：caller 的固定成本 ----------
// KV10_Caller 把 caller 成本和 5 对字段的编码成本混在一起。
// 挂到最省的纯消息场景上，caller 自身的 runtime 栈回溯成本才看得清。

func logMsgGoLoggerCaller() { glCaller.Info(benchMsg) }
func logMsgZapCaller()      { zlCaller.Info(benchMsg) }
func logMsgZerologCaller()  { zelogCaller.Info().Msg(benchMsg) }
func logMsgSlogCaller()     { slogCaller.Info(benchMsg) }

// ---------- Disabled_KV10：级别短路但带 5 对字段 ----------
// 已有的 Disabled 只带 1 对，四库看着都"几乎免费"，于是很容易得出
// "把级别调高就没成本了"。字段加到 5 对后结论要修正：参数在**调用点**
// 就构造完了，级别判断再早也省不掉这一步，短路路径的耗时随字段数上涨。
//
// 但"上涨"不等于"分配"——这一点必须靠实测而不是推理：本机实测
// go-logger / zerolog / slog 在短路路径上都是 0 allocs/op（值是常量与小整数，
// 装箱不落堆，且 []any 未逃逸），只有 zap 例外，它的强类型 Field 即使在
// 级别不通过时也照样构造并落到堆上。
//
// 所以这一组真正量化的是两件事：短路路径随字段数的涨幅，以及谁在
// "本该啥也不做"的路径上仍然分配内存。

func logDisabledKV10GoLogger() {
	glWarn.DebugKV(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn")
}

func logDisabledKV10Zap() {
	zlWarn.Debug(benchMsg,
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("ns", "business"),
		zap.Int("conn", 9527), zap.String("region", "cn"))
}

func logDisabledKV10Zerolog() {
	zelogWarn.Debug().
		Int("uid", 10086).Str("app", "gameai").Str("ns", "business").
		Int("conn", 9527).Str("region", "cn").
		Msg(benchMsg)
}

func logDisabledKV10Slog() {
	slogWarn.Debug(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn")
}
