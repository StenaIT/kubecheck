package checks

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset

// KubernetesConfig defines the configuration for Kubernetes
type KubernetesConfig struct {
	InClusterConfig bool
}

func getKubernetesClientset() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}

	var err error
	var restConfig *rest.Config

	if config.Kubernetes.InClusterConfig {
		restConfig, err = rest.InClusterConfig()
	} else {
		kubeconfig := filepath.Join(homeDir(), ".kube", "config")
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		panic(err.Error())
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
