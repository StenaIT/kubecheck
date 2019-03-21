package config

import (
	"github.com/StenaIT/kubecheck/checks"

	"github.com/gorilla/mux"
)

// Kubecheck defines the context for Kubecheck
type Kubecheck struct {
	Config       *KubecheckConfig
	Healthchecks []checks.Healthcheck
	Router       *mux.Router
}

// KubecheckConfig defines the configuration for Kubecheck
type KubecheckConfig struct {
	Debug    bool
	LogLevel string
}
