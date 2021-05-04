package checks

import (
	"fmt"
	"strings"

	"github.com/apex/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesPodAntiAffinityHealthcheck defines a healthcheck that can be executed
type KubernetesPodAntiAffinityHealthcheck struct {
	Name               string
	Description        string
	ExcludeNamespaces  []string
	ExcludeDeployments []string
	HealthcheckExpectations
}

// KubernetesPodAntiAffinityExpectation defines the expectation interface
type KubernetesPodAntiAffinityExpectation interface {
	Verify(deployments []appsv1.Deployment, pods []corev1.Pod) []*AssertionGroup
}

// KubernetesNodeSpreadExpectation defines expectations on Kubernetes node count
type KubernetesNodeSpreadExpectation struct {
	Min int
}

// kubernetesDeploymentNodeSpread contains node spread information for a deployment
type kubernetesDeploymentNodeSpread struct {
	Description string                        `json:"description"`
	Deployment  string                        `json:"deployment"`
	Namespace   string                        `json:"namespace"`
	Pods        []kubernetesDeploymentPodInfo `json:"pods"`
	NodeSpread  int                           `json:"nodeSpread"`
}

// kubernetesDeploymentPodInfo contains pod information for a deployment
type kubernetesDeploymentPodInfo struct {
	Name     string `json:"name"`
	NodeName string `json:"nodeName"`
}

// Execute runs the healthcheck
func (c KubernetesPodAntiAffinityHealthcheck) Execute() Result {
	clientset := getKubernetesClientset()

	deps, err := clientset.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	deployments := make([]appsv1.Deployment, 0)
	for _, d := range deps.Items {
		if contains(c.ExcludeNamespaces, d.GetNamespace()) || contains(c.ExcludeDeployments, d.GetName()) {
			continue
		}
		deployments = append(deployments, d)
	}

	return c.VerifyExpectation(nil, func(expectation interface{}) []*AssertionGroup {
		return expectation.(KubernetesPodAntiAffinityExpectation).Verify(deployments, pods.Items)
	})
}

// Describe returns the description of the healthcheck
func (c KubernetesPodAntiAffinityHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// WithExpectations adds expectations to healthcheck
func (c KubernetesPodAntiAffinityHealthcheck) WithExpectations(expectations ...KubernetesNodeSpreadExpectation) KubernetesPodAntiAffinityHealthcheck {
	for _, e := range expectations {
		c.Expectations = append(c.Expectations, e)
	}
	return c
}

// ExpectNodeCountRange creates an expecation
func ExpectNodeSpread(min int) KubernetesNodeSpreadExpectation {
	return KubernetesNodeSpreadExpectation{
		Min: min,
	}
}

// Verify is called to verify expectations
func (e KubernetesNodeSpreadExpectation) Verify(deployments []appsv1.Deployment, pods []corev1.Pod) []*AssertionGroup {
	ags := make([]*AssertionGroup, 0)
	for _, d := range deployments {
		if *d.Spec.Replicas < 2 {
			continue
		}

		nodes := make(map[string]int)
		podResults := make([]kubernetesDeploymentPodInfo, 0)

		l := log.WithFields(log.Fields{
			"namespace":  d.GetNamespace(),
			"deployment": d.GetName(),
			"replicas":   *d.Spec.Replicas,
		})

		l.Debugf("checking pod anti affinity")

		for _, p := range pods {
			prefix := fmt.Sprintf("%s-", d.GetName())
			if strings.HasPrefix(p.GetName(), prefix) && p.GetNamespace() == d.GetNamespace() {
				podResults = append(podResults, kubernetesDeploymentPodInfo{
					Name:     p.GetName(),
					NodeName: p.Spec.NodeName,
				})
				nodes[p.Spec.NodeName] = nodes[p.Spec.NodeName] + 1
			}
		}

		spread := kubernetesDeploymentNodeSpread{
			Description: fmt.Sprintf("%d pod(s) are spread across %d node(s).", len(podResults), len(nodes)),
			Deployment:  d.GetName(),
			Namespace:   d.GetNamespace(),
			NodeSpread:  len(nodes),
			Pods:        podResults,
		}

		ag := NewAssertionGroup("NodeSpread", spread)

		condition := len(nodes) >= e.Min
		ag.AssertTrue("Min", condition, e.Min, spread.NodeSpread)

		if condition == true {
			l.Debugf("success: %s", spread.Description)
		} else {
			l.Debugf("failed: %s", spread.Description)
		}

		ags = append(ags, ag)
	}

	return ags
}
