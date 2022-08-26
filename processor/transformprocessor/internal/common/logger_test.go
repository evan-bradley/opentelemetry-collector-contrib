package common

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func createLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	return zap.New(core), logs
}

func Test_logger(t *testing.T) {
	tests := []struct {
		name           string
		fields         []map[string]any
		msg            string
		expectedFields map[string]any
	}{
		{
			name:           "Empty case",
			fields:         []map[string]any{},
			msg:            "",
			expectedFields: map[string]any{},
		},
		{
			name:           "No fields",
			fields:         []map[string]any{},
			msg:            "log message",
			expectedFields: map[string]any{},
		},
		{
			name: "One fields call",
			fields: []map[string]any{
				{
					"string": "test",
					"int":    int64(1),
					"float":  1.0,
					"bool":   true,
					"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
					"nil":    nil,
				},
			},
			msg: "log message",
			expectedFields: map[string]any{
				"string": "test",
				"int":    int64(1),
				"float":  1.0,
				"bool":   true,
				"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
				"nil":    nil,
			},
		},
		{
			name: "Two fields calls",
			fields: []map[string]any{
				{
					"string": "test",
					"int":    int64(1),
					"float":  1.0,
					"bool":   true,
					"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
					"nil":    nil,
				},
				{
					"string": "test1",
				},
			},
			msg: "log message",
			expectedFields: map[string]any{
				"string": "test1",
				"int":    int64(1),
				"float":  1.0,
				"bool":   true,
				"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
				"nil":    nil,
			},
		},
		{
			name: "Three fields calls",
			fields: []map[string]any{
				{
					"string": "test",
					"int":    int64(1),
					"float":  1.0,
					"bool":   true,
					"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
					"nil":    nil,
				},
				{
					"string": "test1",
				},
				{
					"string": "test2",
					"int":    int64(2),
				},
			},
			msg: "log message",
			expectedFields: map[string]any{
				"string": "test2",
				"int":    int64(2),
				"float":  1.0,
				"bool":   true,
				"[]byte": []byte{0, 1, 2, 3, 4, 5, 6, 7},
				"nil":    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logs := createLogger()
			tqll := NewTQLLogger(logger)

			for _, f := range tt.fields {
				tqll = tqll.WithFields(f).(TQLLogger)
			}

			tqll.Info(tt.msg)

			log := logs.All()[0]

			assert.Equal(t, tt.msg, log.Message)

			context := map[string]zap.Field{}

			for _, field := range log.Context {
				context[field.Key] = field
			}

			for k, v := range tt.expectedFields {
				assert.Equal(t, k, context[k].Key)
				switch context[k].Type {
				case zapcore.StringType:
					switch v.(type) {
					case string:
						assert.Equal(t, v, context[k].String)
					case []byte:
						assert.Equal(t, fmt.Sprintf("%x", v), context[k].String)
					case nil:
						assert.Equal(t, fmt.Sprintf("%x", v), context[k].String)
					}
				case zapcore.Int64Type:
					assert.Equal(t, v, context[k].Integer)
				case zapcore.Float64Type:
					assert.Equal(t, v, math.Float64frombits(uint64(context[k].Integer)))
				}
			}
		})
	}
}

func Test_logger_logging(t *testing.T) {
	tests := []struct {
		name      string
		loglevels []string
		log       func(tqll TQLLogger, msg string)
	}{
		{
			name:      "Info",
			loglevels: []string{"info"},
			log: func(tqll TQLLogger, msg string) {
				tqll.Info(msg)
			},
		},
		{
			name:      "Error",
			loglevels: []string{"error"},
			log: func(tqll TQLLogger, msg string) {
				tqll.Error(msg)
			},
		},
		{
			name:      "Multiple Info",
			loglevels: []string{"info", "info", "info"},
			log: func(tqll TQLLogger, msg string) {
				tqll.Info(msg)
				tqll.Info(msg)
				tqll.Info(msg)
			},
		},
		{
			name:      "Multiple Error",
			loglevels: []string{"error", "error", "error"},
			log: func(tqll TQLLogger, msg string) {
				tqll.Error(msg)
				tqll.Error(msg)
				tqll.Error(msg)
			},
		},
		{
			name:      "Mixed loglevels",
			loglevels: []string{"info", "error", "info"},
			log: func(tqll TQLLogger, msg string) {
				tqll.Info(msg)
				tqll.Error(msg)
				tqll.Info(msg)
			},
		},
	}

	for _, tt := range tests {
		logger, logs := createLogger()
		tqll := NewTQLLogger(logger)

		tt.log(tqll, "testing")

		for i, log := range logs.All() {
			assert.Equal(t, tt.loglevels[i], log.Level.String())
		}
	}
}

func BenchmarkAddingTenFields(b *testing.B) {
	tqll := NewTQLLogger(zap.NewNop())
	fields := map[string]any{
		"string0": "test0",
		"string1": "test1",
		"string2": "test2",
		"string3": "test3",
		"string4": "test4",
		"int":     int64(1),
		"float":   1.0,
		"bool":    true,
		"[]byte":  []byte{0, 1, 2, 3, 4, 5, 6, 7},
		"nil":     nil,
	}

	for i := 0; i < b.N; i++ {
		_ = tqll.WithFields(fields).(TQLLogger)
	}
}
