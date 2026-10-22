// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl"
)

func newTestSlice(t *testing.T, raw []any) pcommon.Slice {
	t.Helper()
	s := pcommon.NewSlice()
	require.NoError(t, s.FromRaw(raw))
	return s
}

func sliceGetters(slices ...pcommon.Slice) []ottl.PSliceGetter[any] {
	getters := make([]ottl.PSliceGetter[any], 0, len(slices))
	for _, s := range slices {
		getters = append(getters, rawSliceGetter(s))
	}
	return getters
}

func rawSliceGetter(v any) ottl.PSliceGetter[any] {
	return ottl.StandardPSliceGetter[any]{
		Getter: func(context.Context, any) (any, error) { return v, nil },
	}
}

func Test_appendSlices(t *testing.T) {
	tests := []struct {
		name   string
		slices [][]any
		want   []any
	}{
		{
			name:   "two slices",
			slices: [][]any{{"a", "b"}, {"c"}},
			want:   []any{"a", "b", "c"},
		},
		{
			name:   "three slices",
			slices: [][]any{{"a"}, {"b"}, {"c", "d"}},
			want:   []any{"a", "b", "c", "d"},
		},
		{
			name:   "empty slices mixed with non-empty",
			slices: [][]any{{}, {"a"}, {}, {"b"}, {}},
			want:   []any{"a", "b"},
		},
		{
			name:   "all empty slices",
			slices: [][]any{{}, {}},
			want:   []any{},
		},
		{
			name:   "single slice",
			slices: [][]any{{"a", int64(1)}},
			want:   []any{"a", int64(1)},
		},
		{
			name:   "empty outer list",
			slices: [][]any{},
			want:   []any{},
		},
		{
			name: "mixed element types",
			slices: [][]any{
				{"s", int64(1), 1.5},
				{true, []byte{1, 2}, nil, map[string]any{"k": "v"}, []any{"x", int64(2)}},
			},
			want: []any{"s", int64(1), 1.5, true, []byte{1, 2}, nil, map[string]any{"k": "v"}, []any{"x", int64(2)}},
		},
		{
			name:   "nested slices are kept as single elements",
			slices: [][]any{{int64(1), []any{int64(2)}}, {int64(3)}},
			want:   []any{int64(1), []any{int64(2)}, int64(3)},
		},
		{
			name:   "maps",
			slices: [][]any{{map[string]any{"a": "1"}}, {map[string]any{"b": "2"}}},
			want:   []any{map[string]any{"a": "1"}, map[string]any{"b": "2"}},
		},
	}

	// Run with proto pooling both disabled and enabled, since the result must not share data with the inputs.
	for _, pooling := range []bool{false, true} {
		t.Run(fmt.Sprintf("pooling=%t", pooling), func(t *testing.T) {
			setProtoPoolingForTest(t, pooling)
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					inputs := make([]pcommon.Slice, 0, len(tt.slices))
					originals := make([][]any, 0, len(tt.slices))
					for _, raw := range tt.slices {
						s := newTestSlice(t, raw)
						inputs = append(inputs, s)
						originals = append(originals, s.AsRaw())
					}
					exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, sliceGetters(inputs...)))

					got, err := exprFunc(t.Context(), nil)
					require.NoError(t, err)
					gotSlice, ok := got.(pcommon.Slice)
					require.True(t, ok)
					assert.Equal(t, tt.want, gotSlice.AsRaw())

					for i, s := range inputs {
						assert.Equal(t, originals[i], s.AsRaw(), "input slice %d was modified", i)
					}
				})
			}
		})
	}
}

func Test_appendSlices_result_independent_of_inputs(t *testing.T) {
	for _, pooling := range []bool{false, true} {
		t.Run(fmt.Sprintf("pooling=%t", pooling), func(t *testing.T) {
			setProtoPoolingForTest(t, pooling)
			s := newTestSlice(t, []any{map[string]any{"x": "y"}, []any{"a"}, "plain"})
			original := s.AsRaw()
			exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, sliceGetters(s)))
			got, err := exprFunc(t.Context(), nil)
			require.NoError(t, err)
			result := got.(pcommon.Slice)

			// Modifying the inputs leaves the result unchanged.
			s.At(0).Map().PutStr("extra", "1")
			s.At(1).Slice().AppendEmpty().SetStr("b")
			s.At(2).SetStr("changed")
			s.AppendEmpty().SetStr("new")
			assert.Equal(t, original, result.AsRaw())

			// Modifying the result leaves the inputs unchanged.
			s2 := newTestSlice(t, []any{map[string]any{"x": "y"}, []any{"a"}})
			original2 := s2.AsRaw()
			exprFunc = appendSlices(xottl.NewTestingSliceGetter[any](true, sliceGetters(s2)))
			got, err = exprFunc(t.Context(), nil)
			require.NoError(t, err)
			result = got.(pcommon.Slice)
			result.At(0).Map().PutStr("extra", "1")
			result.At(1).Slice().AppendEmpty().SetStr("b")
			result.AppendEmpty().SetStr("new")
			assert.Equal(t, original2, s2.AsRaw())
		})
	}
}

func Test_appendSlices_same_slice_twice(t *testing.T) {
	for _, pooling := range []bool{false, true} {
		t.Run(fmt.Sprintf("pooling=%t", pooling), func(t *testing.T) {
			setProtoPoolingForTest(t, pooling)
			s := newTestSlice(t, []any{"a", map[string]any{"k": "v"}})
			original := s.AsRaw()
			exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, sliceGetters(s, s)))
			got, err := exprFunc(t.Context(), nil)
			require.NoError(t, err)
			assert.Equal(t, append(append([]any{}, original...), original...), got.(pcommon.Slice).AsRaw())
			assert.Equal(t, original, s.AsRaw())
		})
	}
}

func Test_appendSlices_typed_go_slice(t *testing.T) {
	s := newTestSlice(t, []any{"a"})
	getters := []ottl.PSliceGetter[any]{rawSliceGetter(s), rawSliceGetter([]string{"b", "c"}), rawSliceGetter([]int64{1})}
	exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, getters))
	got, err := exprFunc(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b", "c", int64(1)}, got.(pcommon.Slice).AsRaw())
}

func Test_appendSlices_dynamic_slice(t *testing.T) {
	s1 := newTestSlice(t, []any{"a", "b"})
	s2 := newTestSlice(t, []any{"c"})
	exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](false, sliceGetters(s1, s2)))
	got, err := exprFunc(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b", "c"}, got.(pcommon.Slice).AsRaw())
}

func Test_appendSlices_errors(t *testing.T) {
	good := newTestSlice(t, []any{"a"})

	t.Run("non-slice element", func(t *testing.T) {
		getters := []ottl.PSliceGetter[any]{sliceGetters(good)[0], rawSliceGetter("prod")}
		exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, getters))
		_, err := exprFunc(t.Context(), nil)
		require.Error(t, err)
	})

	t.Run("nil element", func(t *testing.T) {
		getters := []ottl.PSliceGetter[any]{sliceGetters(good)[0], rawSliceGetter(nil)}
		exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, getters))
		_, err := exprFunc(t.Context(), nil)
		require.Error(t, err)
	})

	t.Run("getter error", func(t *testing.T) {
		getters := []ottl.PSliceGetter[any]{ottl.StandardPSliceGetter[any]{
			Getter: func(context.Context, any) (any, error) { return nil, errors.New("boom") },
		}}
		exprFunc := appendSlices(xottl.NewTestingSliceGetter[any](true, getters))
		_, err := exprFunc(t.Context(), nil)
		assert.ErrorContains(t, err, "boom")
	})
}

func Test_AppendFactory(t *testing.T) {
	slices := xottl.NewTestingSliceGetter[any](true, sliceGetters(pcommon.NewSlice()))

	t.Run("factory creation", func(t *testing.T) {
		factory := NewAppendFactory[any]()
		assert.Equal(t, "Append", factory.Name())
	})

	t.Run("default arguments", func(t *testing.T) {
		factory := NewAppendFactory[any]()
		args := factory.CreateDefaultArguments()
		assert.IsType(t, &appendArguments[any]{}, args)
		assertArgumentFieldNames(t, args, []string{"Slices"})
	})

	t.Run("function creation", func(t *testing.T) {
		factory := NewAppendFactory[any]()
		args, ok := factory.CreateDefaultArguments().(*appendArguments[any])
		require.True(t, ok)
		args.Slices = *slices

		fn, err := factory.CreateFunction(ottl.FunctionContext{}, args)
		require.NoError(t, err)
		require.NotNil(t, fn)
		got, err := fn(t.Context(), nil)
		require.NoError(t, err)
		assert.Equal(t, []any{}, got.(pcommon.Slice).AsRaw())
	})

	t.Run("invalid arguments type", func(t *testing.T) {
		_, err := createAppendFunction[any](ottl.FunctionContext{}, "invalid args")
		assert.ErrorContains(t, err, "AppendFactory args must be of type *appendArguments[K]")
	})
}
