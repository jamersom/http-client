# http-client

Cliente HTTP simples em Go para consumir APIs com timeout, contexto, retries, headers padrao e headers personalizados.

## Recursos

- Base URL configuravel.
- Timeout por requisicao usando `http.Client`.
- Suporte a `context.Context` para cancelamento e deadline.
- Retry automatico para erros temporarios.
- Exponential backoff entre tentativas.
- Headers `Accept` e `Content-Type` configuraveis.
- Headers personalizados, como `Authorization`, `X-API-Key`, `X-Client-ID`, etc.
- Atalhos para metodos HTTP comuns: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head` e `Options`.
- Logger opcional para acompanhar tentativas e retries.

## Instalacao

```bash
go get github.com/jamersom/http-client
```

## Exemplo de Uso

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	httpclient "github.com/jamersom/http-client"
)

func main() {
	client := httpclient.NewClient(httpclient.Config{
		BaseURL:    "http://localhost:8080",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
		Logger:     log.Default(),
		Headers: map[string]string{
			"Authorization": "Bearer 550e8400-e29b-41d4-a716-446655440000",
			"X-API-Key":     "550e8400-e29b-41d4-a716-446655440000",
		},
	})

	ctx := context.Background()

	body, err := client.Get(ctx, "/users")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(body))
}
```

## Exemplo com POST

```go
body := []byte(`{"name":"Joao"}`)

resp, err := client.Post(context.Background(), "/users", body)
if err != nil {
	log.Fatal(err)
}

fmt.Println(string(resp))
```

## Configuracao

```go
type Config struct {
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	Logger      *log.Logger
	Accept      string
	ContentType string
	Headers     map[string]string
}
```

## Campos Obrigatorios

### `BaseURL`

URL base da API.

Exemplo:

```go
BaseURL: "https://api.example.com"
```

As chamadas podem ser feitas com path com ou sem barra inicial:

```go
client.Get(ctx, "users")
client.Get(ctx, "/users")
```

Ambos geram:

```text
https://api.example.com/users
```

## Campos Opcionais

### `Timeout`

Tempo maximo para uma requisicao HTTP.

```go
Timeout: 5 * time.Second
```

Se nao informado, o `http.Client` fica sem timeout configurado. Em aplicacoes reais, recomenda-se sempre definir um timeout.

### `MaxRetries`

Quantidade maxima de novas tentativas em caso de erro temporario.

```go
MaxRetries: 3
```

Com `MaxRetries: 3`, o client pode executar ate 4 tentativas no total: a primeira chamada mais 3 retries.

### `RetryDelay`

Intervalo inicial entre retries.

```go
RetryDelay: 500 * time.Millisecond
```

O intervalo usa exponential backoff. Por exemplo, com `500ms`, as proximas esperas serao `1s`, `2s`, e assim por diante.

### `Logger`

Logger opcional para visualizar tentativas e retries.

```go
Logger: log.Default()
```

Se nao for informado, o client nao imprime logs.

### `Accept`

Define o header `Accept`, ou seja, o tipo de resposta esperado.

```go
Accept: "application/json"
```

Valor padrao:

```text
application/json
```

### `ContentType`

Define o header `Content-Type` para requisicoes com body, como `POST`, `PUT` e `PATCH`.

```go
ContentType: "application/json"
```

Valor padrao:

```text
application/json
```

O `Content-Type` so e enviado quando a requisicao possui body.

### `Headers`

Mapa para headers personalizados.

Use este campo para autenticacao, API keys, identificadores de cliente, tenant, trace ID, etc.

```go
Headers: map[string]string{
	"Authorization": "Bearer token",
	"X-API-Key":     "minha-chave",
	"X-Client-ID":   "meu-client-id",
}
```

Como `Headers` e aplicado depois de `Accept` e `ContentType`, ele tambem pode sobrescrever esses valores:

```go
Headers: map[string]string{
	"Accept": "text/plain",
}
```

## Retries

O client tenta novamente quando ocorre erro de rede ou quando a API retorna estes status:

- `429 Too Many Requests`
- `500 Internal Server Error`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`

O retry respeita o `context.Context`. Se o contexto for cancelado ou atingir deadline, a espera entre retries e a requisicao sao interrompidas.

## Metodos Disponiveis

```go
client.Get(ctx, "/users")
client.Post(ctx, "/users", body)
client.Put(ctx, "/users/1", body)
client.Patch(ctx, "/users/1", body)
client.Delete(ctx, "/users/1")
client.Head(ctx, "/users")
client.Options(ctx, "/users")
```

## Observacoes

- A biblioteca retorna o corpo da resposta como `[]byte`.
- O status HTTP nao e convertido automaticamente em erro. Se a API retornar `401`, `404` ou outro status, o corpo ainda sera retornado quando nao houver erro de rede.
- Para APIs HTTPS com certificado publico valido, o Go ja valida o certificado automaticamente.
- Para certificados internos, self-signed ou mTLS, sera necessario evoluir a lib para aceitar um `*http.Client` customizado.
