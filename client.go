package httpclient

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
)

// Config define as configurações usadas para criar um Client.
type Config struct {
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	Logger      *log.Logger
	Accept      string
	ContentType string
}

// Client executa chamadas HTTP para uma API usando as regras de Config.
type Client struct {
	baseURL     string
	maxRetries  int
	retryDelay  time.Duration
	httpClient  *http.Client
	logger      *log.Logger
	accept      string
	contentType string
}

// NewClient cria uma instância de Client.
//
// O cfg informa a URL base, o timeout HTTP, a quantidade máxima de tentativas
// e o intervalo inicial entre retries.
//
// Retorna um Client pronto para fazer chamadas HTTP para a API configurada.
func NewClient(cfg Config) *Client {
	accept := cfg.Accept
	if accept == "" {
		accept = jsonContentType
	}

	contentType := cfg.ContentType
	if contentType == "" {
		contentType = jsonContentType
	}

	return &Client{
		baseURL:     cfg.BaseURL,
		maxRetries:  cfg.MaxRetries,
		retryDelay:  cfg.RetryDelay,
		logger:      cfg.Logger,
		accept:      accept,
		contentType: contentType,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Request executa uma chamada HTTP usando o metodo, path e body informados.
//
// A chamada respeita o contexto recebido, aplica a politica de retry configurada
// no Client e retorna o corpo da resposta final como bytes.
func (c *Client) Request(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	resp, err := c.doWithRetry(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, context.Canceled
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Get executa uma chamada HTTP GET para o path informado.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodGet, path, nil)
}

// Head executa uma chamada HTTP HEAD para o path informado.
func (c *Client) Head(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodHead, path, nil)
}

// Post executa uma chamada HTTP POST para o path informado.
func (c *Client) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.Request(ctx, http.MethodPost, path, body)
}

// Put executa uma chamada HTTP PUT para o path informado.
func (c *Client) Put(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.Request(ctx, http.MethodPut, path, body)
}

// Patch executa uma chamada HTTP PATCH para o path informado.
func (c *Client) Patch(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.Request(ctx, http.MethodPatch, path, body)
}

// Delete executa uma chamada HTTP DELETE para o path informado.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodDelete, path, nil)
}

// Connect executa uma chamada HTTP CONNECT para o path informado.
func (c *Client) Connect(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodConnect, path, nil)
}

// Options executa uma chamada HTTP OPTIONS para o path informado.
func (c *Client) Options(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodOptions, path, nil)
}

// Trace executa uma chamada HTTP TRACE para o path informado.
func (c *Client) Trace(ctx context.Context, path string) ([]byte, error) {
	return c.Request(ctx, http.MethodTrace, path, nil)
}
