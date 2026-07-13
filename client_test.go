package httpclient

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BaseURL:    "https://api.example.com",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
	}

	client := NewClient(cfg)

	if client.baseURL != cfg.BaseURL {
		t.Fatalf("expected baseURL %q, got %q", cfg.BaseURL, client.baseURL)
	}

	if client.maxRetries != cfg.MaxRetries {
		t.Fatalf("expected maxRetries %d, got %d", cfg.MaxRetries, client.maxRetries)
	}

	if client.retryDelay != cfg.RetryDelay {
		t.Fatalf("expected retryDelay %v, got %v", cfg.RetryDelay, client.retryDelay)
	}

	if client.httpClient == nil {
		t.Fatal("expected httpClient to be initialized")
	}

	if client.httpClient.Timeout != cfg.Timeout {
		t.Fatalf("expected timeout %v, got %v", cfg.Timeout, client.httpClient.Timeout)
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		err  error
		want bool
	}{
		{
			name: "returns true when request has error",
			err:  errors.New("network error"),
			want: true,
		},
		{
			name: "returns true for 429",
			resp: &http.Response{StatusCode: http.StatusTooManyRequests},
			want: true,
		},
		{
			name: "returns true for 500",
			resp: &http.Response{StatusCode: http.StatusInternalServerError},
			want: true,
		},
		{
			name: "returns true for 502",
			resp: &http.Response{StatusCode: http.StatusBadGateway},
			want: true,
		},
		{
			name: "returns true for 503",
			resp: &http.Response{StatusCode: http.StatusServiceUnavailable},
			want: true,
		},
		{
			name: "returns true for 504",
			resp: &http.Response{StatusCode: http.StatusGatewayTimeout},
			want: true,
		},
		{
			name: "returns false for 200",
			resp: &http.Response{StatusCode: http.StatusOK},
			want: false,
		},
		{
			name: "returns false for 404",
			resp: &http.Response{StatusCode: http.StatusNotFound},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRetry(tt.resp, tt.err)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
