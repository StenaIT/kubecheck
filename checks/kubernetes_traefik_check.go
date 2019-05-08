package checks

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/StenaIT/kubecheck/http"

	"github.com/apex/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesTraefikHealthcheck defines a healthcheck that can be executed
type KubernetesTraefikHealthcheck struct {
	Name        string
	Description string
	Config      KubernetesTraefikConfig
	HealthcheckExpectations
}

// KubernetesTraefikConfig defines the configuration for the healthcheck
type KubernetesTraefikConfig struct {
	Namespace       string        `json:"namespace"`
	DaemonSetName   string        `json:"daemonsetName"`
	ServiceName     string        `json:"serviceName"`
	ServicePortName string        `json:"servicePort"`
	GracePeriod     time.Duration `json:"gracePeriod"`
}

// KubernetesTraefikExpectation defines the expectation interface
type KubernetesTraefikExpectation interface {
	Verify(context KubernetesTraefikExpectationContext) []*AssertionGroup
}

// KubernetesTraefikExpectationContext defines the context object for expectations
type KubernetesTraefikExpectationContext struct {
	Config    KubernetesTraefikConfig
	DaemonSet *appsv1.DaemonSet
	Service   *corev1.Service
	Endpoints *corev1.Endpoints
}

// KubernetesTraefikDaemonsetExpectation defines expectations using the traefik daemonset
type KubernetesTraefikDaemonsetExpectation struct{}

// KubernetesTraefikServiceEndpointExpectation defines expectations using the traefik service
type KubernetesTraefikServiceEndpointExpectation struct{}

// NewKubernetesTraefikHealthcheck creates the healthcheck
func NewKubernetesTraefikHealthcheck(name string, config KubernetesTraefikConfig) KubernetesTraefikHealthcheck {
	dse := KubernetesTraefikDaemonsetExpectation{}
	sse := KubernetesTraefikServiceEndpointExpectation{}

	if config.GracePeriod == time.Duration(0) {
		config.GracePeriod = time.Duration(10 * time.Minute)
	}

	return KubernetesTraefikHealthcheck{
		Name:        name,
		Description: "Performs Traefik healthchecks",
		Config:      config,
		HealthcheckExpectations: HealthcheckExpectations{
			Expectations: []interface{}{dse, sse},
		},
	}
}

// Execute runs the healthcheck
func (c KubernetesTraefikHealthcheck) Execute() Result {
	clientset := getKubernetesClientset()

	daemonset, err := clientset.AppsV1().DaemonSets(c.Config.Namespace).Get(c.Config.DaemonSetName, metav1.GetOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	service, err := clientset.CoreV1().Services(c.Config.Namespace).Get(c.Config.ServiceName, metav1.GetOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	endpoints, err := clientset.CoreV1().Endpoints(c.Config.Namespace).Get(c.Config.ServiceName, metav1.GetOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	context := KubernetesTraefikExpectationContext{
		DaemonSet: daemonset,
		Service:   service,
		Endpoints: endpoints,
		Config:    c.Config,
	}

	return c.VerifyExpectation(c.Config, func(expectation interface{}) []*AssertionGroup {
		return expectation.(KubernetesTraefikExpectation).Verify(context)
	})
}

// Describe returns the description of the healthcheck
func (c KubernetesTraefikHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// Verify is called to verify expectations
func (a KubernetesTraefikDaemonsetExpectation) Verify(context KubernetesTraefikExpectationContext) []*AssertionGroup {
	daemonset := context.DaemonSet

	ag := NewAssertionGroup("DaemonSet", daemonset.Name)

	gracePeriodEnd := daemonset.GetCreationTimestamp().Add(context.Config.GracePeriod)
	if time.Since(gracePeriodEnd) > 0 {
		status := daemonset.Status
		ag.AssertTrue("CurrentNumberScheduled", status.CurrentNumberScheduled == status.DesiredNumberScheduled, status.DesiredNumberScheduled, status.CurrentNumberScheduled)
		ag.AssertTrue("NumberMisscheduled", status.NumberMisscheduled == 0, 0, status.NumberMisscheduled)
		ag.AssertTrue("DesiredNumberScheduled", status.DesiredNumberScheduled >= 2, ">=2", status.DesiredNumberScheduled)
		ag.AssertTrue("NumberReady", status.NumberReady >= 2, ">=2", status.NumberReady)
		ag.AssertTrue("NumberAvailable", status.NumberAvailable >= 2, ">=2", status.NumberAvailable)
		ag.AssertTrue("NumberUnavailable", status.NumberUnavailable == 0, 0, status.NumberUnavailable)
		ag.AssertTrue("UpdatedNumberScheduled", status.UpdatedNumberScheduled == status.DesiredNumberScheduled, status.DesiredNumberScheduled, status.UpdatedNumberScheduled)
	}

	return []*AssertionGroup{ag}
}

// Verify is called to verify expectations
func (e KubernetesTraefikServiceEndpointExpectation) Verify(context KubernetesTraefikExpectationContext) []*AssertionGroup {
	ag := NewAssertionGroup("ServiceEndpoints", context.Service.Name)

	gracePeriodEnd := context.Service.GetCreationTimestamp().Add(context.Config.GracePeriod)
	if time.Since(gracePeriodEnd) > 0 {
		servers := getEndpoints(context.Config, context.Endpoints)

		pingsOK := int64(0)
		sumPingTime := int64(0)

		for _, server := range servers {
			client := http.NewClient(server)

			start := time.Now()
			response, _ := client.Get("/ping")
			timePassed := time.Now().Sub(start)

			statusCode := 0
			if response != nil {
				statusCode = response.StatusCode
			}

			ok := statusCode == 200
			if ok {
				pingsOK++
				sumPingTime = sumPingTime + int64(timePassed)
			}

			pu, _ := url.Parse(server)
			ag.AssertTrue(fmt.Sprintf("PingOK_%s:%s", pu.Hostname(), pu.Port()), ok, 200, statusCode)
		}

		ag.AssertTrue("Reachable", pingsOK >= 2, ">=2", pingsOK)

		if pingsOK > 0 && sumPingTime > 0 {
			maxAvgDuration := 100 * time.Millisecond
			avgPingDuration := time.Duration(sumPingTime / pingsOK)
			ag.AssertTrue("PingAverageResponseTime", avgPingDuration <= maxAvgDuration, fmt.Sprintf("<=%v", maxAvgDuration), fmt.Sprintf("%v", avgPingDuration))
		}
	}

	return []*AssertionGroup{ag}
}

func getEndpoints(config KubernetesTraefikConfig, endpoints *corev1.Endpoints) []string {
	servers := make([]string, 0)

	var port int32
	for _, subset := range endpoints.Subsets {
		for _, p := range subset.Ports {
			if config.ServicePortName == p.Name {
				port = p.Port
				break
			}
		}

		if port == 0 {
			addrs := selectFrom(subset.Addresses, func(item interface{}) interface{} {
				ea := item.(corev1.EndpointAddress)
				return struct {
					IP       string
					Hostname string
				}{
					IP:       ea.IP,
					Hostname: ea.Hostname,
				}
			})
			log.Errorf("failed to locate service port \"%s\" in the endpoint subset (%v)", config.ServicePortName, addrs)
			break
		}

		protocol := "http"
		if port == 443 || strings.HasPrefix(config.ServicePortName, "https") {
			protocol = "https"
		}

		for _, addr := range subset.Addresses {
			servers = append(servers, fmt.Sprintf("%s://%s:%d", protocol, addr.IP, port))
		}
	}

	return servers
}

func selectFrom(items interface{}, selector func(item interface{}) interface{}) []interface{} {
	listVal := reflect.ValueOf(items)
	results := make([]interface{}, 0)
	for i := 0; i < listVal.Len(); i++ {
		iface := listVal.Index(i).Interface()
		results = append(results, selector(iface))
	}
	return results
}
