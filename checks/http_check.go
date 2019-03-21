package checks

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/StenaIT/kubecheck/http"
)

// HTTPGetHealthcheck defines a healthcheck that can be executed
type HTTPGetHealthcheck struct {
	Name        string
	Description string
	URL         string
	HealthcheckExpectations
}

// HTTPResponseExpectation defines the expectation interface
type HTTPResponseExpectation interface {
	Verify(response *nethttp.Response) []*AssertionGroup
}

// HTTPStatusCodeExpectation defines expectations on HTTP Status Codes
type HTTPStatusCodeExpectation struct {
	MinStatusCode int
	MaxStatusCode int
}

// HTTPCertificateExpectation defines expectations on HTTP Status Codes
type HTTPCertificateExpectation struct {
	ExpiresAfterDays int
}

// HTTPResponseBodyExpectation defines expectations on HTTP Response Body content
type HTTPResponseBodyExpectation struct {
	Expected   string
	ExactMatch bool
}

// HTTPResponseHeaderExpectation defines expectations on HTTP Response Header
type HTTPResponseHeaderExpectation struct {
	Header   string
	Expected string
}

// Execute runs the healthcheck
func (c HTTPGetHealthcheck) Execute() Result {
	input := struct {
		URL string `json:"url"`
	}{
		http.CleanURL(c.URL),
	}

	client := http.NewClient(c.URL)
	resp, err := client.Get("")
	if err != nil {
		return FailWithInput(err.Error(), input)
	}

	return c.VerifyExpectation(input, func(assertion interface{}) []*AssertionGroup {
		return assertion.(HTTPResponseExpectation).Verify(resp)
	})
}

// Describe returns the description of the healthcheck
func (c HTTPGetHealthcheck) Describe() Description {
	return Description{
		Name:        c.Name,
		Description: c.Description,
	}
}

// WithExpectations adds expectations to the healthcheck
func (c HTTPGetHealthcheck) WithExpectations(expectations ...HTTPResponseExpectation) HTTPGetHealthcheck {
	for _, e := range expectations {
		c.Expectations = append(c.Expectations, e)
	}
	return c
}

// ExpectStatusCode creates an expecation
func ExpectStatusCode(expected int) HTTPStatusCodeExpectation {
	return HTTPStatusCodeExpectation{
		MinStatusCode: expected,
		MaxStatusCode: expected,
	}
}

// ExpectValidCertificate creates an expecation
func ExpectValidCertificate(expiresAfterDays int) HTTPCertificateExpectation {
	return HTTPCertificateExpectation{
		ExpiresAfterDays: expiresAfterDays,
	}
}

// ExpectStatusCodeRange creates an expecation
func ExpectStatusCodeRange(min int, max int) HTTPStatusCodeExpectation {
	return HTTPStatusCodeExpectation{
		MinStatusCode: min,
		MaxStatusCode: max,
	}
}

// ExpectStatusCodeSuccess creates an expecation
func ExpectStatusCodeSuccess() HTTPStatusCodeExpectation {
	return HTTPStatusCodeExpectation{
		MinStatusCode: 100,
		MaxStatusCode: 299,
	}
}

// ExpectBodyEquals creates an expecation
func ExpectBodyEquals(expected string) HTTPResponseBodyExpectation {
	return HTTPResponseBodyExpectation{
		Expected:   expected,
		ExactMatch: true,
	}
}

// ExpectBodyContains creates an expecation
func ExpectBodyContains(expected string) HTTPResponseBodyExpectation {
	return HTTPResponseBodyExpectation{
		Expected:   expected,
		ExactMatch: false,
	}
}

// ExpectHeader creates an expecation
func ExpectHeader(header string, expected string) HTTPResponseHeaderExpectation {
	return HTTPResponseHeaderExpectation{
		Header:   header,
		Expected: expected,
	}
}

// Verify is called to verify expectations
func (e HTTPStatusCodeExpectation) Verify(response *nethttp.Response) []*AssertionGroup {
	ag := NewAssertionGroup("HTTPStatusCode", nil)

	if e.MinStatusCode == e.MaxStatusCode {
		ag.AssertTrue("Equals", response.StatusCode == e.MinStatusCode, e.MinStatusCode, response.StatusCode)
	} else {
		ag.AssertTrue("InRange", response.StatusCode >= e.MinStatusCode && response.StatusCode <= e.MaxStatusCode, fmt.Sprintf("min=%d max=%d", e.MinStatusCode, e.MaxStatusCode), response.StatusCode)
	}

	return []*AssertionGroup{ag}
}

// Verify is called to verify expectations
func (e HTTPCertificateExpectation) Verify(response *nethttp.Response) []*AssertionGroup {
	out := make([]*AssertionGroup, 0)

	empty := response.TLS == nil || len(response.TLS.PeerCertificates) <= 0
	if !empty {
		for _, cert := range response.TLS.PeerCertificates {
			ag := NewAssertionGroup("Certificate", struct {
				Subject string
				Issuer  string
			}{Subject: cert.Subject.String(), Issuer: cert.Issuer.String()})

			if e.ExpiresAfterDays > 0 {
				expiresInDays := certExpiresInDays(cert)
				ag.AssertTrue("Expires", expiresInDays >= e.ExpiresAfterDays, fmt.Sprintf("after %d days", e.ExpiresAfterDays), fmt.Sprintf("in %d days", expiresInDays))
			}

			out = append(out, ag)
		}
	} else {
		ag := NewAssertionGroup("Certificate", nil)
		ag.AssertTrue("HasValue", !empty, true, !empty)
		out = append(out, ag)
	}

	return out
}

func certExpiresInDays(cert *x509.Certificate) int {
	return int(cert.NotAfter.Sub(time.Now()).Hours() / 24)
}

// Verify is called to verify expectations
func (e HTTPResponseBodyExpectation) Verify(response *nethttp.Response) []*AssertionGroup {
	ag := NewAssertionGroup("HTTPResponseBody", nil)

	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	bodyString := string(body[:])
	bodyOutput := bodyString
	if (len(bodyString) - 1) > 50 {
		bodyOutput = fmt.Sprintf("%s...[TRIMMED]", bodyString[:50])
	}

	if e.ExactMatch {

		ag.AssertTrue("Equals", bodyString == e.Expected, e.Expected, bodyOutput)
	} else {
		ag.AssertTrue("Contains", strings.Contains(bodyString, e.Expected), e.Expected, bodyOutput)
	}

	return []*AssertionGroup{ag}
}

// Verify is called to verify expectations
func (e HTTPResponseHeaderExpectation) Verify(response *nethttp.Response) []*AssertionGroup {
	ag := NewAssertionGroup("HTTPResponseHeader", e.Header)

	actual := response.Header.Get(e.Header)
	ag.AssertTrue("Equals", actual == e.Expected, e.Expected, actual)

	return []*AssertionGroup{ag}
}
