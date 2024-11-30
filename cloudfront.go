package traefik_cloudfront

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	// RemoteAddressHeader is the request header added by Cloudfront containing the client's IP address.
	RemoteAddressHeader = "Cloudfront-Viewer-Address"
)

// Config the plugin configuration.
type Config struct {
	Headers    []string `json:"headers,omitempty"`
	TrustedIPs []string `json:"trustedIPs,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Headers:    []string{"X-Forwarded-For", "X-Real-IP"},
		TrustedIPs: []string{},
	}
}

// Cloudfront a Cloudfront plugin.
type Cloudfront struct {
	next       http.Handler
	headers    []string
	trustedIps []*net.IPNet
	name       string
}

// New created a new Cloudfront plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	trustedIps := []*net.IPNet{}

	for _, ip := range config.TrustedIPs {
		_, ipnet, err := net.ParseCIDR(ip)

		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %s: %w", ip, err)
		}

		trustedIps = append(trustedIps, ipnet)
	}

	return &Cloudfront{
		headers:    config.Headers,
		trustedIps: trustedIps,
		next:       next,
		name:       name,
	}, nil
}

// ServeHTTP processes the request.
func (cf *Cloudfront) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if len(cf.trustedIps) == 0 {
		cf.next.ServeHTTP(rw, req)
		return
	}

	if cf.trustedIps[0].String() != "0.0.0.0/0" {
		remoteIP := net.ParseIP(req.RemoteAddr)

		if remoteIP == nil {
			cf.next.ServeHTTP(rw, req)
			return
		}

		trusted := false

		for _, addr := range cf.trustedIps {
			if addr.Contains(remoteIP) {
				trusted = true
				break
			}
		}

		if !trusted {
			cf.next.ServeHTTP(rw, req)
			return
		}
	}

	// Cloudfront sets the view address to IP:PORT, but it is using a non-standard representation for IPv6,
	// which cannot be parsed by SplitHostPort, so we just cut it from the string
	split := strings.Split(req.Header.Get(RemoteAddressHeader), ":")
	var addr string

	switch l := len(split); {
	case l < 2:
		cf.next.ServeHTTP(rw, req)
		return
	case l == 2:
		addr = split[0]
	default:
		addr = strings.Join(split[0:len(split)-1], ":")
	}

	for _, header := range cf.headers {
		req.Header.Set(header, addr)
	}

	cf.next.ServeHTTP(rw, req)
}
