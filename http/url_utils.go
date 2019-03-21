package http

import (
	"fmt"
	"net/url"
	"strings"
)

// CleanURL removes basic auth credentials from the URL
func CleanURL(u string) string {
	uri, _ := url.Parse(u)
	p, _ := uri.User.Password()
	pf := fmt.Sprintf(":%s@", p)
	u = strings.Replace(u, pf, ":****@", -1)
	return u
}
