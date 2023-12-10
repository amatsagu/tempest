package tempest

import (
	"context"
)

type HTTPServer interface {
	Close() error
	Shutdown(ctx context.Context) error
	ListenAndServe() error
	ListenAndServeTLS(certFile string, keyFile string) error
}
