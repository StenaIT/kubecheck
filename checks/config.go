package checks

// Config defines configuration for checks
type Config struct {
	Kubernetes KubernetesConfig
}

var config = Config{
	Kubernetes: KubernetesConfig{},
}

// Configure allows configuration of the checks package
func Configure(kubernetes KubernetesConfig) {
	config.Kubernetes = kubernetes
}
