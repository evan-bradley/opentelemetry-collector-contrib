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

package tql // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

type TransformContext interface {
	GetItem() interface{}
	GetInstrumentationScope() pcommon.InstrumentationScope
	GetResource() pcommon.Resource
	GetProcessorContext() ProcessorContext
}

// ProcessorContext is a basic interface for providing runtime dependencies from a processor to TQL functions.
type ProcessorContext interface {
	GetLogger() *zap.Logger
}

// ProcessorContextData is a concrete implementation to the ProcessorContext interface.
type ProcessorContextData struct {
	Logger *zap.Logger
}

func (pcd ProcessorContextData) GetLogger() *zap.Logger {
	return pcd.Logger
}

type ExprFunc func(ctx TransformContext) interface{}

type Enum int64

type Getter interface {
	Get(ctx TransformContext) interface{}
}

type Setter interface {
	Set(ctx TransformContext, val interface{})
}

type GetSetter interface {
	Getter
	Setter
}

type StandardGetSetter struct {
	Getter func(ctx TransformContext) interface{}
	Setter func(ctx TransformContext, val interface{})
}

func (path StandardGetSetter) Get(ctx TransformContext) interface{} {
	return path.Getter(ctx)
}

func (path StandardGetSetter) Set(ctx TransformContext, val interface{}) {
	path.Setter(ctx, val)
}

type Literal struct {
	Value interface{}
}

func (l Literal) Get(ctx TransformContext) interface{} {
	return l.Value
}

type exprGetter struct {
	expr ExprFunc
}

func (g exprGetter) Get(ctx TransformContext) interface{} {
	return g.expr(ctx)
}

func NewGetter(val Value, functions map[string]interface{}, pathParser PathExpressionParser, enumParser EnumParser) (Getter, error) {
	if val.IsNil != nil && *val.IsNil {
		return &Literal{Value: nil}, nil
	}

	if s := val.String; s != nil {
		return &Literal{Value: *s}, nil
	}
	if f := val.Float; f != nil {
		return &Literal{Value: *f}, nil
	}
	if i := val.Int; i != nil {
		return &Literal{Value: *i}, nil
	}
	if b := val.Bool; b != nil {
		return &Literal{Value: bool(*b)}, nil
	}
	if b := val.Bytes; b != nil {
		return &Literal{Value: ([]byte)(*b)}, nil
	}

	if val.Enum != nil {
		enum, err := enumParser(val.Enum)
		if err != nil {
			return nil, err
		}
		return &Literal{Value: int64(*enum)}, nil
	}

	if val.Path != nil {
		return pathParser(val.Path)
	}

	if val.Invocation == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, fmt.Errorf("no value field set. This is a bug in the Telemetry Query Language")
	}
	call, err := NewFunctionCall(*val.Invocation, functions, pathParser, enumParser)
	if err != nil {
		return nil, err
	}
	return &exprGetter{
		expr: call,
	}, nil
}
