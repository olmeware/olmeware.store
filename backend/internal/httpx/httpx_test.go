package httpx

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{"ipv4 with port", "203.0.113.5:54321", "", "203.0.113.5"},
		{"ipv6 with port", "[::1]:51724", "", "::1"},
		{"xff first hop", "10.0.0.1:1", "198.51.100.9, 10.0.0.1", "198.51.100.9"},
		{"bad xff falls back", "203.0.113.5:80", "not-an-ip", "203.0.113.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{RemoteAddr: c.remoteAddr, Header: http.Header{}}
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := ClientIP(r); got != c.want {
				t.Errorf("ClientIP() = %q, want %q", got, c.want)
			}
		})
	}
}
