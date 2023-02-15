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
	"fmt"
)

type ExprFunc[K any] func(ctx context.Context, tCtx K) (interface{}, error)

type ExprI[K any] interface {
	ExprFunc() ExprFunc[K]
	Eval(ctx context.Context, tCtx K) (interface{}, error)
}

type Expr[K any] struct {
	exprFunc ExprFunc[K]
}

func (e Expr[K]) Eval(ctx context.Context, tCtx K) (interface{}, error) {
	return e.exprFunc(ctx, tCtx)
}

func (e Expr[K]) ExprFunc() ExprFunc[K] {
	return e.exprFunc
}

type StaticExpr[K any] struct {
	exprFunc ExprFunc[K]
}

func (e StaticExpr[K]) Eval(ctx context.Context, tCtx K) (interface{}, error) {
	return e.exprFunc(ctx, tCtx)
}

func (e StaticExpr[K]) ExprFunc() ExprFunc[K] {
	return e.exprFunc
}

func (e StaticExpr[K]) GetStatic() (any, error) {
	var x K
	res, err := e.exprFunc(context.Background(), x)
	if err != nil {
		return nil, err
	}
	return res, nil
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
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
	Setter func(ctx context.Context, tCtx K, val interface{}) error
}

func (path StandardGetSetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return path.Getter(ctx, tCtx)
}

func (path StandardGetSetter[K]) Set(ctx context.Context, tCtx K, val interface{}) error {
	return path.Setter(ctx, tCtx, val)
}

type StaticGetter interface {
	GetStatic() (any, error)
}

type literal[K any] struct {
	value interface{}
}

func (l literal[K]) Get(context.Context, K) (interface{}, error) {
	return l.value, nil
}

func (l literal[K]) GetStatic() (any, error) {
	return l.value, nil
}

type exprGetter[K any] struct {
	expr ExprI[K]
}

func (g exprGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return g.expr.Eval(ctx, tCtx)
}

type listGetter[K any] struct {
	slice []Getter[K]
}

func (l *listGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	evaluated := make([]any, len(l.slice))

	for i, v := range l.slice {
		val, err := v.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		evaluated[i] = val
	}

	return evaluated, nil
}

type StringGetter[K any] interface {
	Get(ctx context.Context, tCtx K) (*string, error)
}

type IntGetter[K any] interface {
	Get(ctx context.Context, tCtx K) (*int64, error)
}

type StandardTypeGetter[K any, T any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

type StaticTypeGetter[K any, T any] struct {
	value *T
}

func NewStandardTypeGetter[K any, T any](getter Getter[K]) (any, error) {
	if sg, ok := getter.(StaticGetter); ok {
		val, err := sg.GetStatic()
		if err != nil {
			return StandardTypeGetter[K, T]{}, err
		}
		if val == nil {
			return StaticTypeGetter[K, T]{value: nil}, nil
		}
		v, ok := val.(T)
		if !ok {
			return nil, fmt.Errorf("expected %T but got %T", v, val)
		}
		return StaticTypeGetter[K, T]{value: &v}, nil
	}

	return StandardTypeGetter[K, T]{Getter: getter.Get}, nil
}

func (g StandardTypeGetter[K, T]) Get(ctx context.Context, tCtx K) (*T, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	v, ok := val.(T)
	if !ok {
		return nil, fmt.Errorf("expected %T but got %T", v, val)
	}
	return &v, nil
}

func (g StaticTypeGetter[K, T]) Get(context.Context, K) (*T, error) {
	return g.value, nil
}

func (g StaticTypeGetter[K, T]) GetStatic() (any, error) {
	return g.value, nil
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
		if eL.Converter != nil {
			call, err := p.newFunctionCall(invocation{
				Function:  eL.Converter.Function,
				Arguments: eL.Converter.Arguments,
			})
			if err != nil {
				return nil, err
			}
			if sc, ok := call.(StaticGetter); ok {
				res, err := sc.GetStatic()
				if err != nil {
					return nil, err
				}
				return &literal[K]{
					value: res,
				}, nil
			}
			return &exprGetter[K]{
				expr: call,
			}, nil
		}
	}

	if val.List != nil {
		lg := listGetter[K]{slice: make([]Getter[K], len(val.List.Values))}
		for i, v := range val.List.Values {
			getter, err := p.newGetter(v)
			if err != nil {
				return nil, err
			}
			lg.slice[i] = getter
		}
		return &lg, nil
	}

	if val.MathExpression == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, fmt.Errorf("no value field set. This is a bug in the OpenTelemetry Transformation Language")
	}
	return p.evaluateMathExpression(val.MathExpression)
}
