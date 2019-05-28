package hook

import (
	"bytes"

	"github.com/StenaIT/kubecheck/http"
	"github.com/apex/log"
)

// Event defines an event
type Event string

// Webhook defines a webhook
type Webhook struct {
	Name   string
	URL    string
	Data   string
	Events []Event
}

// TriggerWebhooks invokes webhooks matching the event
func TriggerWebhooks(hooks []Webhook, e Event) {
	if hooks != nil {
		for _, hook := range hooks {
			if contains(hook.Events, e) {
				log.WithFields(log.Fields{
					"hook":  hook.Name,
					"event": e,
				}).Info("Invoking webhook")
				body := bytes.NewReader([]byte(hook.Data))
				http.NewClient(hook.URL).Post("", body)
			}
		}
	}
}

func contains(s []Event, e Event) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
