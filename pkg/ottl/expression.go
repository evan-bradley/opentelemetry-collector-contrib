// Copyright The OpenTelemetry Authors
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

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
)

type ExprFunc[K any] func(ctx context.Context, tCtx K) (interface{}, error)

type Expr[K any] struct {
	exprFunc ExprFunc[K]
}

func (e Expr[K]) Eval(ctx context.Context, tCtx K) (interface{}, error) {
	return e.exprFunc(ctx, tCtx)
}

type Getter[K any] interface {
	Get(ctx context.Context, tCtx K) (interface{}, error)
}

type Setter[K any] interface {
	Set(ctx context.Context, tCtx K, val interface{}) error
}

type GetSetter[K any] interface {
	Getter[K]
	Setter[K]
}

type StandardGetSetter[K any] struct {
	Getter func(ctx context.Context, tCx K) (interface{}, error)
	Setter func(ctx context.Context, tCx K, val interface{}) error
}

func (path StandardGetSetter[K]) Get(ctx context.Context, tCx K) (interface{}, error) {
	return path.Getter(ctx, tCx)
}

func (path StandardGetSetter[K]) Set(ctx context.Context, tCx K, val interface{}) error {
	return path.Setter(ctx, tCx, val)
}

type literal[K any] struct {
	value interface{}
}

func (l literal[K]) Get(context.Context, K) (interface{}, error) {
	return l.value, nil
}

type exprGetter[K any] struct {
	expr Expr[K]
}

func (g exprGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return g.expr.Eval(ctx, tCtx)
}

func (p *Parser[K]) newGetter(val value) (Getter[K], error) {
	if val.IsNil != nil && *val.IsNil {
		return &literal[K]{value: nil}, nil
	}

	if s := val.String; s != nil {
		return &literal[K]{value: *s}, nil
	}
	if b := val.Bool; b != nil {
		return &literal[K]{value: bool(*b)}, nil
	}
	if b := val.Bytes; b != nil {
		return &literal[K]{value: ([]byte)(*b)}, nil
	}

	if val.Enum != nil {
		enum, err := p.enumParser(val.Enum)
		if err != nil {
			return nil, err
		}
		return &literal[K]{value: int64(*enum)}, nil
	}

	if eL := val.Literal; eL != nil {
		if f := eL.Float; f != nil {
			return &literal[K]{value: *f}, nil
		}
		if i := eL.Int; i != nil {
			return &literal[K]{value: *i}, nil
		}
		if eL.Path != nil {
			return p.pathParser(eL.Path)
		}
		if eL.Invocation != nil {
			call, err := p.newFunctionCall(*eL.Invocation)
			if err != nil {
				return nil, err
			}
			return &exprGetter[K]{
				expr: call,
			}, nil
		}
	}

	if val.List != nil {
		list := make([]any, len(val.List.Values))
		for i, v := range val.List.Values {
			getter, err := p.newGetter(v)
			if err != nil {
				return nil, err
			}
			literal, ok := getter.(*literal[K])
			if !ok {
				return nil, errors.New("a non-literal value was used in a Getter list literal")
			}
			list[i] = literal.value
		}
		typedList, err := convertSliceType(list)
		if err != nil {
			return nil, fmt.Errorf("unable to create slice for Getter list literal: %w", err)
		}

		return &literal[K]{value: typedList}, nil
	}

	if val.MathExpression == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, fmt.Errorf("no value field set. This is a bug in the OpenTelemetry Transformation Language")
	}
	return p.evaluateMathExpression(val.MathExpression)
}

// convertSliceType converts a slice with potentially heterogeneous
// types to a slice with a defined type. If the slice does have
// heterogeneous types, an error is returned.
//
// Most arrays are intended to be homogeneous according to the
// OpenTelemetry specification, so we apply the same restriction here.
func convertSliceType(untypedSlice []any) (any, error) {
	if len(untypedSlice) == 0 {
		return untypedSlice, nil
	}

	switch untypedSlice[0].(type) {
	case string:
		return convertSlice[string](untypedSlice)
	case int64:
		return convertSlice[int64](untypedSlice)
	case float64:
		return convertSlice[float64](untypedSlice)
	case bool:
		return convertSlice[bool](untypedSlice)
	case []byte:
		return convertSlice[[]byte](untypedSlice)
	default:
		return nil, errors.New("unsupported list type")
	}
}

func convertSlice[K any](inputSlice []any) ([]K, error) {
	outputSlice := make([]K, len(inputSlice))
	for i, val := range inputSlice {
		typedVal, ok := val.(K)

		if !ok {
			return nil, errors.New("elements of multiple types exist within the list")
		}

		outputSlice[i] = typedVal
	}

	return outputSlice, nil
}
