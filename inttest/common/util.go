package common

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// WaitForCalicoReady waits to see all calico pods healthy
func WaitForCalicoReady(kc *kubernetes.Clientset) error {
	return wait.PollImmediate(100*time.Millisecond, 5*time.Minute, func() (done bool, err error) {
		ds, err := kc.AppsV1().DaemonSets("kube-system").Get(context.TODO(), "calico-node", v1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled, nil
	})
}

func WaitForMetricsReady(c *rest.Config) error {
	apiServiceClientset, err := clientset.NewForConfig(c)
	if err != nil {
		return err
	}

	return wait.PollImmediate(100*time.Millisecond, 5*time.Minute, func() (done bool, err error) {
		apiService, err := apiServiceClientset.ApiregistrationV1().APIServices().Get(context.TODO(), "v1beta1.metrics.k8s.io", v1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, c := range apiService.Status.Conditions {
			if c.Type == "Available" && c.Status == "True" {
				return true, nil
			}
		}

		return false, nil
	})
}
