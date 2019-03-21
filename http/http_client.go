package http

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"
)

// Client defines a new HTTP client
type Client struct {
	BaseURL     string
	Port        int
	Scheme      string
	ContentType string
	Client      *http.Client
}

// NewClient creates a new HTTP client
func NewClient(baseURL string) *Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		log.WithFields(log.Fields{
			"service": "HTTP-Client",
		}).WithError(err).Error("failed to create HTTP client")
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		port = 80
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &Client{
		BaseURL:     baseURL,
		Port:        port,
		Scheme:      u.Scheme,
		ContentType: "application/json",
		Client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: tr,
		},
	}
}

func (c *Client) request(method string, path string, requestBody io.Reader) (*http.Response, error) {
	url := c.BaseURL + path
	if strings.HasPrefix(path, "http") {
		url = path
	}
	req, _ := http.NewRequest(method, url, requestBody)
	req.Header.Set("Content-Type", c.ContentType)

	log.WithFields(log.Fields{
		"service": "HTTP-Client",
	}).Debugf("HTTP %s %s", method, CleanURL(url))

	resp, err := c.Client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"service": "HTTP-Client",
		}).Debugf("error: %v", err)
		return nil, err
	}

	return resp, nil
}

// Get performs a HTTP GET request
func (c *Client) Get(path string) (*http.Response, error) {
	return c.request("GET", path, nil)
}

// Post performs a HTTP POST request
func (c *Client) Post(path string, body io.Reader) (*http.Response, error) {
	return c.request("POST", path, body)
}

// Put performs a HTTP POST request
func (c *Client) Put(path string, body io.Reader) (*http.Response, error) {
	return c.request("PUT", path, body)
}

// Delete performs a HTTP DELETE request
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.request("DELETE", path, nil)
}
