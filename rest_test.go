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
	go requestGateway(rest)
	go requestGateway(rest)
	go requestGateway(rest)
	requestGateway(rest)
	requestGateway(rest)
	requestGateway(rest)
}

func requestGateway(rest *Rest) {
	body, err := rest.Request(http.MethodGet, "/gateway/bot", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))
}
