package server

import (
	"reflect"

	"github.com/StenaIT/kubecheck/checks"

	"github.com/apex/log"
)

// TODO: Run checks async
func runHealtchecks(healthchecks []checks.Healthcheck, resultMapper func(d checks.Description, r checks.Result) interface{}) map[string]interface{} {
	results := make(map[string]interface{})

	for _, check := range healthchecks {
		typeName, _ := NameOf(check)
		d := check.Describe()
		result := check.Execute()

		l := log.WithFields(log.Fields{
			"type":        typeName,
			"name":        d.Name,
			"description": d.Description,
			"status":      result.Status,
			"reason":      result.Reason,
			"input":       result.Input,
			"output":      result.Output,
		})

		if result.Status == checks.Failed {
			l.Warn("finished executing healthcheck")
		} else {
			l.Debug("finished executing healthcheck")
		}

		results[d.Name] = resultMapper(d, result)
	}

	return results
}

// NameOf returns the name and type for types and pointers
func NameOf(i interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Name(), t
}
