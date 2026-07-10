// Package client fornece um cliente HTTP simples com timeout, context e retry.
package client

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Config define as configurações usadas para criar um Client.
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// Client executa chamadas HTTP para uma API usando as regras de Config.
type Client struct {
	baseURL    string
	maxRetries int
	retryDelay time.Duration
	httpClient *http.Client
}

// NewClient cria uma instância de Client.
//
// O cfg informa a URL base, o timeout HTTP, a quantidade máxima de tentativas
// e o intervalo inicial entre retries.
//
// Retorna um Client pronto para fazer chamadas HTTP para a API configurada.
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:    cfg.BaseURL,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Get faz uma requisição HTTP GET para a API configurada.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	var (
		resp *http.Response
		err  error
	)

	delay := c.retryDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return nil, err
		}

		resp, err = c.httpClient.Do(req)

		if !ShouldRetry(resp, err) {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt == c.maxRetries {
			break
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		delay *= 2
	}

	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, context.Canceled
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// ShouldRetry informa se uma chamada HTTP deve ser tentada novamente.
func ShouldRetry(resp *http.Response, err error) bool {
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
