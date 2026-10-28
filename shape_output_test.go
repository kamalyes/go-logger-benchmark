/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-10-05 19:33:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-10-05 19:33:00
 * @FilePath: \go-logger-benchmark\shape_output_test.go
 * @Description: 场景有效性校验 — 四库是否真的产出了等价的日志内容
 *
 * 基准测试最隐蔽的失效方式是「测了个空操作」：某个库的字段没真正写出、
 * ctx 里的 trace_id 没被取到、级别判断把整条日志短路掉了——跑起来一样快，
 * 一样不报错，只是那一行数字毫无意义，还会被当成"某库快得多"的证据。
 *
 * 这里对新增的几组场景逐库断言输出内容，保证对比的是同一份工作量。
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package bench

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	gologger "github.com/kamalyes/go-logger"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// shapeSinks 四库写入各自 buffer 的一组 Logger（与基准装置同配置，仅换输出目标）
type shapeSinks struct {
	goBuf, zapBuf, zeroBuf, slogBuf *bytes.Buffer
	gl                              *gologger.Logger
	zl                              *zap.Logger
	zelog                           zerolog.Logger
	slogL                           *slog.Logger
}

func newShapeSinks() *shapeSinks {
	s := &shapeSinks{
		goBuf:   &bytes.Buffer{},
		zapBuf:  &bytes.Buffer{},
		zeroBuf: &bytes.Buffer{},
		slogBuf: &bytes.Buffer{},
	}
	s.gl = baselineGoLogger(s.goBuf, false)
	s.zl = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(s.zapBuf),
		zapcore.InfoLevel,
	))
	s.zelog = zerolog.New(s.zeroBuf).With().Timestamp().Logger()
	s.slogL = slog.New(slog.NewJSONHandler(s.slogBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return s
}

// each 对四库输出逐一执行断言
func (s *shapeSinks) each(t *testing.T, check func(t *testing.T, lib, line string)) {
	t.Helper()
	for _, c := range []struct {
		lib string
		buf *bytes.Buffer
	}{
		{"go-logger", s.goBuf},
		{"zap", s.zapBuf},
		{"zerolog", s.zeroBuf},
		{"slog", s.slogBuf},
	} {
		line := c.buf.String()
		if strings.TrimSpace(line) == "" {
			t.Errorf("%s: 没有任何输出，该场景在这个库上测的是空操作", c.lib)
			continue
		}
		check(t, c.lib, line)
	}
}

// mustContain 断言输出同时含有若干子串
func mustContain(t *testing.T, lib, line string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(line, sub) {
			t.Errorf("%s: 输出缺少 %q，四库工作量不等价\n  实得: %s", lib, sub, strings.TrimSpace(line))
		}
	}
}

// TestShape_Context 四库都必须真的把 trace_id 写进日志行
//
// 这是最容易悄悄失效的一组：go-logger 从 ctx 自动提取，其余三库靠
// traceIDFromCtx 显式取值。helper 若取不到（ctx key 类型变了、字段名改了），
// 那三库就少写一个字段、少做一份工作，而基准只会显示它们"更快"。
func TestShape_Context(t *testing.T) {
	s := newShapeSinks()

	s.gl.InfoContextKV(benchCtx, benchMsg, "uid", 10086, "app", "gameai", "region", "cn")
	s.zl.Info(benchMsg,
		zap.String(gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx)),
		zap.Int("uid", 10086), zap.String("app", "gameai"), zap.String("region", "cn"))
	s.zelog.Info().
		Str(gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx)).
		Int("uid", 10086).Str("app", "gameai").Str("region", "cn").
		Msg(benchMsg)
	s.slogL.InfoContext(benchCtx, benchMsg,
		gologger.ContextKeyTraceID, traceIDFromCtx(benchCtx),
		"uid", 10086, "app", "gameai", "region", "cn")

	s.each(t, func(t *testing.T, lib, line string) {
		mustContain(t, lib, line, benchTraceID, "uid", "10086", "app", "gameai", "region")
	})
}

// TestShape_EscapeHeavy 四库都必须真的转义并写出该字段
//
// 若某库把整个值丢了或截断了，转义成本自然就低——那不是更快，是没做。
func TestShape_EscapeHeavy(t *testing.T) {
	s := newShapeSinks()

	s.gl.InfoKV(benchMsg, "body", benchEscape)
	s.zl.Info(benchMsg, zap.String("body", benchEscape))
	s.zelog.Info().Str("body", benchEscape).Msg(benchMsg)
	s.slogL.Info(benchMsg, "body", benchEscape)

	s.each(t, func(t *testing.T, lib, line string) {
		// 转义后原文不可直接比对，改为校验关键片段与转义结果都在。
		mustContain(t, lib, line, "body", "SELECT", `\"`, `\n`, "张三")
	})
}

// TestShape_ObjectField 四库都必须真的展开结构体
//
// 若某库只写出 "{}" 或类型名，反射成本就被跳过了。
func TestShape_ObjectField(t *testing.T) {
	s := newShapeSinks()

	s.gl.InfoKV(benchMsg, "order", benchObj)
	s.zl.Info(benchMsg, zap.Any("order", benchObj))
	s.zelog.Info().Interface("order", benchObj).Msg(benchMsg)
	s.slogL.Info(benchMsg, "order", benchObj)

	s.each(t, func(t *testing.T, lib, line string) {
		mustContain(t, lib, line, "order", "ORD-20260929-0001", "25900", "CNY")
	})
}

// TestShape_KVScaling KV2 / KV10 / KV20 的字段数必须真的是 1 / 5 / 10 对
//
// 三点连线才能读出每字段斜率，前提是这三点的字段数确实不同。
// 任何一处漏写或多写，斜率就是错的，而基准不会报错。
func TestShape_KVScaling(t *testing.T) {
	cases := []struct {
		name      string
		wantPairs int
		keys      []string
	}{
		{"KV2", 1, []string{"uid"}},
		{"KV10", 5, []string{"uid", "app", "ns", "conn", "region"}},
		{"KV20", 10, []string{"uid", "app", "ns", "conn", "region", "zone", "shard", "tenant", "ver", "seq"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newShapeSinks()
			switch c.name {
			case "KV2":
				s.gl.InfoKV(benchMsg, "uid", 10086)
			case "KV10":
				s.gl.InfoKV(benchMsg, "uid", 10086, "app", "gameai", "ns", "business",
					"conn", 9527, "region", "cn")
			case "KV20":
				s.gl.InfoKV(benchMsg,
					"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn",
					"zone", "cn-sh-1", "shard", 7, "tenant", "acme", "ver", "v2.3.1", "seq", 880123)
			}
			line := s.goBuf.String()
			for _, k := range c.keys {
				if !strings.Contains(line, `"`+k+`"`) {
					t.Errorf("%s 缺少字段 %q，字段规模不是声称的 %d 对\n  实得: %s",
						c.name, k, c.wantPairs, strings.TrimSpace(line))
				}
			}
		})
	}
}

// TestShape_DisabledIsReallyDisabled 级别短路场景必须真的没有输出
//
// 反过来的失效方式：阈值设错，日志照常写出，那组数字测的就不是短路路径。
func TestShape_DisabledIsReallyDisabled(t *testing.T) {
	var buf bytes.Buffer
	warnOnly := baselineGoLogger(&buf, false).WithLevel(gologger.WARN)
	warnOnly.DebugKV(benchMsg,
		"uid", 10086, "app", "gameai", "ns", "business", "conn", 9527, "region", "cn")
	if buf.Len() != 0 {
		t.Errorf("WARN 阈值下 Debug 仍有输出，Disabled 组测的不是短路路径: %s", buf.String())
	}
}
