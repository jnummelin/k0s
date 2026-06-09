// SPDX-FileCopyrightText: 2026 k0s authors
// SPDX-License-Identifier: Apache-2.0

package airgap_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/k0sproject/k0s/cmd"

	"github.com/spf13/cobra"

	"github.com/stretchr/testify/require"
)

func newAirgapListImagesCmdWithConfig(t *testing.T, config string, args ...string) (_ *cobra.Command, out, err *strings.Builder) {
	configFile := filepath.Join(t.TempDir(), "k0s.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(config), 0644))

	out, err = new(strings.Builder), new(strings.Builder)
	cmd := cmd.NewRootCmd()
	cmd.SetArgs(append([]string{"airgap", "--config=" + configFile, "list-images"}, args...))
	cmd.SetIn(iotest.ErrReader(errors.New("unexpected read from standard input")))
	cmd.SetOut(out)
	cmd.SetErr(err)
	return cmd, out, err
}
