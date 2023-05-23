package tempest

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

// Spams any request to check for Rest race conditions.
func TestRest(t *testing.T) {
	rest := NewRest("Bot " + os.Getenv("BOT_TOKEN"))
	go requestGateway(rest, t)
	go requestGateway(rest, t)
	go requestGateway(rest, t)
	requestGateway(rest, t)
	requestGateway(rest, t)
	requestGateway(rest, t)
}

func requestGateway(rest *Rest, t *testing.T) {
	body, err := rest.Request(http.MethodGet, "/gateway/bot", nil)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(body))
}
