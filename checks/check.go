package checks

import (
	"reflect"
)

// Failed defines a failed check
const Failed string = "failed"

// Passed defines a passed check
const Passed string = "passed"

// Result defines a healthcheck result
type Result struct {
	Status string
	Reason string
	Input  interface{}
	Output interface{}
}

// Description defines a healthcheck description
type Description struct {
	Name        string
	Description string
}

// Healthcheck defines a healthcheck that can be executed
type Healthcheck interface {
	Describe() Description
	Execute() Result
}

// Fail creates a failed healthcheck result
func Fail(reason string) Result {
	return FailWithIO(reason, nil, nil)
}

// FailWithInput creates a failed healthcheck result
func FailWithInput(reason string, input interface{}) Result {
	return FailWithIO(reason, input, nil)
}

// FailWithOutput creates a failed healthcheck result
func FailWithOutput(reason string, output interface{}) Result {
	return FailWithIO(reason, nil, output)
}

// FailWithIO creates a failed healthcheck result
func FailWithIO(reason string, input interface{}, output interface{}) Result {
	return Result{
		Status: Failed,
		Reason: reason,
		Input:  input,
		Output: output,
	}
}

// Pass creates a successful healthcheck result
func Pass() Result {
	return PassWithIO(nil, nil)
}

// PassWithInput creates a successful healthcheck result
func PassWithInput(input interface{}) Result {
	return PassWithIO(input, nil)
}

// PassWithOutput creates a successful healthcheck result
func PassWithOutput(output interface{}) Result {
	return PassWithIO(nil, output)
}

// PassWithIO creates a successful healthcheck result
func PassWithIO(input interface{}, output interface{}) Result {
	return Result{
		Status: Passed,
		Input:  input,
		Output: output,
	}
}

// NameOf returns the name and type for types and pointers
func NameOf(i interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name(), t
}

// NilOrError returns nil or an error string
func NilOrError(err error) interface{} {
	if err != nil {
		return err.Error()
	}
	return nil
}
