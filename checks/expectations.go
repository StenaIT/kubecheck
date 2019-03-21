package checks

// HealthcheckExpectations defines expectations for a healthcheck
type HealthcheckExpectations struct {
	Expectations []interface{}
}

// Assertion represents a single assertion and it's result
type Assertion struct {
	Type     string      `json:"type"`
	Result   string      `json:"result"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
}

// AssertionGroup represents an group of assertions
type AssertionGroup struct {
	Name       string       `json:"name"`
	Entity     interface{}  `json:"entity,omitempty"`
	Result     string       `json:"result"`
	Assertions []*Assertion `json:"assertions"`
}

// VerifyExpectation executes healthcheck assertions and returns the result
func (he HealthcheckExpectations) VerifyExpectation(input interface{}, expectationVerifyer func(expectation interface{}) []*AssertionGroup) Result {
	output := make([]*AssertionGroup, 0)
	ok := true

	if he.Expectations != nil {
		for _, expectation := range he.Expectations {
			ags := expectationVerifyer(expectation)
			if ags != nil && len(ags) > 0 {
				for _, ag := range ags {
					for _, a := range ag.Assertions {
						if a.Result == Failed {
							ok = false
						}
					}
					output = append(output, ag)
				}
			}
		}
	}

	if !ok {
		return FailWithIO("one or more expectations we're not met", input, output)
	}

	return PassWithIO(input, output)
}

// NewAssertionGroup creates a new assertion group
func NewAssertionGroup(name string, entity interface{}) *AssertionGroup {
	ag := &AssertionGroup{Name: name, Entity: entity, Result: Passed}
	ag.Assertions = make([]*Assertion, 0)
	return ag
}

// AssertTrue creates and adds an assertion to the group
func (ag *AssertionGroup) AssertTrue(name string, condition bool, expected interface{}, actual interface{}) {
	assertion := &Assertion{
		Type:     name,
		Result:   Passed,
		Expected: expected,
		Actual:   actual,
	}

	if condition == false {
		assertion.Result = Failed
	}

	ag.Assertions = append(ag.Assertions, assertion)

	result := Passed
	for _, assertion := range ag.Assertions {
		if assertion.Result == Failed {
			result = Failed
		}
	}

	ag.Result = result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
