package httpclient

import "fmt"

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("httpclient: unexpected status code %d: %s", e.StatusCode, string(e.Body))
}
