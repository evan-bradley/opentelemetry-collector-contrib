package common

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
	"go.uber.org/zap"
)

type TQLLogger struct {
	logger *zap.Logger
	fields []zap.Field
}

func NewTQLLogger(logger *zap.Logger) TQLLogger {
	return TQLLogger{
		logger: logger,
		fields: []zap.Field{},
	}
}

func (tqll TQLLogger) WithFields(newFields map[string]interface{}) tql.Logger {
	// Zap will not deduplicate fields on its own, therefore we want to ensure
	// there is only a single instance of a field key and that users can
	// overwrite fields by putting them into a set and deduplicating them here.
	fieldSet := make(map[string]zap.Field)

	for _, field := range tqll.fields {
		fieldSet[field.Key] = field
	}

	for k, v := range newFields {
		switch val := v.(type) {
		case int64:
			fieldSet[k] = zap.Int64(k, val)
		case float64:
			fieldSet[k] = zap.Float64(k, val)
		case string:
			fieldSet[k] = zap.String(k, val)
		case bool:
			fieldSet[k] = zap.Bool(k, val)
		}
	}

	fields := []zap.Field{}

	for _, v := range fieldSet {
		fields = append(fields, v)
	}

	return TQLLogger{
		fields: fields,
		logger: tqll.logger,
	}
}

func (tqll TQLLogger) Info(msg string) {
	tqll.logger.Info(msg, tqll.fields...)
}

func (tqll TQLLogger) Error(msg string) {
	tqll.logger.Error(msg, tqll.fields...)
}
