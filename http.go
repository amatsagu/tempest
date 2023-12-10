package tempest

import (
	"context"
	"net/http"
)

type HTTPServer interface {
	Close() error
	Shutdown(ctx context.Context) error
	ListenAndServe(addr string, handler http.Handler) error
	ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
