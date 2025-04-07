package tempest

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (client *Client) DiscordRequestHandler(w http.ResponseWriter, r *http.Request) {
	verified := verifyRequest(r, ed25519.PublicKey(client.PublicKey))
	if !verified {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	buf := client.jsonBufferPool.Get().(*[]byte)
	defer client.jsonBufferPool.Put(buf)

	n, err := r.Body.Read(*buf)
	if err != nil && err != io.EOF {
		http.Error(w, "bad request - failed to read body payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var extractor InteractionTypeExtractor
	if err := json.Unmarshal((*buf)[:n], &extractor); err != nil {
		http.Error(w, "bad request - invalid body json payload", http.StatusBadRequest)
		return
	}

	switch extractor.Type {
	case PING_INTERACTION_TYPE:
		fmt.Println("Got ping!")
		w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
		w.Write(bodyPingResponse)
		return
	case APPLICATION_COMMAND_INTERACTION_TYPE:
		var interaction CommandInteraction
		if err := json.Unmarshal((*buf)[:n], &interaction); err != nil {
			http.Error(w, "bad request - failed to decode CommandInteraction", http.StatusBadRequest)
			return
		}

		itx, cmd, available := client.CommandRegistry.HandleInteraction(interaction)
		if !available {
			w.Header().Add("Content-Type", CONTENT_TYPE_JSON)
			w.Write(bodyUnknownCommandResponse)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		itx.Client = client

		if client.CommandRegistry.PreCommandHook(itx) {
			cmd.CommandHandler(itx)
		}

		client.CommandRegistry.PostCommandHook(itx)
		return
	}
}
