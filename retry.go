package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"
)

// doWithRetry executa a requisicao HTTP repetindo tentativas quando shouldRetry
// indicar que o erro ou status code da resposta permite retry.
func (c *Client) doWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	delay := c.retryDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		c.logformat("httpclient: attempt %d/%d %s %s", attempt+1, c.maxRetries+1, method, path)

		resp, err = c.do(ctx, method, path, body)
		if !shouldRetry(resp, err) || attempt == c.maxRetries {
			return resp, err
		}

		c.logRetry(attempt, delay, resp, err)

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

// do executa uma unica tentativa HTTP sem aplicar retry ou backoff.
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

// logRetry registra a tentativa de retry e o motivo da nova tentativa.
func (c *Client) logRetry(attempt int, delay time.Duration, resp *http.Response, err error) {
	if err != nil {
		c.logformat("httpclient: retry %d/%d in %s after error: %v", attempt+1, c.maxRetries, delay, err)
		return
	}

	if resp != nil {
		c.logformat("httpclient: retry %d/%d in %s after status code: %d", attempt+1, c.maxRetries, delay, resp.StatusCode)
	}
}

// logformat registra mensagens da lib quando um logger foi configurado.
func (c *Client) logformat(format string, args ...any) {
	if c.logger == nil {
		return
	}

	c.logger.Printf(format, args...)
}

// buildURL combina a URL base com o path mantendo uma unica barra entre eles.
func (c *Client) buildURL(path string) string {
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// shouldRetry informa se uma chamada HTTP deve ser tentada novamente.
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

// waitBackoff aguarda o intervalo informado antes da proxima tentativa.
//
// A espera tambem respeita o contexto, retornando imediatamente quando ele for
// cancelado ou atingir timeout.
func waitBackoff(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// nextBackoff calcula o proximo intervalo usando exponential backoff.
func nextBackoff(delay time.Duration) time.Duration {
	return delay * 2
}
