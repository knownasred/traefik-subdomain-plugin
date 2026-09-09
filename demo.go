// Package plugindemo extracts a tenant from the request host for upstream services.
package plugindemo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
)

// Config controls host matching and the upstream tenant header.
type Config struct {
	HostRegex    string `json:"hostRegex,omitempty"`
	HeaderName   string `json:"headerName,omitempty"`
	CaptureGroup int    `json:"captureGroup,omitempty"`
}

// CreateConfig returns defaults equivalent to the KEPA NGINX snippet.
func CreateConfig() *Config {
	return &Config{HostRegex: `^([^.]+)\.app\.kepa\.ch$`, HeaderName: "X-Tenant", CaptureGroup: 1}
}

// Tenant sets a request header from a configured host capture group.
type Tenant struct {
	next         http.Handler
	hostRegex    *regexp.Regexp
	headerName   string
	captureGroup int
}

// New validates configuration once, when Traefik constructs the middleware.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("%s: configuration is required", name)
	}
	if config.HostRegex == "" {
		return nil, fmt.Errorf("%s: hostRegex is required", name)
	}
	re, err := regexp.Compile(config.HostRegex)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid hostRegex: %w", name, err)
	}
	if config.CaptureGroup < 1 || config.CaptureGroup > re.NumSubexp() {
		return nil, fmt.Errorf("%s: captureGroup must be between 1 and %d", name, re.NumSubexp())
	}
	if !validHeaderName(config.HeaderName) || strings.EqualFold(config.HeaderName, "Host") {
		return nil, fmt.Errorf("%s: headerName must be a valid HTTP header name other than Host", name)
	}
	return &Tenant{next: next, hostRegex: re, headerName: http.CanonicalHeaderKey(config.HeaderName), captureGroup: config.CaptureGroup}, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c) {
			continue
		}
		return false
	}
	return true
}

// ServeHTTP overwrites untrusted tenant headers and omits the header on no match.
func (t *Tenant) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := req.Host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	// Delete even noncanonical keys introduced by an earlier middleware.
	for key := range req.Header {
		if strings.EqualFold(key, t.headerName) {
			delete(req.Header, key)
		}
	}
	if match := t.hostRegex.FindStringSubmatch(host); match != nil && match[t.captureGroup] != "" {
		if req.Header == nil {
			req.Header = make(http.Header)
		}
		req.Header.Set(t.headerName, match[t.captureGroup])
	}
	t.next.ServeHTTP(rw, req)
}
