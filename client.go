package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const jsonContentType = "application/json"

var (
	// ErrEmptyBaseURL is returned when Config.BaseURL is empty.
	ErrEmptyBaseURL = errors.New("httpclient: base URL is required")

	// ErrInvalidBaseURL is returned when Config.BaseURL is not a valid HTTP URL.
	ErrInvalidBaseURL = errors.New("httpclient: base URL must be a valid http or https URL")
)

// Config defines the settings used by NewClient.
type Config struct {
	// BaseURL is the base address used to build requests.
	//
	// Example:
	//
	//	BaseURL: "https://api.example.com"
	//
	// Paths passed to methods such as Get and Post may include or omit the
	// leading slash. Both "users" and "/users" become
	// "https://api.example.com/users". BaseURL must be a valid absolute http or
	// https URL.
	BaseURL string

	// Timeout is applied to the default http.Client created by NewClient.
	//
	// This field is ignored when HTTPClient is provided. In that case, configure
	// the timeout directly in the custom *http.Client.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts after the first request.
	//
	// For example, MaxRetries: 3 may execute up to 4 attempts total: the first
	// request plus 3 retries. Retries only happen for methods allowed by
	// RetryMethods and for retryable network errors or status codes.
	MaxRetries int

	// RetryDelay is the initial wait time before the first retry.
	//
	// The delay is increased with exponential backoff after each failed attempt.
	RetryDelay time.Duration

	// Logger receives optional retry and attempt logs.
	//
	// When Logger is nil, the client does not write logs.
	Logger *log.Logger

	// Accept sets the Accept header for every request.
	//
	// When empty, it defaults to "application/json".
	Accept string

	// ContentType sets the Content-Type header for requests with a body.
	//
	// When empty, it defaults to "application/json". The header is only sent
	// when the request body is not empty.
	ContentType string

	// Headers defines additional headers sent with every request.
	//
	// Use this field for authentication and custom headers such as
	// Authorization, X-API-Key, X-Client-ID, tenant IDs, trace IDs, and similar
	// values. Headers are applied after Accept and ContentType, so entries such
	// as "Accept" or "Content-Type" in this map override those fields.
	Headers map[string]string

	// HTTPClient allows using a custom *http.Client.
	//
	// Use this field for custom TLS settings, mTLS, proxies, transports,
	// redirects, cookies, or tests with a fake RoundTripper. When HTTPClient is
	// nil, NewClient creates a default *http.Client using Timeout.
	HTTPClient *http.Client

	// RetryMethods defines which HTTP methods may be retried.
	//
	// When nil, the default retryable methods are GET, HEAD, and OPTIONS. This
	// avoids retrying non-idempotent methods such as POST, PUT, and PATCH unless
	// explicitly configured. Use an empty slice to disable retries by method.
	RetryMethods []string

	// RetryStatusCodes defines which HTTP response status codes may be retried.
	//
	// When nil, the default retryable status codes are 429, 500, 502, 503, and
	// 504. Network errors may still be retried when the request method is
	// retryable. Use an empty slice to disable retries based on status codes.
	RetryStatusCodes []int

	// MaxResponseBodySize limits how many bytes can be read from a response body.
	//
	// When zero or negative, response bodies are read without a library-defined
	// limit. When positive, responses larger than this value return a
	// *ResponseBodyTooLargeError.
	MaxResponseBodySize int64
}

// Client sends HTTP requests to an API using the rules defined by Config.
type Client struct {
	baseURL             string
	maxRetries          int
	retryDelay          time.Duration
	logger              *log.Logger
	accept              string
	contentType         string
	headers             map[string]string
	httpClient          *http.Client
	retryMethods        map[string]struct{}
	retryStatusCodes    map[int]struct{}
	maxResponseBodySize int64
}

// NewClient creates a Client using cfg.
//
// The returned client uses BaseURL to build request URLs, applies default
// Accept and Content-Type headers when they are not configured, copies Headers
// so later changes to cfg.Headers do not affect the client, and uses either the
// provided HTTPClient or a default *http.Client configured with Timeout.
// NewClient returns an error when BaseURL is empty, invalid, or does not use
// the http or https scheme.
//
// Retry behavior is controlled by MaxRetries, RetryDelay, RetryMethods, and
// RetryStatusCodes. By default, only GET, HEAD, and OPTIONS are retried.
// Configure RetryMethods explicitly to allow retries for methods such as POST,
// PUT, PATCH, or DELETE. MaxResponseBodySize can be used to prevent reading
// response bodies larger than the configured number of bytes.
func NewClient(cfg Config) (*Client, error) {
	baseURL, err := validateBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	accept := cfg.Accept
	if accept == "" {
		accept = jsonContentType
	}

	contentType := cfg.ContentType
	if contentType == "" {
		contentType = jsonContentType
	}

	headers := make(map[string]string, len(cfg.Headers))
	for key, value := range cfg.Headers {
		headers[key] = value
	}

	retryMethods := makeRetryMethods(cfg.RetryMethods)
	retryStatusCodes := makeRetryStatusCodes(cfg.RetryStatusCodes)

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &Client{
		baseURL:             baseURL,
		maxRetries:          cfg.MaxRetries,
		retryDelay:          cfg.RetryDelay,
		logger:              cfg.Logger,
		accept:              accept,
		contentType:         contentType,
		headers:             headers,
		httpClient:          httpClient,
		retryMethods:        retryMethods,
		retryStatusCodes:    retryStatusCodes,
		maxResponseBodySize: cfg.MaxResponseBodySize,
	}, nil
}

func validateBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", ErrEmptyBaseURL
	}

	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidBaseURL, baseURL)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("%w: %q", ErrInvalidBaseURL, baseURL)
	}

	return strings.TrimRight(baseURL, "/"), nil
}

// request sends an HTTP request using the provided method, path, and body.
//
// The request respects the provided context, applies the configured retry
// policy, and returns the final response body as bytes.
func (c *Client) request(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	resp, err := c.doWithRetry(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, context.Canceled
	}

	defer resp.Body.Close()

	responseBody, err := readResponseBody(resp.Body, c.maxResponseBodySize)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       responseBody,
		}
	}

	return responseBody, nil
}

func readResponseBody(body io.Reader, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		return io.ReadAll(body)
	}

	responseBody, err := io.ReadAll(io.LimitReader(body, maxSize+1))
	if err != nil {
		return nil, err
	}

	if int64(len(responseBody)) > maxSize {
		return nil, &ResponseBodyTooLargeError{
			MaxSize: maxSize,
		}
	}

	return responseBody, nil
}

// Get sends an HTTP GET request to path.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodGet, path, nil)
}

// Head sends an HTTP HEAD request to path.
func (c *Client) Head(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodHead, path, nil)
}

// Post sends an HTTP POST request to path with body.
func (c *Client) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.request(ctx, http.MethodPost, path, body)
}

// Put sends an HTTP PUT request to path with body.
func (c *Client) Put(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.request(ctx, http.MethodPut, path, body)
}

// Patch sends an HTTP PATCH request to path with body.
func (c *Client) Patch(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.request(ctx, http.MethodPatch, path, body)
}

// Delete sends an HTTP DELETE request to path.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodDelete, path, nil)
}

// Options sends an HTTP OPTIONS request to path.
func (c *Client) Options(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, http.MethodOptions, path, nil)
}
