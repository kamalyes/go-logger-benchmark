/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-09-29 09:12:36
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-09-29 09:12:36
 * @FilePath: \go-logger-benchmark\loggers.go
 * @Description: 测试装置 — 四库对齐初始化与场景日志函数
 *
 * 对齐口径：JSON 格式 + Info 级别 + io.Discard 输出（同机同场景对比）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"io"
	"log/slog"

	gologger "github.com/kamalyes/go-logger"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// benchMsg 公共测试消息（所有场景复用，消除消息长度差异带来的干扰）
const benchMsg = "user login"

// ---------- go-logger ----------

// gl go-logger 基线：caller 关闭
var gl = gologger.NewLogger().
	WithFormat(gologger.FormatJSON).
	WithOutput(io.Discard).
	WithShowCaller(false)

// glCaller go-logger caller 开启
var glCaller = gologger.NewLogger().
	WithFormat(gologger.FormatJSON).
	WithOutput(io.Discard).
	WithShowCaller(true)

// glWarn go-logger 仅放行 WARN 及以上（级别短路场景用）
var glWarn = gologger.NewLogger().
	WithFormat(gologger.FormatJSON).
	WithOutput(io.Discard).
	WithShowCaller(false).
	WithLevel(gologger.WARN)

// ---------- zap ----------

// zl zap core（强类型 Field API）
var zl = zap.New(zapcore.NewCore(
	zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
	zapcore.AddSync(io.Discard),
	zapcore.InfoLevel,
))

// zlCaller zap caller 开启
var zlCaller = zap.New(zapcore.NewCore(
	zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
	zapcore.AddSync(io.Discard),
	zapcore.InfoLevel,
), zap.AddCaller())

// zsugar zap Sugared（KV 装箱 API）
var zsugar = zl.Sugar()

// zlWarn zap 仅放行 WARN 及以上（级别短路场景用）
var zlWarn = zap.New(zapcore.NewCore(
	zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
	zapcore.AddSync(io.Discard),
	zap.NewAtomicLevelAt(zapcore.WarnLevel),
))

// ---------- zerolog ----------

// zelog zerolog
var zelog = zerolog.New(io.Discard).With().Timestamp().Logger()

// zelogCaller zerolog caller 开启
var zelogCaller = zerolog.New(io.Discard).With().Timestamp().Caller().Logger()

// zelogWarn zerolog 仅放行 WARN 及以上（级别短路场景用）
var zelogWarn = zelog.Level(zerolog.WarnLevel)

// ---------- slog ----------

// slogL slog JSON Handler
var slogL = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

// slogCaller slog 输出源码位置（等价 caller 场景）
var slogCaller = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
	Level:     slog.LevelInfo,
	AddSource: true,
}))

// slogWarn slog 仅放行 WARN 及以上（级别短路场景用）
var slogWarn = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))

// ============================================================================
// 场景日志函数：同一场景下各库按各自惯用 API 记录等价日志
// 字段口径统一为 5 对（10 个）混合类型 KV：3 个 string + 2 个 int
// ============================================================================

// ---------- KV10：5 对（10 个）混合类型字段 ----------

func logKV10GoLogger() {
	gl.InfoKV(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

func logKV10GoLoggerCaller() {
	glCaller.InfoKV(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

func logKV10Zap() {
	zl.Info(benchMsg,
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("ns", "business"),
		zap.Int("conn", 9527), zap.String("region", "cn"))
}

func logKV10ZapCaller() {
	zlCaller.Info(benchMsg,
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("ns", "business"),
		zap.Int("conn", 9527), zap.String("region", "cn"))
}

func logKV10ZapSugared() {
	zsugar.Infow(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

func logKV10Zerolog() {
	zelog.Info().
		Int("uid", 10086).Str("app", "gameai").Str("ns", "business").
		Int("conn", 9527).Str("region", "cn").
		Msg(benchMsg)
}

func logKV10ZerologCaller() {
	zelogCaller.Info().
		Int("uid", 10086).Str("app", "gameai").Str("ns", "business").
		Int("conn", 9527).Str("region", "cn").
		Msg(benchMsg)
}

func logKV10Slog() {
	slogL.Info(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

func logKV10SlogCaller() {
	slogCaller.Info(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

// ---------- MsgOnly：纯消息（无字段） ----------

func logMsgGoLogger() { gl.Info(benchMsg) }
func logMsgZap()      { zl.Info(benchMsg) }
func logMsgZerolog()  { zelog.Info().Msg(benchMsg) }
func logMsgSlog()     { slogL.Info(benchMsg) }

// ---------- Disabled：级别短路（WARN 阈值下打 Debug） ----------

func logDisabledGoLogger() { glWarn.DebugKV(benchMsg, "uid", 10086) }
func logDisabledZap()      { zlWarn.Debug(benchMsg, zap.Int("uid", 10086)) }
func logDisabledZerolog()  { zelogWarn.Debug().Int("uid", 10086).Msg(benchMsg) }
func logDisabledSlog()     { slogWarn.Debug(benchMsg, "uid", 10086) }
