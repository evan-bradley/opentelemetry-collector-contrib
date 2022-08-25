package common

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
	"go.uber.org/zap"
)

type TQLLogger struct {
	logger *zap.Logger
	fields *[]zap.Field
}

func NewTQLLogger(logger *zap.Logger) TQLLogger {
	return TQLLogger{
		logger: logger,
		fields: &[]zap.Field{},
	}
}

func (tqll TQLLogger) WithFields(newFields map[string]interface{}) tql.Logger {
	fieldSet := make(map[string]zap.Field)

	for _, field := range *tqll.fields {
		fieldSet[field.Key] = field
	}

	for k, v := range newFields {
		switch val := v.(type) {
		// zap.Any will base64 encode byte slices, but we want them printed as hexadecimal.
		case []byte:
			fieldSet[k] = zap.String(k, fmt.Sprintf("%x", val))
		default:
			fieldSet[k] = zap.Any(k, val)
		}
	}

	fields := []zap.Field{}

	for _, v := range fieldSet {
		fields = append(fields, v)
	}

	return TQLLogger{
		fields: &fields,
		logger: tqll.logger,
	}
}

func (tqll TQLLogger) Info(msg string) {
	tqll.logger.Info(msg, *tqll.fields...)
}

func (tqll TQLLogger) Error(msg string) {
	tqll.logger.Error(msg, *tqll.fields...)
}
