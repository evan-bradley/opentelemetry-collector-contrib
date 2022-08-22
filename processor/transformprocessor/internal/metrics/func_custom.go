// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/contexts/tqlmetrics"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor/internal/common"
)

// Implements a function using a custom ProcessorContext implementation.
func custom(target tql.Setter, value tql.Getter) (tql.ExprFunc, error) {
	return func(ctx tql.TransformContext) interface{} {
		mtc, ok := ctx.(tqlmetrics.MetricTransformContext)
		if !ok {
			return nil
		}

		tpc, ok := mtc.GetProcessorContext().(common.TransformProcessorContext)
		if !ok {
			return nil
		}

		mtc.GetProcessorContext().GetLogger().Info("Example log")
		tpc.GetLogger().Info(tpc.Info)

		return nil
	}, nil
}
