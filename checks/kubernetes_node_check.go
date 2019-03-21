package checks

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesNodeHealthcheck defines a healthcheck that can be executed
type KubernetesNodeHealthcheck struct {
	Name        string
	Description string
	HealthcheckExpectations
}

// KubernetesNodeExpectation defines the expectation interface
type KubernetesNodeExpectation interface {
	Verify(nodes []corev1.Node) []*AssertionGroup
}

// KubernetesNodeCountExpectation defines expectations on Kubernetes node count
type KubernetesNodeCountExpectation struct {
	Min int
	Max int
}

// KubernetesNodeStatusExpectation defines expectations on Kubernetes node status
type KubernetesNodeStatusExpectation struct {
	CreatedGracePeriod time.Duration
}

// Execute runs the healthcheck
func (c KubernetesNodeHealthcheck) Execute() Result {
	clientset := getKubernetesClientset()

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return Fail(err.Error())
	}

	return c.VerifyExpectation(nil, func(expectation interface{}) []*AssertionGroup {
		return expectation.(KubernetesNodeExpectation).Verify(nodes.Items)
	})
}

// Describe returns the description of the healthcheck
func (c KubernetesNodeHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// WithExpectations adds expectations to healthcheck
func (c KubernetesNodeHealthcheck) WithExpectations(expectations ...KubernetesNodeExpectation) KubernetesNodeHealthcheck {
	for _, e := range expectations {
		c.Expectations = append(c.Expectations, e)
	}
	return c
}

// ExpectNodeCount creates an expecation
func ExpectNodeCount(expected int) KubernetesNodeCountExpectation {
	return KubernetesNodeCountExpectation{
		Min: expected,
		Max: expected,
	}
}

// ExpectNodeCountRange creates an expecation
func ExpectNodeCountRange(min int, max int) KubernetesNodeCountExpectation {
	return KubernetesNodeCountExpectation{
		Min: min,
		Max: max,
	}
}

// ExpectNodeCountMin creates an expecation
func ExpectNodeCountMin(min int) KubernetesNodeCountExpectation {
	return KubernetesNodeCountExpectation{
		Min: min,
		Max: -1,
	}
}

// ExpectNodeCountMax creates an expecation
func ExpectNodeCountMax(max int) KubernetesNodeCountExpectation {
	return KubernetesNodeCountExpectation{
		Min: -1,
		Max: max,
	}
}

// ExpectNodeStatusOK creates an expecation
func ExpectNodeStatusOK(graceperiod time.Duration) KubernetesNodeStatusExpectation {
	return KubernetesNodeStatusExpectation{
		CreatedGracePeriod: graceperiod,
	}
}

// Verify is called to verify expectations
func (a KubernetesNodeCountExpectation) Verify(nodes []corev1.Node) []*AssertionGroup {
	ag := NewAssertionGroup("NodeCount", nil)

	nodeCount := len(nodes)
	if a.Min == a.Max {
		ag.AssertTrue("Equals", nodeCount != a.Min, a.Min, nodeCount)
	} else if a.Max == -1 && a.Min > -1 {
		ag.AssertTrue("Min", nodeCount >= a.Min, fmt.Sprintf(">=%d", a.Min), nodeCount)
	} else if a.Min == -1 && nodeCount > a.Max {
		ag.AssertTrue("Max", nodeCount <= a.Max, fmt.Sprintf("<=%d", a.Max), nodeCount)
	} else {
		ag.AssertTrue("InRange", nodeCount >= a.Min && nodeCount <= a.Max, fmt.Sprintf("min=%d max=%d", a.Min, a.Max), nodeCount)
	}

	return []*AssertionGroup{ag}
}

// Verify is called to verify expectations
func (a KubernetesNodeStatusExpectation) Verify(nodes []corev1.Node) []*AssertionGroup {
	out := make([]*AssertionGroup, 0)

	for _, node := range nodes {
		ag := NewAssertionGroup("NodeStatus", node.Name)

		gracePeriodEnd := node.GetCreationTimestamp().Add(a.CreatedGracePeriod)
		if time.Since(gracePeriodEnd) <= 0 {
			continue
		}

		for _, nc := range node.Status.Conditions {
			if nc.Type == corev1.NodeReady {
				ag.AssertTrue(string(nc.Type), nc.Status == corev1.ConditionTrue, corev1.ConditionTrue, nc.Status)
			} else {
				ag.AssertTrue(string(nc.Type), nc.Status == corev1.ConditionFalse, corev1.ConditionFalse, nc.Status)
			}
		}

		out = append(out, ag)
	}

	return out
}
