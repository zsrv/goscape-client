package main

import "testing"

func TestParseOndemandServer(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    string
		wantErr bool
	}{
		{"http localhost", "http://127.0.0.1:8888", "http://127.0.0.1:8888", false},
		{"https host", "https://cache.example.com:443", "https://cache.example.com:443", false},
		{"root path stripped", "http://cache.example.com:8888/", "http://cache.example.com:8888", false},
		{"ipv6", "http://[::1]:8888", "http://[::1]:8888", false},
		{"missing scheme", "cache.example.com:8888", "", true},
		{"missing port", "http://cache.example.com", "", true},
		{"unsupported scheme", "ftp://cache:21", "", true},
		{"has path", "http://cache:8888/path", "", true},
		{"empty host", "http://:8888", "", true},
		{"bad port", "http://cache:notaport", "", true},
		{"port out of range", "http://cache:70000", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOndemandServer(tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOndemandServer(%q) = nil error, want error", tt.arg)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOndemandServer(%q) unexpected error: %v", tt.arg, err)
			}
			if got != tt.want {
				t.Fatalf("parseOndemandServer(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}
