package tempest

import (
	"context"
	"net/http"
)

var _ HTTPServer = (*http.Server)(nil)
var _ HTTPServeMux = (*http.ServeMux)(nil)

type HTTPServer interface {
	Close() error
	Shutdown(ctx context.Context) error
	ListenAndServe() error
	ListenAndServeTLS(certFile string, keyFile string) error
}

type HTTPServeMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	Handler(r *http.Request) (h http.Handler, pattern string)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
