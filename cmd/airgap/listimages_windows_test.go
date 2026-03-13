//go:build windows

// SPDX-FileCopyrightText: 2026 k0s authors
// SPDX-License-Identifier: Apache-2.0

package airgap_test

import (
	"strings"
	"testing"

	"github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirgapListImagesWindows(t *testing.T) {
	defaults := v1beta1.DefaultClusterImages()
	expectedImages := []string{
		defaults.Windows.Pause.URI(),
		defaults.Windows.KubeProxy.URI(),
		defaults.Calico.Windows.CNI.URI(),
		defaults.Calico.Windows.Node.URI(),
	}

	t.Run("DefaultConfig", func(t *testing.T) {
		underTest, out, err := newAirgapListImagesCmdWithConfig(t, "{}")

		require.NoError(t, underTest.Execute())
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")

		assert.Len(t, lines, len(expectedImages), "Expected exactly %d images", len(expectedImages))
		for _, img := range expectedImages {
			assert.Contains(t, lines, img)
		}
		assert.Empty(t, err.String())
	})

	t.Run("AllFlagIsNoOp", func(t *testing.T) {
		underTest, out, err := newAirgapListImagesCmdWithConfig(t, "{}", "--all")

		require.NoError(t, underTest.Execute())
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")

		// --all has no additional effect on Windows; same 4 images expected
		assert.Len(t, lines, len(expectedImages), "Expected exactly %d images with --all", len(expectedImages))
		assert.Empty(t, err.String())
	})
}
