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
	"os"
	"path/filepath"

	"github.com/k0sproject/k0s/internal/pkg/stringslice"
	ctrlpods "github.com/k0sproject/k0s/pkg/apis/ctrlpods.k0sproject.io/v1beta1"
	"github.com/k0sproject/k0s/pkg/component"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
)

var _ component.Component = &ControllerPods{}

type ControllerPods struct {
	controllerManager ctrl.Manager
}

func New(restConfig *rest.Config, dataDir string) (*ControllerPods, error) {

	controllerManager, err := ctrl.NewManager(restConfig, ctrl.Options{Namespace: "kube-system"})
	if err != nil {
		return nil, err
	}

	err = ctrlpods.AddToScheme(controllerManager.GetScheme())
	if err != nil {
		return nil, err
	}
	manifestDir := filepath.Join(dataDir, "kubelet", "manifests")
	err = ctrl.
		NewControllerManagedBy(controllerManager).
		For(&ctrlpods.ControllerPod{}).
		Complete(&controllerPodReconciler{
			Client:      controllerManager.GetClient(),
			manifestDir: manifestDir,
		})
	if err != nil {
		return nil, err
	}
	return &ControllerPods{
		controllerManager: controllerManager,
	}, nil
}

func (c *ControllerPods) Init() error {
	return nil
}

func (c *ControllerPods) Run(ctx context.Context) error {
	go func() {
		err := c.controllerManager.Start(ctx)
		if err != nil {
			logrus.Errorf("failed to start controller manager for controllerpods: %s", err.Error())
		}
	}()
	return nil
}

func (c *ControllerPods) Stop() error {
	return nil
}

func (c *ControllerPods) Healthy() error {
	return nil
}

type controllerPodReconciler struct {
	client.Client
	scheme *runtime.Scheme

	manifestDir string
}

const finalizer = "ctrlpod.k0sproject.io/finalizer"

func (r *controllerPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logrus.WithField("component", "controllerpod-reconciler").WithField("pod", req.NamespacedName)
	var ctrlPod ctrlpods.ControllerPod
	log.Debugf("reconciling %s", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, &ctrlPod); err != nil {
		log.Error(err, "unable to fetch ControllerPod")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if ctrlPod.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !stringslice.Contains(ctrlPod.GetFinalizers(), finalizer) {
			controllerutil.AddFinalizer(&ctrlPod, finalizer)
			if err := r.Update(ctx, &ctrlPod); err != nil {
				log.Error(err, "failed to add finalizer for controllerpod")
				return ctrl.Result{}, err
			}
		}
		// Write the file
		if err := r.writePodFile(&ctrlPod); err != nil {
			log.Error(err, "failed to write static pod manifest")
			return ctrl.Result{}, err
		}
		log.Info("successfully synced pod manifest")
	} else {
		// The object is being deleted
		if stringslice.Contains(ctrlPod.GetFinalizers(), finalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deletePodFile(&ctrlPod); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				log.Error(err, "failed to remove static pod manifest")
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			// This allows the API to actually delete the object
			controllerutil.RemoveFinalizer(&ctrlPod, finalizer)
			if err := r.Update(ctx, &ctrlPod); err != nil {
				log.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *controllerPodReconciler) writePodFile(ctrlPod *ctrlpods.ControllerPod) error {
	// Create the pod out of ctrl pod
	pod := corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:        ctrlPod.Name,
			Labels:      ctrlPod.Labels,
			Annotations: ctrlPod.Annotations,
		},
		Spec: ctrlPod.Spec,
	}
	// Dunno why these are not set from the object type when marshaling
	pod.APIVersion = corev1.SchemeGroupVersion.String()
	pod.Kind = "Pod"

	// Serialize to yaml
	data, err := yaml.Marshal(pod)
	if err != nil {
		return err
	}

	return os.WriteFile(r.filenameForPod(ctrlPod), data, 0755)
}

func (r *controllerPodReconciler) deletePodFile(ctrlPod *ctrlpods.ControllerPod) error {
	return os.Remove(r.filenameForPod(ctrlPod))
}

func (r *controllerPodReconciler) filenameForPod(ctrlPod *ctrlpods.ControllerPod) string {
	return filepath.Join(r.manifestDir, fmt.Sprintf("%s.yaml", ctrlPod.Name))
}
