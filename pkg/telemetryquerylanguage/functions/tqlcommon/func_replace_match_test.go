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

package tqlcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/telemetryquerylanguage/tql/tqltest"
)

func Test_replaceMatch(t *testing.T) {
	input := pcommon.NewValueString("hello world")

	target := &tql.StandardGetSetter{
		Getter: func(ctx tql.TransformContext) interface{} {
			return ctx.GetItem().(pcommon.Value).StringVal()
		},
		Setter: func(ctx tql.TransformContext, val interface{}) {
			ctx.GetItem().(pcommon.Value).SetStringVal(val.(string))
		},
	}

	tests := []struct {
		name        string
		target      tql.GetSetter
		pattern     string
		replacement string
		want        func(pcommon.Value)
	}{
		{
			name:        "replace match",
			target:      target,
			pattern:     "hello*",
			replacement: "hello {universe}",
			want: func(expectedValue pcommon.Value) {
				expectedValue.SetStringVal("hello {universe}")
			},
		},
		{
			name:        "no match",
			target:      target,
			pattern:     "goodbye*",
			replacement: "goodbye {universe}",
			want: func(expectedValue pcommon.Value) {
				expectedValue.SetStringVal("hello world")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioValue := pcommon.NewValueString(input.StringVal())

			ctx := tqltest.NewTestTransformContext(
				t,
				scenarioValue,
			)

			exprFunc, _ := ReplaceMatch(tt.target, tt.pattern, tt.replacement)
			exprFunc(ctx)

			expected := pcommon.NewValueString("")
			tt.want(expected)

			assert.Equal(t, expected, scenarioValue)
		})
	}
}

func Test_replaceMatch_bad_input(t *testing.T) {
	input := pcommon.NewValueInt(1)
	ctx := tqltest.NewTestTransformContext(
		t,
		input,
	)

	target := &tql.StandardGetSetter{
		Getter: func(ctx tql.TransformContext) interface{} {
			return ctx.GetItem()
		},
		Setter: func(ctx tql.TransformContext, val interface{}) {
			t.Errorf("nothing should be set in this scenario")
		},
	}

	exprFunc, _ := ReplaceMatch(target, "*", "{replacement}")
	exprFunc(ctx)

	assert.Equal(t, pcommon.NewValueInt(1), input)
}

func Test_replaceMatch_get_nil(t *testing.T) {
	ctx := tqltest.NewTestTransformContext(
		t,
		nil,
	)

	target := &tql.StandardGetSetter{
		Getter: func(ctx tql.TransformContext) interface{} {
			return ctx.GetItem()
		},
		Setter: func(ctx tql.TransformContext, val interface{}) {
			t.Errorf("nothing should be set in this scenario")
		},
	}

	exprFunc, _ := ReplaceMatch(target, "*", "{anything}")
	exprFunc(ctx)
}
