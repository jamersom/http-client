# http-client

Client HTTP simples em Go para consumir APIs com timeout, contexto, retries, headers padrao e headers personalizados.

## Recursos

- Base URL configuravel.
- Timeout por requisicao usando `http.Client`.
- Suporte a `context.Context` para cancelamento e deadline.
- Retry automatico para erros temporarios.
- Exponential backoff entre tentativas.
- Headers `Accept` e `Content-Type` configuraveis.
- Headers personalizados, como `Authorization`, `X-API-Key`, `X-Client-ID`, etc.
- Suporte a `*http.Client` customizado para certificados, proxy, mTLS e transportes especificos.
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
	"errors"
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
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			log.Fatalf("erro HTTP %d: %s", httpErr.StatusCode, string(httpErr.Body))
		}

		log.Fatal(err)
	}

	fmt.Println(string(body))
}
```

## Criando o Client

Use `NewClient` passando um `httpclient.Config`.

O unico campo obrigatorio e `BaseURL`. Os demais campos sao opcionais e podem ser configurados conforme a necessidade da API.

```go
client := httpclient.NewClient(httpclient.Config{
	BaseURL: "https://api.example.com",

	Timeout:    5 * time.Second,
	MaxRetries: 3,
	RetryDelay: 500 * time.Millisecond,

	RetryMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
	},

	RetryStatusCodes: []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	},

	Logger: log.Default(),

	Accept:      "application/json",
	ContentType: "application/json",

	Headers: map[string]string{
		"Authorization": "Bearer token",
		"X-API-Key":     "api-key",
		"X-Client-ID":   "client-id",
	},
})
```

Se precisar de certificado, proxy, mTLS ou `Transport` customizado, crie um `*http.Client` e informe em `HTTPClient`:

```go
customHTTPClient := &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
}

client := httpclient.NewClient(httpclient.Config{
	BaseURL:    "https://api.example.com",
	HTTPClient: customHTTPClient,
})
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
	BaseURL          string
	Timeout          time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	Logger           *log.Logger
	Accept           string
	ContentType      string
	Headers          map[string]string
	HTTPClient       *http.Client
	RetryMethods     []string
	RetryStatusCodes []int
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

### `RetryMethods`

Define quais metodos HTTP podem ser repetidos automaticamente em caso de erro temporario.

Por padrao, apenas metodos considerados seguros fazem retry:

```go
GET
HEAD
OPTIONS
```

Isso evita repetir automaticamente chamadas como `POST`, `PUT` e `PATCH`, que podem criar, alterar ou duplicar dados no servidor.

Para permitir retry em outro metodo, configure explicitamente:

```go
client := httpclient.NewClient(httpclient.Config{
	BaseURL: "https://api.example.com",
	MaxRetries: 3,
	RetryDelay: 500 * time.Millisecond,
	RetryMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPost,
	},
})
```

Para desabilitar retries por metodo, informe uma lista vazia:

```go
RetryMethods: []string{}
```

### `RetryStatusCodes`

Define quais status HTTP podem disparar retry.

Por padrao, estes status fazem retry:

- `429 Too Many Requests`
- `500 Internal Server Error`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`

Para usar outra lista, configure explicitamente:

```go
client := httpclient.NewClient(httpclient.Config{
	BaseURL: "https://api.example.com",
	MaxRetries: 3,
	RetryDelay: 500 * time.Millisecond,
	RetryStatusCodes: []int{
		http.StatusConflict,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
	},
})
```

Para desabilitar retries baseados em status HTTP, informe uma lista vazia:

```go
RetryStatusCodes: []int{}
```

Erros de rede ainda podem ser repetidos quando o metodo da requisicao estiver permitido em `RetryMethods`.

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

### `HTTPClient`

Permite informar um `*http.Client` customizado.

Use este campo quando precisar configurar certificado, proxy, mTLS, `Transport` customizado ou alguma politica especifica do Go para conexões HTTP.

```go
customHTTPClient := &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
}

client := httpclient.NewClient(httpclient.Config{
	BaseURL:    "https://api.example.com",
	HTTPClient: customHTTPClient,
})
```

Quando `HTTPClient` e informado, o campo `Timeout` do `Config` nao e aplicado automaticamente. Nesse caso, configure o timeout diretamente no `*http.Client` customizado.

## Retries

O client tenta novamente quando ocorre erro de rede ou quando a API retorna estes status:

- `429 Too Many Requests`
- `500 Internal Server Error`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`

Por padrao, retries acontecem apenas para `GET`, `HEAD` e `OPTIONS`. Para liberar retries em metodos como `POST`, configure `RetryMethods`. Para alterar os status retryaveis, configure `RetryStatusCodes`.

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

## Tratamento de Erros HTTP

Quando a API responde com status fora da faixa `2xx`, o client retorna um `*httpclient.HTTPError`.

```go
body, err := client.Get(ctx, "/users")
if err != nil {
	var httpErr *httpclient.HTTPError
	if errors.As(err, &httpErr) {
		fmt.Printf("status: %d\n", httpErr.StatusCode)
		fmt.Printf("body: %s\n", string(httpErr.Body))
		return
	}

	log.Fatal(err)
}

fmt.Println(string(body))
```

Exemplos de status que viram erro:

- `400 Bad Request`
- `401 Unauthorized`
- `404 Not Found`
- `500 Internal Server Error`

## Observacoes

- A biblioteca retorna o corpo da resposta como `[]byte`.
- Respostas com status `2xx` sao consideradas sucesso.
- Respostas fora da faixa `2xx` retornam `*httpclient.HTTPError`.
- Para APIs HTTPS com certificado publico valido, o Go ja valida o certificado automaticamente.
- Para certificados internos, self-signed, proxy ou mTLS, use o campo `HTTPClient` com um `Transport` customizado.
