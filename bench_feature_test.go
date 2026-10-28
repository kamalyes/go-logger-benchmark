/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-26 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-26 00:00:00
 * @FilePath: \go-logger-benchmark\bench_feature_test.go
 * @Description: go-logger 独有能力的开启代价（脱敏、行长上限、服务身份字段）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"errors"
	"io"
	"strings"
	"testing"

	gologger "github.com/kamalyes/go-logger"
)

// ══════════════════════════════════════════════════════════════════════════════
// 为什么这组基准不参与四库对比
//
// 脱敏、行长上限、服务身份字段是 zap/zerolog/slog 都不具备的能力。把它们算进
// 四库对比等于让 go-logger 多做一份别人不做的工作，数字会失真——所以四库场景
// 里这些功能一律关闭（见 loggers.go 的 baselineGoLogger），真实代价在此单列。
//
// 命名用 Feature 前缀而非库名前缀，报告生成器因此不会把它们并入对比表。
// ══════════════════════════════════════════════════════════════════════════════

// featureLogger 构造只改动指定能力的 go-logger 装置
func featureLogger(opts ...func(*gologger.Logger) *gologger.Logger) *gologger.Logger {
	l := gologger.NewLogger().
		WithFormat(gologger.FormatJSON).
		WithOutput(io.Discard).
		WithShowCaller(false).
		WithSanitize(gologger.SanitizeOff).
		WithMaxLineBytes(0)
	for _, opt := range opts {
		l = opt(l)
	}
	return l
}

var (
	// 脱敏三档
	featSanOff    = featureLogger()
	featSanKeys   = featureLogger(func(l *gologger.Logger) *gologger.Logger { return l.WithSanitize(gologger.SanitizeKeys) })
	featSanValues = featureLogger(func(l *gologger.Logger) *gologger.Logger { return l.WithSanitize(gologger.SanitizeKeysAndValues) })

	// 行长上限开关
	featLimitOff = featureLogger()
	featLimitOn  = featureLogger(func(l *gologger.Logger) *gologger.Logger {
		return l.WithMaxLineBytes(gologger.DefaultMaxLineBytes)
	})

	// 服务身份字段
	featIdentityOff = featureLogger()
	featIdentityOn  = featureLogger(func(l *gologger.Logger) *gologger.Logger {
		return l.WithService("order-svc").WithInstance("order-svc-7d9f-abcde").WithEnv("prod")
	})
)

var featErr = errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

// logPlainKV 常规字段：无一命中脱敏规则，测的是"开启脱敏的净代价"
func logPlainKV(l *gologger.Logger) {
	l.InfoKV(benchMsg, "uid", 10086, "app", "gameai", "ns", "business",
		"conn", 9527, "region", "cn")
}

// logOpsKV 运维字段：全部要过一遍豁免表与后缀字典树，是字段名判定的最坏情况
func logOpsKV(l *gologger.Logger) {
	l.InfoKV(benchMsg, "ip_address", "10.0.0.1", "cache_key", "user:1001",
		"error_code", "E4001", "idempotency_key", "idem-1", "status_code", 200)
}

// logSensitiveKV 命中脱敏规则的字段
func logSensitiveKV(l *gologger.Logger) {
	l.InfoKV(benchMsg, "authorization", "Bearer abc", "password", "x",
		"customer_phone", "13800138000", "customer_email", "a@b.com", "uid", 1)
}

func benchFeature(b *testing.B, l *gologger.Logger, log func(*gologger.Logger)) {
	b.Helper()
	b.ReportAllocs()
	for range b.N {
		log(l)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Sanitize：脱敏
// ══════════════════════════════════════════════════════════════════════════════

// ─── 常规字段（无一命中规则）：开启脱敏的净代价 ───

func BenchmarkFeature_Sanitize_Off(b *testing.B)    { benchFeature(b, featSanOff, logPlainKV) }
func BenchmarkFeature_Sanitize_Keys(b *testing.B)   { benchFeature(b, featSanKeys, logPlainKV) }
func BenchmarkFeature_Sanitize_Values(b *testing.B) { benchFeature(b, featSanValues, logPlainKV) }

// ─── 运维字段：字段名判定的最坏情况 ───

func BenchmarkFeature_Sanitize_OpsKeys_Off(b *testing.B)  { benchFeature(b, featSanOff, logOpsKV) }
func BenchmarkFeature_Sanitize_OpsKeys_Keys(b *testing.B) { benchFeature(b, featSanKeys, logOpsKV) }

// ─── 命中规则的字段 ───

func BenchmarkFeature_Sanitize_Hit(b *testing.B) { benchFeature(b, featSanKeys, logSensitiveKV) }

// ─── 子 Logger 的常驻字段：脱敏在预编码阶段完成，稳态应与关闭时持平 ───

func BenchmarkFeature_Sanitize_ChildSteady(b *testing.B) {
	child := featSanKeys.WithField("access_token", "at_secret_value").WithField("request_id", "rq_1")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		child.InfoMsg(benchMsg)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// MaxLineBytes：行长上限
// ══════════════════════════════════════════════════════════════════════════════

// ─── 常规日志：上限检查的净代价 ───

func BenchmarkFeature_Limit_Off(b *testing.B) { benchFeature(b, featLimitOff, logPlainKV) }
func BenchmarkFeature_Limit_On(b *testing.B)  { benchFeature(b, featLimitOn, logPlainKV) }

// ─── 失控的大字段：上限的实际收益 ───

var featBigPayload = strings.Repeat("x", 200*1024)

func logBigPayload(l *gologger.Logger) {
	l.InfoKV(benchMsg, "payload", featBigPayload)
}

func BenchmarkFeature_Limit_BigPayload_Off(b *testing.B) {
	benchFeature(b, featLimitOff, logBigPayload)
}
func BenchmarkFeature_Limit_BigPayload_On(b *testing.B) {
	benchFeature(b, featLimitOn, logBigPayload)
}

// ══════════════════════════════════════════════════════════════════════════════
// Identity：服务身份字段（预编码片段，每条一次 memcpy）
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkFeature_Identity_Off(b *testing.B) { benchFeature(b, featIdentityOff, logPlainKV) }
func BenchmarkFeature_Identity_On(b *testing.B)  { benchFeature(b, featIdentityOn, logPlainKV) }

// ══════════════════════════════════════════════════════════════════════════════
// FailureLog：持续失败的限频汇总
//
// 对比"每次失败都直接打日志"与"经限频汇总"的成本。限频侧的收益不在单条耗时，
// 而在输出条数——故障期间后者把 N 条压成个位数，这里用条数差体现。
// ══════════════════════════════════════════════════════════════════════════════

func BenchmarkFeature_Failure_Direct(b *testing.B) {
	l := featureLogger()
	b.ReportAllocs()
	for range b.N {
		l.ErrorKV("dial upstream failed", "addr", "10.0.0.1:5432", "err", featErr)
	}
}

func BenchmarkFeature_Failure_RateLimited(b *testing.B) {
	l := featureLogger()
	fl := gologger.NewFailureLog(l, gologger.FailureLogConfig{})
	b.ReportAllocs()
	for range b.N {
		fl.Fail(nil, "upstream:dial", gologger.ERROR, "dial upstream failed",
			"addr", "10.0.0.1:5432", "err", featErr)
	}
}
