package common

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
	"go.uber.org/zap"
)

type TQLLogger struct {
	logger *zap.Logger
}

func NewTQLLogger(logger *zap.Logger) TQLLogger {
	return TQLLogger{
		logger: logger,
	}
}

// WithFields creates a new logger that will include the specified fields
// in all subsequent logs in addition to fields attached to the context
// of the parent logger. Note that fields are not deduplicated.
func (tqll TQLLogger) WithFields(fields map[string]any) tql.Logger {
	newFields := make([]zap.Field, len(fields))
	i := 0

	for k, v := range fields {
		switch val := v.(type) {
		// zap.Any will base64 encode byte slices, but we want them printed as hexadecimal.
		case []byte:
			newFields[i] = zap.String(k, fmt.Sprintf("%x", val))
		default:
			newFields[i] = zap.Any(k, val)
		}
		i++
	}

	return TQLLogger{
		logger: tqll.logger.With(newFields...),
	}
}

func (tqll TQLLogger) Info(msg string) {
	tqll.logger.Info(msg)
}

func (tqll TQLLogger) Error(msg string) {
	tqll.logger.Error(msg)
}
