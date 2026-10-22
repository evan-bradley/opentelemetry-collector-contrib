// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
)

// setProtoPoolingForTest sets pdata's pdata.useProtoPooling feature gate for the duration of the test.
// pdata does not export the gate, so it is looked up by ID.
func setProtoPoolingForTest(t *testing.T, enabled bool) {
	t.Helper()
	var gate *featuregate.Gate
	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		if g.ID() == "pdata.useProtoPooling" {
			gate = g
		}
	})
	require.NotNil(t, gate, "pdata.useProtoPooling feature gate is not registered")
	t.Cleanup(testutil.SetFeatureGateForTest(t, gate, enabled))
}
