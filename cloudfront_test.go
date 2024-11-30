package cloudfront_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/invit/traefik-cloudfront"
)

func TestCloudfront(t *testing.T) {
	tests := []struct {
		name          string
		trustedIPs    []string
		viewerAddress string
		clientIP      string
		remoteAddr    string
		headers       []string
		trusted       bool
	}{
		{
			name:          "IPv4 remote / IPv4 trusted",
			trustedIPs:    []string{"10.0.0.0/8"},
			viewerAddress: "1.2.3.4:1585",
			clientIP:      "1.2.3.4",
			remoteAddr:    "10.0.0.1",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       true,
		}, {
			name:          "IPv4 remote / IPv4 all trusted",
			trustedIPs:    []string{"0.0.0.0/0"},
			viewerAddress: "1.2.3.4:1585",
			clientIP:      "1.2.3.4",
			remoteAddr:    "10.0.0.1",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       true,
		}, {
			name:          "IPv4 remote / IPv4 untrusted",
			trustedIPs:    []string{"10.0.0.0/8"},
			viewerAddress: "1.2.3.4:1585",
			clientIP:      "1.2.3.4",
			remoteAddr:    "100.0.0.1",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       false,
		}, {
			name:          "IPv6 remote (full length) / IPv4 trusted",
			trustedIPs:    []string{"10.0.0.0/8"},
			viewerAddress: "1:2:3:4:5:6:7:8:9:4242",
			clientIP:      "1:2:3:4:5:6:7:8:9",
			remoteAddr:    "10.0.0.1",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       true,
		}, {
			name:          "IPv6 remote (shortened) / IPv4 trusted",
			trustedIPs:    []string{"10.0.0.0/8"},
			viewerAddress: "1::9:4242",
			clientIP:      "1::9",
			remoteAddr:    "10.0.0.1",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       true,
		}, {
			name:          "IPv6 remote (full length) / IPv6 trusted",
			trustedIPs:    []string{"10::0/64"},
			viewerAddress: "1:2:3:4:5:6:7:8:9:4242",
			clientIP:      "1:2:3:4:5:6:7:8:9",
			remoteAddr:    "10::ffff:ffff:ffff:fffe",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       true,
		}, {
			name:          "IPv6 remote (full length) / IPv6 untrusted",
			trustedIPs:    []string{"10::0/64"},
			viewerAddress: "1:2:3:4:5:6:7:8:9:4242",
			clientIP:      "1:2:3:4:5:6:7:8:9",
			remoteAddr:    "11::ffff:ffff:ffff:fffe",
			headers:       []string{"X-Forwarded-For", "X-Real-IP"},
			trusted:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cfg := cloudfront.CreateConfig()
			cfg.TrustedIPs = tt.trustedIPs
			cfg.Headers = tt.headers

			next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

			handler, err := cloudfront.New(ctx, next, cfg, "cloudfront")
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)

			if err != nil {
				t.Fatal(err)
			}

			req.RemoteAddr = tt.remoteAddr
			req.Header.Set(cloudfront.RemoteAddressHeader, tt.viewerAddress)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			expected := ""

			if tt.trusted {
				expected = tt.clientIP
			}

			for _, h := range tt.headers {
				assertHeader(t, req, h, expected)
			}
		})
	}
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()

	if req.Header.Get(key) != expected {
		t.Errorf("invalid header value: %s", req.Header.Get(key))
	}
}
