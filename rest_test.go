package tempest

import (
	"net/http"
	"os"
	"testing"
)

// Spams any request to check for Rest race conditions.
func TestRest(t *testing.T) {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		t.Skip("can't test rest due to no provided token")
	}

	rest := NewRest(token)
	go requestGateway(rest, t)
	go requestGateway(rest, t)
	go requestGateway(rest, t)
	requestGateway(rest, t)
	requestGateway(rest, t)
	requestGateway(rest, t)
}

func requestGateway(rest *Rest, t *testing.T) {
	_, err := rest.Request(http.MethodGet, "/gateway/bot", nil)
	if err != nil {
		t.Error(err)
	}
}
