package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"
)

// doWithRetry sends an HTTP request and retries while shouldRetry indicates the
// error or response status code is retryable.
func (c *Client) doWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	maxRetries := c.maxRetries
	if !c.canRetryMethod(method) {
		maxRetries = 0
	}

	delay := c.retryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		c.logformat("httpclient: attempt %d/%d %s %s", attempt+1, maxRetries+1, method, path)

		resp, err = c.do(ctx, method, path, body)
		if !shouldRetry(resp, err) || attempt == maxRetries {
			return resp, err
		}

		c.logRetry(attempt, maxRetries, delay, resp, err)

		if resp != nil {
			resp.Body.Close()
		}

		if err := waitBackoff(ctx, delay); err != nil {
			return nil, err
		}

		delay = nextBackoff(delay)
	}

	return resp, err
}

// do sends a single HTTP request attempt without retry or backoff.
func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c.accept != "" {
		req.Header.Set("Accept", c.accept)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", c.contentType)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// logRetry logs the retry attempt and the reason for retrying.
func (c *Client) logRetry(attempt, maxRetries int, delay time.Duration, resp *http.Response, err error) {
	if err != nil {
		c.logformat("httpclient: retry %d/%d in %s after error: %v", attempt+1, maxRetries, delay, err)
		return
	}

	if resp != nil {
		c.logformat("httpclient: retry %d/%d in %s after status code: %d", attempt+1, maxRetries, delay, resp.StatusCode)
	}
}

// logformat writes library logs when a logger is configured.
func (c *Client) logformat(format string, args ...any) {
	if c.logger == nil {
		return
	}

	c.logger.Printf(format, args...)
}

// buildURL combines the base URL and path with a single slash between them.
func (c *Client) buildURL(path string) string {
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// shouldRetry reports whether an HTTP request should be retried.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}

	return false
}

func makeRetryMethods(methods []string) map[string]struct{} {
	if methods == nil {
		methods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
		}
	}

	retryMethods := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		retryMethods[strings.ToUpper(method)] = struct{}{}
	}

	return retryMethods
}

func (c *Client) canRetryMethod(method string) bool {
	_, ok := c.retryMethods[strings.ToUpper(method)]
	return ok
}

// waitBackoff waits for the provided delay before the next retry attempt.
//
// The wait also respects the context and returns immediately when the context
// is canceled or reaches its deadline.
func waitBackoff(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// nextBackoff calculates the next delay using exponential backoff.
func nextBackoff(delay time.Duration) time.Duration {
	return delay * 2
}
