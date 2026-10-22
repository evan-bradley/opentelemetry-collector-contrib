// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"
)

type appendArguments[K any] struct {
	Slices xottl.SliceGetter[K, ottl.PSliceGetter[K]]
}

// NewAppendFactory returns a factory for the Append OTTL function.
// See https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/xottl/ottlfuncs/README.md#append
//
// The function requires the pkg.ottl.functions.enableExperimental feature gate to be enabled.
func NewAppendFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Append", &appendArguments[K]{}, createAppendFunction[K], ottl.WithExperimental[K]())
}

func createAppendFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*appendArguments[K])
	if !ok {
		return nil, errors.New("AppendFactory args must be of type *appendArguments[K]")
	}
	return appendSlices(&args.Slices), nil
}

func appendSlices[K any](slices *xottl.SliceGetter[K, ottl.PSliceGetter[K]]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		var resolved []pcommon.Slice
		total := 0
		var getErr error
		_, err := slices.Range(ctx, tCtx, func(getter ottl.PSliceGetter[K]) bool {
			s, err := getter.Get(ctx, tCtx)
			if err != nil {
				getErr = err
				return false
			}
			resolved = append(resolved, s)
			total += s.Len()
			return true
		})
		if err != nil {
			return nil, err
		}
		if getErr != nil {
			return nil, getErr
		}

		result := pcommon.NewSlice()
		result.EnsureCapacity(total)
		for _, s := range resolved {
			for _, v := range s.All() {
				v.CopyTo(result.AppendEmpty())
			}
		}
		return result, nil
	}
}
