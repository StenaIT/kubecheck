package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/StenaIT/kubecheck/checks"
	"github.com/StenaIT/kubecheck/config"
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

	return healthchecks
}

func envOrDefault(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}