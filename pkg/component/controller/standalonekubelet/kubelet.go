/*
Copyright 2021 k0s authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package standalonekubelet

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/k0sproject/k0s/internal/pkg/dir"
	"github.com/k0sproject/k0s/internal/pkg/stringmap"
	"github.com/k0sproject/k0s/pkg/assets"
	"github.com/k0sproject/k0s/pkg/component"
	"github.com/k0sproject/k0s/pkg/constant"
	"github.com/k0sproject/k0s/pkg/supervisor"
	"github.com/sirupsen/logrus"
)

var _ component.Component = &Kubelet{}

var log *logrus.Entry = logrus.WithField("component", "standalone-kubelet")

type Kubelet struct {
	K0sVars    constant.CfgVars
	supervisor supervisor.Supervisor
}

// Init extracts the needed binaries
func (k *Kubelet) Init() error {
	cmds := []string{"kubelet", "xtables-legacy-multi"}

	for _, cmd := range cmds {
		err := assets.Stage(k.K0sVars.BinDir, cmd, constant.BinDirMode)
		if err != nil {
			return err
		}
	}

	if runtime.GOOS == "linux" {
		for _, symlink := range []string{"iptables-save", "iptables-restore", "ip6tables", "ip6tables-save", "ip6tables-restore"} {
			symlinkPath := filepath.Join(k.K0sVars.BinDir, symlink)

			// remove if it exist and ignore error if it doesn't
			_ = os.Remove(symlinkPath)

			err := os.Symlink("xtables-legacy-multi", symlinkPath)
			if err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", symlink, err)
			}
		}
	}

	dataDir := filepath.Join(k.K0sVars.DataDir, "kubelet", "manifests")
	err := dir.Init(dataDir, constant.DataDirMode)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dataDir, err)
	}

	return nil
}

// Run runs kubelet
func (k *Kubelet) Run(ctx context.Context) error {
	cmd := "kubelet"
	dataDir := filepath.Join(k.K0sVars.DataDir, "kubelet")
	manifestDir := filepath.Join(dataDir, "manifests")
	sockPath := path.Join(k.K0sVars.RunDir, "containerd.sock")

	// args for kubelet. by omitting --kubeconfig we set kubelet into "standalone" mode
	args := stringmap.StringMap{
		"--root-dir":                   dataDir,
		"--v":                          "4", // TODO Get this through the usual log mapping
		"--runtime-cgroups":            "/system.slice/containerd.service",
		"--pod-manifest-path":          manifestDir,
		"--address":                    "127.0.0.1", // So kubelet is not accessible from outside
		"--container-runtime":          "remote",
		"--container-runtime-endpoint": fmt.Sprintf("unix://%s", sockPath),
		"--containerd":                 sockPath,
	}

	log.Infof("starting kubelet with args: %v", args)
	k.supervisor = supervisor.Supervisor{
		Name:    cmd,
		BinPath: assets.BinPath(cmd, k.K0sVars.BinDir),
		RunDir:  k.K0sVars.RunDir,
		DataDir: k.K0sVars.DataDir,
		Args:    args.ToArgs(),
	}

	return k.supervisor.Supervise()
}

// Stop stops kubelet
func (k *Kubelet) Stop() error {
	log.Info("stopping")
	return k.supervisor.Stop()
}

// Health-check interface
func (k *Kubelet) Healthy() error {
	// client := http.DefaultClient
	// resp, err := client.Get("http://localhost:10248?verbose")
	// if err != nil {
	// 	return err
	// }
	// defer resp.Body.Close()
	// if resp.StatusCode != http.StatusOK {
	// 	body, err := io.ReadAll(resp.Body)
	// 	if err == nil {
	// 		logrus.Debugf("kubelet healthz output:\n %s", string(body))
	// 	}
	// 	return fmt.Errorf("expected 200 for api server ready check, got %d", resp.StatusCode)
	// }
	return nil
}
