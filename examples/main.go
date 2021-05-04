package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/StenaIT/kubecheck/checks"
	"github.com/StenaIT/kubecheck/config"
	"github.com/StenaIT/kubecheck/hook"
	"github.com/StenaIT/kubecheck/server"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
)

func main() {
	kubecheck := configureKubecheck()
	server.New(kubecheck).ListenAndServe()
}

func configureKubecheck() *config.Kubecheck {
	debug, _ := strconv.ParseBool(envOrDefault("KUBECHECK_DEBUG", "true"))
	logLevel := envOrDefault("KUBECHECK_LOG_LEVEL", "info")
	k8sInClusterConfig, _ := strconv.ParseBool(envOrDefault("KUBECHECK_K8S_INCLUSTERCONFIG", "false"))

	kubecheck := &config.Kubecheck{
		Config: &config.KubecheckConfig{
			Debug:    debug,
			LogLevel: logLevel,
			Webhooks: []hook.Webhook{
				hook.Webhook{
					Name:   "healthchecks-io",
					URL:    "https://hc-ping.com/b68522d5-eb89-44a9-8335-7f668f1aa691",
					Events: []hook.Event{config.OnHealthcheckCompletedEvent},
				},
				// hook.Webhook{
				// 	Name:   "slack-on-completed",
				// 	URL:    "<SLACK_WEBHOOK_URL>",
				// 	Data:   `{"username": "Kubecheck", "text": "Healthchecks ran to completion"}`,
				// 	Events: []hook.Event{config.OnHealthcheckCompletedEvent},
				// },
			},
		},
		Healthchecks: configureHealthchecks(),
		Router:       nil,
	}

	checks.Configure(checks.KubernetesConfig{
		InClusterConfig: k8sInClusterConfig,
	})

	log.SetLevelFromString(kubecheck.Config.LogLevel)
	log.SetHandler(text.New(os.Stdout))

	return kubecheck
}

func configureHealthchecks() []checks.Healthcheck {
	healthchecks := make([]checks.Healthcheck, 0)

	healthchecks = append(healthchecks, checks.RandomFailHealthcheck{
		Name:        "random-failure",
		Description: "Randomly fails at the given failure rate. Usually used for debugging alarms. Failures may be ignored!",
		FailRate:    10,
	})

	healthchecks = append(healthchecks, checks.HTTPGetHealthcheck{
		Name:        "http-get",
		Description: "Performs a HTTP GET request",
		URL:         "https://www.google.com/",
	}.WithExpectations(
		checks.ExpectStatusCode(200),
		checks.ExpectBodyContains("Google"),
		checks.ExpectHeader("content-type", "text/html; charset=ISO-8859-1"),
		checks.ExpectValidCertificate(7),
	))

	healthchecks = append(healthchecks, checks.DNSLookupHealthcheck{
		Name:        "dns-lookup-google-com",
		Description: "Performs a DNS lookup to verify that domain names can be resolved",
		Host:        "google.com",
	})

	healthchecks = append(healthchecks, checks.KubernetesNodeHealthcheck{
		Name:        "kubernetes-node-health",
		Description: "Performs kubernetes node healthchecks",
	}.WithExpectations(
		checks.ExpectNodeCountRange(2, 6),
		checks.ExpectNodeStatusOK(10*time.Minute),
	))

	healthchecks = append(healthchecks, checks.NewKubernetesTraefikHealthcheck(
		"kubernetes-traefik-health",
		checks.KubernetesTraefikConfig{
			Namespace:       "kube-system",
			DaemonSetName:   "traefik-ingress",
			ServiceName:     "traefik",
			ServicePortName: "web",
		},
	))

	healthchecks = append(healthchecks, checks.KubernetesPodAntiAffinityHealthcheck{
		Name:        "kubernetes-pod-anti-affinity-health",
		Description: "Performs kubernetes pod anti-affinity healthchecks",
	}.WithExpectations(
		checks.ExpectNodeSpread(2),
	))

	return healthchecks
}

func envOrDefault(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}
