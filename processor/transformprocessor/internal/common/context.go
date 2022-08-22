package common

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
)

// Provides an example of extending the base ProcessorContext interface provided
// by the TQL, in this case with the default ProcessorContextData implementation.
// In theory this could be any runtime data provided by a processor.
type TransformProcessorContext struct {
	tql.ProcessorContextData

	Info          string
	ContextObject context.Context
}
