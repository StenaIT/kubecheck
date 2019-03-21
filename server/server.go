package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/StenaIT/kubecheck/checks"
	"github.com/StenaIT/kubecheck/config"

	"github.com/apex/log"
	"github.com/gorilla/mux"
)

type indexResponse struct {
	Checks []checkDescriptionResponse `json:"checks"`
}

type checkDescriptionResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type apiCheckResponse struct {
	Description string      `json:"description"`
	Status      string      `json:"status"`
	Reason      string      `json:"reason,omitempty"`
	Input       interface{} `json:"input,omitempty"`
	Output      interface{} `json:"output,omitempty"`
}

// New creates a new HTTP server for kubecheck
func New(kubecheck *config.Kubecheck) *http.Server {
	if kubecheck.Router == nil {
		kubecheck.Router = mux.NewRouter()
		kubecheck.Router.HandleFunc("/", indexHandler(kubecheck))
		kubecheck.Router.HandleFunc("/checks/", healtchecksHandler(kubecheck.Config, kubecheck.Healthchecks))

		for _, c := range kubecheck.Healthchecks {
			hcks := []checks.Healthcheck{c}
			kubecheck.Router.HandleFunc(getHealthcheckPath(c), healtchecksHandler(kubecheck.Config, hcks))
		}

		kubecheck.Router.Use(loggingMiddleware)
	}

	srv := &http.Server{
		Handler:      kubecheck.Router,
		Addr:         ":8113",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.WithFields(log.Fields{
		"service": "HTTP-Server",
		"address": srv.Addr,
	}).Infof("listening on %s", srv.Addr)

	return srv
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"service":   "HTTP-Server",
			"method":    r.Method,
			"path":      r.RequestURI,
			"direction": "incoming",
		}).Debugf("HTTP %s %s", r.Method, r.RequestURI)

		scrw := &statusCodeResponseWriter{w, http.StatusOK}
		next.ServeHTTP(scrw, r)

		log.WithFields(log.Fields{
			"service":   "HTTP-Server",
			"status":    scrw.statusCode,
			"direction": "outgoing",
		}).Debugf("HTTP %s %s - %d %s", r.Method, r.RequestURI, scrw.statusCode, http.StatusText(scrw.statusCode))
	})
}

type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *statusCodeResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func indexHandler(kubecheck *config.Kubecheck) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		response := indexResponse{
			Checks: make([]checkDescriptionResponse, 0),
		}

		for _, c := range kubecheck.Healthchecks {
			d := c.Describe()
			response.Checks = append(response.Checks, checkDescriptionResponse{
				Name:        d.Name,
				Description: d.Description,
				URL:         generateURL(r, c),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		js, err := json.Marshal(response)
		if err == nil {
			w.Write(js)
		}
	}
}

func healtchecksHandler(config *config.KubecheckConfig, healthchecks []checks.Healthcheck) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		results := runHealtchecks(healthchecks, func(d checks.Description, r checks.Result) interface{} {
			if r.Status == checks.Failed {
				statusCode = http.StatusFailedDependency
			}
			var input interface{}
			var output interface{}
			if r.Status == checks.Failed || config.Debug {
				input = r.Input
				output = r.Output
			}
			return apiCheckResponse{
				Description: d.Description,
				Status:      r.Status,
				Reason:      r.Reason,
				Input:       input,
				Output:      output,
			}
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		js, err := json.Marshal(results)
		if err == nil {
			w.Write(js)
		}
	}
}

func generateURL(r *http.Request, healthcheck checks.Healthcheck) string {
	path := getHealthcheckPath(healthcheck)

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
}

func getHealthcheckPath(healthcheck checks.Healthcheck) string {
	name := healthcheck.Describe().Name
	return fmt.Sprintf("/checks/%s", url.PathEscape(name))
}
