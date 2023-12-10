package tempest

import (
	"context"
	"net/http"
)

var _ HTTPServer = (*http.Server)(nil)

type HTTPServer interface {
	Close() error
	Shutdown(ctx context.Context) error
	ListenAndServe() error
	ListenAndServeTLS(certFile string, keyFile string) error
}
