package checks

import (
	"net"
)

// DNSLookupHealthcheck defines a healthcheck that can be executed
type DNSLookupHealthcheck struct {
	Name        string
	Description string
	Host        string
	HealthcheckExpectations
}

// DNSLookupExpectation defines the expectation interface
type DNSLookupExpectation interface {
	Verify(addrs []string) []*AssertionGroup
}

// DNSLookupAddrExpectation defines expectations on DNS Lookup addresses
type DNSLookupAddrExpectation struct {
	Expected []string
}

// Execute runs the healthcheck
func (c DNSLookupHealthcheck) Execute() Result {
	input := struct {
		Host string `json:"host"`
	}{
		c.Host,
	}

	addrs, err := net.LookupHost(c.Host)
	if err != nil {
		return FailWithInput(err.Error(), input)
	}

	return c.VerifyExpectation(input, func(expecation interface{}) []*AssertionGroup {
		return expecation.(DNSLookupExpectation).Verify(addrs)
	})
}

// Describe returns the description of the healthcheck
func (c DNSLookupHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// WithExpectations adds expectations to the healthcheck
func (c DNSLookupHealthcheck) WithExpectations(expectations ...DNSLookupExpectation) DNSLookupHealthcheck {
	for _, e := range expectations {
		c.Expectations = append(c.Expectations, e)
	}
	return c
}

// ExpectAddrs creates an expecation
func ExpectAddrs(addrs ...string) DNSLookupAddrExpectation {
	return DNSLookupAddrExpectation{
		Expected: addrs,
	}
}

// Verify is called to verify expectations
func (a DNSLookupAddrExpectation) Verify(addrs []string) []*AssertionGroup {
	ag := NewAssertionGroup("DNSLookupAddr", nil)

	for _, expectedAddr := range a.Expected {
		ag.AssertTrue("Contains", contains(addrs, expectedAddr), expectedAddr, addrs)
	}

	return []*AssertionGroup{ag}
}
