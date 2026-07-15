package httpclient

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BaseURL:     "https://api.example.com",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		RetryDelay:  100 * time.Millisecond,
		Logger:      log.New(io.Discard, "", 0),
		Accept:      "text/plain",
		ContentType: "text/plain",
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

	if client.logger != cfg.Logger {
		t.Fatal("expected logger to be configured")
	}

	if client.accept != cfg.Accept {
		t.Fatalf("expected accept %q, got %q", cfg.Accept, client.accept)
	}

	if client.contentType != cfg.ContentType {
		t.Fatalf("expected contentType %q, got %q", cfg.ContentType, client.contentType)
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
			got := shouldRetry(tt.resp, tt.err)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestGetBuildsURLWithSingleSlash(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "path without leading slash",
			path: "health",
		},
		{
			name: "path with leading slash",
			path: "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
				}

				if r.URL.Path != "/health" {
					t.Fatalf("expected path /health, got %s", r.URL.Path)
				}

				_, _ = io.WriteString(w, "ok")
			}))
			defer server.Close()

			client := NewClient(Config{
				BaseURL: server.URL,
				Timeout: 5 * time.Second,
			})

			body, err := client.Get(context.Background(), tt.path)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if string(body) != "ok" {
				t.Fatalf("expected body ok, got %q", string(body))
			}
		})
	}
}

func TestPostReusesBodyOnRetry(t *testing.T) {
	attempts := 0
	wantBody := `{"name":"Joao"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if got := r.Header.Get("Accept"); got != jsonContentType {
			t.Fatalf("expected Accept %q, got %q", jsonContentType, got)
		}

		if got := r.Header.Get("Content-Type"); got != jsonContentType {
			t.Fatalf("expected Content-Type %q, got %q", jsonContentType, got)
		}

		if string(body) != wantBody {
			t.Fatalf("expected body %q, got %q", wantBody, string(body))
		}

		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		RetryDelay: time.Millisecond,
	})

	body, err := client.Post(context.Background(), "users", []byte(wantBody))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(body) != "ok" {
		t.Fatalf("expected body ok, got %q", string(body))
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestRequestUsesConfiguredHeaders(t *testing.T) {
	const (
		accept      = "text/plain"
		contentType = "text/plain"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != accept {
			t.Fatalf("expected Accept %q, got %q", accept, got)
		}

		if got := r.Header.Get("Content-Type"); got != contentType {
			t.Fatalf("expected Content-Type %q, got %q", contentType, got)
		}

		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:     server.URL,
		Accept:      accept,
		ContentType: contentType,
	})

	body, err := client.Post(context.Background(), "messages", []byte("hello"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(body) != "ok" {
		t.Fatalf("expected body ok, got %q", string(body))
	}
}

func TestHTTPVerbHelpers(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       []byte
		call       func(context.Context, *Client, string, []byte) ([]byte, error)
		expectBody bool
	}{
		{
			name:   "head",
			method: http.MethodHead,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Head(ctx, path)
			},
		},
		{
			name:       "post",
			method:     http.MethodPost,
			body:       []byte("created"),
			expectBody: true,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Post(ctx, path, body)
			},
		},
		{
			name:       "put",
			method:     http.MethodPut,
			body:       []byte("updated"),
			expectBody: true,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Put(ctx, path, body)
			},
		},
		{
			name:       "patch",
			method:     http.MethodPatch,
			body:       []byte("patched"),
			expectBody: true,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Patch(ctx, path, body)
			},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Delete(ctx, path)
			},
		},
		{
			name:   "connect",
			method: http.MethodConnect,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Connect(ctx, path)
			},
		},
		{
			name:   "options",
			method: http.MethodOptions,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Options(ctx, path)
			},
		},
		{
			name:   "trace",
			method: http.MethodTrace,
			call: func(ctx context.Context, c *Client, path string, body []byte) ([]byte, error) {
				return c.Trace(ctx, path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("expected method %s, got %s", tt.method, r.Method)
				}

				if r.URL.Path != "/users/1" {
					t.Fatalf("expected path /users/1, got %s", r.URL.Path)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}

				if tt.expectBody && string(body) != string(tt.body) {
					t.Fatalf("expected body %q, got %q", string(tt.body), string(body))
				}

				if got := r.Header.Get("Accept"); got != jsonContentType {
					t.Fatalf("expected Accept %q, got %q", jsonContentType, got)
				}

				if tt.expectBody {
					if got := r.Header.Get("Content-Type"); got != jsonContentType {
						t.Fatalf("expected Content-Type %q, got %q", jsonContentType, got)
					}
				}

				_, _ = io.WriteString(w, "ok")
			}))
			defer server.Close()

			client := NewClient(Config{BaseURL: server.URL})

			body, err := tt.call(context.Background(), client, "users/1", tt.body)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.method == http.MethodHead {
				return
			}

			if string(body) != "ok" {
				t.Fatalf("expected body ok, got %q", string(body))
			}
		})
	}
}
