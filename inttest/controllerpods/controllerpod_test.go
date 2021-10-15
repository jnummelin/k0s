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
package controllerpods

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/k0sproject/k0s/inttest/common"
)

var ctrlPod = `
apiVersion: ctrlpods.k0sproject.io/v1beta1
kind: ControllerPod
metadata:
  name: nginx
  namespace: kube-system
  labels:
    name: nginx
spec:
  containers:
  - name: nginx
    image: docker.io/nginx:1-alpine
    resources:
      limits:
        memory: "128Mi"
        cpu: "500m"
    ports:
      - containerPort: 80

`

type ControllerPodSuite struct {
	common.FootlooseSuite
}

func (s *ControllerPodSuite) TestK0sRunsStaticPods() {
	s.NoError(s.InitController(0, "--enable-static-pods"))

	token, err := s.GetJoinToken("worker")
	s.NoError(err)
	s.NoError(s.RunWorkersWithToken(token))

	// kc, err := s.KubeClient(s.ControllerNode(0))
	// s.NoError(err)

	// err = s.WaitForNodeReady(s.WorkerNode(0), kc)
	// s.NoError(err)
	// We've got enough confidence that everything is actually operating properly at this point
	// Let's push a ControllerPod object to API
	ssh, err := s.SSH(s.ControllerNode(0))
	s.Require().NoError(err)
	defer ssh.Disconnect()

	_, err = ssh.ExecWithOutput("mkdir -p /var/lib/k0s/manifests/static-nginx/")
	s.Require().NoError(err)
	s.PutFile(s.ControllerNode(0), "/var/lib/k0s/manifests/static-nginx/nginx.yaml", ctrlPod)

	s.NoError(<-s.waitToSeeNginxRunning())

}

func TestControllerPodSuite(t *testing.T) {
	s := ControllerPodSuite{
		common.FootlooseSuite{
			ControllerCount: 1,
			// WorkerCount:     1,
		},
	}
	suite.Run(t, &s)
}

func (s *ControllerPodSuite) waitToSeeNginxRunning() <-chan error {
	resultChan := make(chan error)
	// Start to poll controller for nginx process
	go func() {
		defer close(resultChan)
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		select {
		case <-ticker.C:
			if s.isProcessRunning("nginx") {
				resultChan <- nil
			}
		case <-timeoutCtx.Done():
			resultChan <- fmt.Errorf("timeout waiting to see nginx process on controller")
			return
		}

	}()
	return resultChan
}

func (s *ControllerPodSuite) isProcessRunning(name string) bool {
	ssh, err := s.SSH(s.ControllerNode(0))
	if err != nil {
		return false
	}
	defer ssh.Disconnect()
	_, err = ssh.ExecWithOutput(fmt.Sprintf("pidof %s", name))
	return err == nil
}
