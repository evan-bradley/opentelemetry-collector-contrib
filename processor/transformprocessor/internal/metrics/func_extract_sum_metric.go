// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor/internal/metrics"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
)

type extractSumMetricArguments struct {
	Monotonic bool `ottlarg:"0"`
}

func newExtractSumMetricFactory() ottl.Factory[ottlmetric.TransformContext] {
	return ottl.NewFactory("extract_sum_metric", &extractSumMetricArguments{}, createExtractSumMetricFunction)
}

func createExtractSumMetricFunction(_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	args, ok := oArgs.(*extractSumMetricArguments)

	if !ok {
		return nil, fmt.Errorf("extractSumMetricFactory args must be of type *extractSumMetricArguments")
	}

	return extractSumMetric(args.Monotonic)
}

// this interface helps unify the logic for extracting data from different histogram types
// all supported metric types' datapoints implement it
type SumCountDataPoint interface {
	Attributes() pcommon.Map
	Sum() float64
	Count() uint64
	StartTimestamp() pcommon.Timestamp
	Timestamp() pcommon.Timestamp
}

type HasSumDataPoint interface {
	HasSum() bool
}

func extractSumMetric(monotonic bool) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	return func(_ context.Context, tCtx ottlmetric.TransformContext) (interface{}, error) {
		return nil, extractDP(tCtx, addSumDataPoint, monotonic, "_sum")
	}, nil
}

func addSumDataPoint(dataPoint SumCountDataPoint, destination pmetric.NumberDataPointSlice) {
	if hs, ok := dataPoint.(HasSumDataPoint); ok && !hs.HasSum() {
		return
	}
	newDp := destination.AppendEmpty()
	dataPoint.Attributes().CopyTo(newDp.Attributes())
	newDp.SetDoubleValue(dataPoint.Sum())
	newDp.SetStartTimestamp(dataPoint.StartTimestamp())
	newDp.SetTimestamp(dataPoint.Timestamp())
}

func extractCountMetric(monotonic bool) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	return func(_ context.Context, tCtx ottlmetric.TransformContext) (interface{}, error) {
		return nil, extractDP(tCtx, addCountDataPoint, monotonic, "_count")
	}, nil
}

func extractDP(tCtx ottlmetric.TransformContext, fillDataPoint func(dataPoint SumCountDataPoint, destination pmetric.NumberDataPointSlice), monotonic bool, suffix string) error {
	metric := tCtx.GetMetric()
	invalidMetricTypeError := fmt.Errorf("input metric must be of type Histogram, ExponentialHistogram or Summary, got %s", metric.Type())

	aggTemp := getAggregationTemporality(metric)
	if aggTemp == pmetric.AggregationTemporalityUnspecified {
		return invalidMetricTypeError
	}

	countMetric := pmetric.NewMetric()
	countMetric.SetDescription(metric.Description())
	countMetric.SetName(metric.Name() + suffix)
	countMetric.SetUnit(metric.Unit())
	countMetric.SetEmptySum().SetAggregationTemporality(aggTemp)
	countMetric.Sum().SetIsMonotonic(monotonic)

	switch metric.Type() {
	case pmetric.MetricTypeHistogram:
		dataPoints := metric.Histogram().DataPoints()
		for i := 0; i < dataPoints.Len(); i++ {
			fillDataPoint(dataPoints.At(i), countMetric.Sum().DataPoints())
		}
	case pmetric.MetricTypeExponentialHistogram:
		dataPoints := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dataPoints.Len(); i++ {
			fillDataPoint(dataPoints.At(i), countMetric.Sum().DataPoints())
		}
	case pmetric.MetricTypeSummary:
		dataPoints := metric.Summary().DataPoints()
		for i := 0; i < dataPoints.Len(); i++ {
			fillDataPoint(dataPoints.At(i), countMetric.Sum().DataPoints())
		}
	default:
		return invalidMetricTypeError
	}

	if countMetric.Sum().DataPoints().Len() > 0 {
		countMetric.MoveTo(tCtx.GetMetrics().AppendEmpty())
	}

	return nil
}

func addCountDataPoint(dataPoint SumCountDataPoint, destination pmetric.NumberDataPointSlice) {
	newDp := destination.AppendEmpty()
	dataPoint.Attributes().CopyTo(newDp.Attributes())
	newDp.SetIntValue(int64(dataPoint.Count()))
	newDp.SetStartTimestamp(dataPoint.StartTimestamp())
	newDp.SetTimestamp(dataPoint.Timestamp())
}

func getAggregationTemporality(metric pmetric.Metric) pmetric.AggregationTemporality {
	switch metric.Type() {
	case pmetric.MetricTypeHistogram:
		return metric.Histogram().AggregationTemporality()
	case pmetric.MetricTypeExponentialHistogram:
		return metric.ExponentialHistogram().AggregationTemporality()
	case pmetric.MetricTypeSummary:
		// Summaries don't have an aggregation temporality, but they *should* be cumulative based on the Openmetrics spec.
		// This should become an optional argument once those are available in OTTL.
		return pmetric.AggregationTemporalityCumulative
	default:
		return pmetric.AggregationTemporalityUnspecified
	}
}
