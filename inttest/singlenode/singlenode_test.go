/*
Copyright 2020 k0s authors

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

package singlenode

import (
	"fmt"

	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/k0sproject/k0s/inttest/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const k0sPartialConfig = `
apiVersion: k0s.k0sproject.io/v1beta1
kind: ClusterConfig
metadata:
  name: test-cluster
spec:
  api:
    sans:
    - 1.2.3.4
`

type SingleNodeSuite struct {
	common.BootlooseSuite
}

func (s *SingleNodeSuite) TestK0sGetsUp() {
	s.PutFile(s.ControllerNode(0), "/tmp/k0s.yaml", k0sPartialConfig)
	s.NoError(s.InitController(0, "--single", "--config=/tmp/k0s.yaml"))

	kc, err := s.KubeClient(s.ControllerNode(0))
	s.Require().NoError(err)

	err = s.WaitForNodeReady(s.ControllerNode(0), kc)
	s.NoError(err)

	s.T().Run("kubeconfig", func(t *testing.T) {
		ssh, err := s.SSH(s.Context(), s.ControllerNode(0))
		require.NoError(t, err, "failed to SSH into controller")
		defer ssh.Disconnect()

		out, err := ssh.ExecWithOutput(s.Context(), fmt.Sprintf("'%s' kubeconfig create foo", s.K0sFullPath))
		require.NoError(t, err, "failed to create kubeconfig")
		// Parse the kubeconfig
		kc, err := clientcmd.Load([]byte(out))
		require.NoError(t, err)
		// Assert that cluster and context name is propagated correctly
		assert.Equal(t, "test-cluster", kc.CurrentContext)
		assert.Equal(t, "test-cluster", kc.Contexts["test-cluster"].Cluster)
		assert.NotEmpty(t, kc.Clusters["test-cluster"].Server)

	})

	s.AssertSomeKubeSystemPods(kc)

	s.T().Log("waiting to see CNI pods ready")
	s.NoError(common.WaitForKubeRouterReady(s.Context(), kc), "CNI did not start")

	s.T().Run("verify", func(t *testing.T) {
		ssh, err := s.SSH(s.Context(), s.ControllerNode(0))
		require.NoError(t, err, "failed to SSH into controller")
		defer ssh.Disconnect()

		t.Run(("kineIsDefaultStorage"), func(t *testing.T) {
			_, err = ssh.ExecWithOutput(s.Context(), "test -e /var/lib/k0s/bin/kine && ps xa | grep kine")
			assert.NoError(t, err)
		})

		t.Run(("noControllerJoinTokens"), func(t *testing.T) {
			noToken, err := ssh.ExecWithOutput(s.Context(), fmt.Sprintf("'%s' token create --role=controller", s.K0sFullPath))
			assert.Error(t, err)
			assert.Equal(t, "Error: refusing to create token: cannot join into a single node cluster", noToken)
		})

		t.Run(("noWorkerJoinTokens"), func(t *testing.T) {
			noToken, err := ssh.ExecWithOutput(s.Context(), fmt.Sprintf("'%s' token create --role=worker", s.K0sFullPath))
			assert.Error(t, err)
			assert.Equal(t, "Error: refusing to create token: cannot join into a single node cluster", noToken)
		})

		t.Run("leader election disabled for scheduler", func(t *testing.T) {
			_, err := kc.CoordinationV1().Leases("kube-system").Get(s.Context(), "kube-scheduler", v1.GetOptions{})
			assert.Error(t, err)
			assert.True(t, apierrors.IsNotFound(err))
		})

		t.Run("leader election disabled for controller manager", func(t *testing.T) {
			_, err := kc.CoordinationV1().Leases("kube-system").Get(s.Context(), "kube-controller-manager", v1.GetOptions{})
			assert.Error(t, err)
			assert.True(t, apierrors.IsNotFound(err))
		})

		// test with etcd backend in config
		t.Run(("killK0s"), func(t *testing.T) {
			_, err = ssh.ExecWithOutput(s.Context(), "kill $(pidof k0s) && while pidof k0s; do sleep 0.1s; done")
			assert.NoError(t, err)
		})

		s.PutFile(s.ControllerNode(0), "/tmp/k0s.yaml", k0sConfig)
		require.NoError(t, err, "failed to upload k0s.yaml")

		s.NoError(s.InitController(0, "--single", "--config=/tmp/k0s.yaml"))

		t.Run(("etcdIsRunning"), func(t *testing.T) {
			_, err = ssh.ExecWithOutput(s.Context(), "test -e /var/lib/k0s/bin/etcd && ps xa | grep etcd")
			assert.NoError(t, err)
		})
	})
}

const k0sConfig = `
spec:
  storage:
    type: etcd
`

func TestSingleNodeSuite(t *testing.T) {
	s := SingleNodeSuite{
		common.BootlooseSuite{
			ControllerCount: 1,
			ControllerUmask: 027,
		},
	}
	suite.Run(t, &s)
}
