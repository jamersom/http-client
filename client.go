package httpclient

import (
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
