package httpclient

import "fmt"

// HTTPError represents an HTTP response with a status code outside the 2xx range.
type HTTPError struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int

	// Body is the response body returned by the server.
	Body []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("httpclient: unexpected status code %d: %s", e.StatusCode, string(e.Body))
}
