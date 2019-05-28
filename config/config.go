package config

import (
	"github.com/StenaIT/kubecheck/checks"
	"github.com/StenaIT/kubecheck/hook"

	"github.com/gorilla/mux"
)

// OnHealthcheckStartedEvent represents the hook event OnHealthcheckStarted
const OnHealthcheckStartedEvent hook.Event = "OnHealthcheckStarted"

// OnHealthcheckCompletedEvent represents the hook event OnHealthcheckCompleted
const OnHealthcheckCompletedEvent hook.Event = "OnHealthcheckCompleted"

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
	Webhooks []hook.Webhook
}
