//go:build windows

// SPDX-FileCopyrightText: 2026 k0s authors
// SPDX-License-Identifier: Apache-2.0

package airgap

import (
	"github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
)

// GetImageURIs returns all image tags for Windows worker nodes
func GetImageURIs(spec *v1beta1.ClusterSpec, _ bool) []string {
	return []string{
		spec.Images.Windows.Pause.URI(),
		spec.Images.Windows.KubeProxy.URI(),
		spec.Images.Calico.Windows.CNI.URI(),
		spec.Images.Calico.Windows.Node.URI(),
	}
}
