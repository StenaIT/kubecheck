package checks

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesPodHealthcheck defines a healthcheck that can be executed
type KubernetesPodHealthcheck struct {
	Name        string
	Description string
	Config      KubernetesPodConfig
	HealthcheckExpectations
}

// KubernetesPodConfig defines the configuration for the healthcheck
type KubernetesPodConfig struct {
	Namespace          string        `json:"namespace"`
	CreatedGracePeriod time.Duration `json:"gracePeriod"`
	ExcludePods        []string      `json:"excludePods"`
}

// KubernetesPodExpectationContext defines the context for expecations
type KubernetesPodExpectationContext struct {
	Config KubernetesPodConfig
	Pods   []corev1.Pod
}

// KubernetesPodExpectation defines the expectation interface
type KubernetesPodExpectation interface {
	Verify(context KubernetesPodExpectationContext) []*AssertionGroup
}

// KubernetesPodStatusExpectation defines expectations on Kubernetes node status
type KubernetesPodStatusExpectation struct {
}

// KubernetesPodContainerExpectation defines expectations on Kubernetes node status
type KubernetesPodContainerExpectation struct {
	MaxRestarts int32
}

// NewKubernetesPodHealthcheck creates the healthcheck
func NewKubernetesPodHealthcheck(name string, config KubernetesPodConfig) KubernetesPodHealthcheck {
	return KubernetesPodHealthcheck{
		Name:        name,
		Description: "Performs pod healthchecks",
		Config:      config,
		HealthcheckExpectations: HealthcheckExpectations{
			Expectations: make([]interface{}, 0),
		},
	}
}

// Execute runs the healthcheck
func (c KubernetesPodHealthcheck) Execute() Result {
	clientset := getKubernetesClientset()

	pods, err := clientset.CoreV1().Pods(c.Config.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	context := KubernetesPodExpectationContext{
		Config: c.Config,
		Pods:   pods.Items,
	}

	return c.VerifyExpectation(nil, func(expectation interface{}) []*AssertionGroup {
		return expectation.(KubernetesPodExpectation).Verify(context)
	})
}

// Describe returns the description of the healthcheck
func (c KubernetesPodHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// WithExpectations adds expectations to healthcheck
func (c KubernetesPodHealthcheck) WithExpectations(expectations ...KubernetesPodExpectation) KubernetesPodHealthcheck {
	for _, e := range expectations {
		c.Expectations = append(c.Expectations, e)
	}
	return c
}

// ExpectPodStatusOK creates an expecation
func ExpectPodStatusOK() KubernetesPodStatusExpectation {
	return KubernetesPodStatusExpectation{}
}

// ExpectPodMaxContainerRestarts creates an expecation
func ExpectPodMaxContainerRestarts(max int32) KubernetesPodContainerExpectation {
	return KubernetesPodContainerExpectation{
		MaxRestarts: max,
	}
}

// Verify is called to verify expectations
func (e KubernetesPodStatusExpectation) Verify(context KubernetesPodExpectationContext) []*AssertionGroup {
	out := make([]*AssertionGroup, 0)

	for _, pod := range context.Pods {
		if excludePod(context, pod) {
			continue
		}

		ag := NewAssertionGroup("PodStatus", struct {
			PodName     string            `json:"podName"`
			Namespace   string            `json:"namespace"`
			Created     metav1.Time       `json:"created"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
		}{
			PodName:     pod.Name,
			Namespace:   pod.Namespace,
			Created:     pod.GetCreationTimestamp(),
			Labels:      pod.GetLabels(),
			Annotations: pod.GetAnnotations(),
		})

		for _, pc := range pod.Status.Conditions {
			ag.AssertTrue(string(pc.Type), pc.Status == corev1.ConditionTrue, corev1.ConditionTrue, pc.Status)
		}

		out = append(out, ag)
	}

	return out
}

// Verify is called to verify expectations
func (e KubernetesPodContainerExpectation) Verify(context KubernetesPodExpectationContext) []*AssertionGroup {
	out := make([]*AssertionGroup, 0)

	for _, pod := range context.Pods {
		if excludePod(context, pod) {
			continue
		}

		for _, pcs := range pod.Status.ContainerStatuses {
			ag := NewAssertionGroup("ContainerStatus", struct {
				ContainerName string            `json:"containerName"`
				PodName       string            `json:"podName"`
				Namespace     string            `json:"namespace"`
				Created       metav1.Time       `json:"created"`
				Labels        map[string]string `json:"labels"`
				Annotations   map[string]string `json:"annotations"`
			}{
				ContainerName: pcs.Name,
				PodName:       pod.Name,
				Namespace:     pod.Namespace,
				Created:       pod.GetCreationTimestamp(),
				Labels:        pod.GetLabels(),
				Annotations:   pod.GetAnnotations(),
			})

			ag.AssertTrue("Ready", pcs.Ready, true, pcs.Ready)

			if e.MaxRestarts >= 0 {
				ag.AssertTrue("RestartCount", pcs.RestartCount <= e.MaxRestarts, "<=1", fmt.Sprint(pcs.RestartCount))
			}

			out = append(out, ag)
		}
	}

	return out
}

func excludePod(context KubernetesPodExpectationContext, pod corev1.Pod) bool {
	gracePeriodEnd := pod.GetCreationTimestamp().Add(context.Config.CreatedGracePeriod)
	if time.Since(gracePeriodEnd) <= 0 {
		return true
	}

	for _, prefix := range context.Config.ExcludePods {
		if strings.HasPrefix(pod.Name, prefix) {
			return true
		}
	}

	return false
}
