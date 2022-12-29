package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

// Verifies incoming request if it's from Discord.
func verifyRequest(r *http.Request, key ed25519.PublicKey) bool {
	var msg bytes.Buffer

	signature := r.Header.Get("X-Signature-Ed25519")
	if signature == "" {
		return false
	}

	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return false
	}

	timestamp := r.Header.Get("X-Signature-Timestamp")
	if timestamp == "" {
		return false
	}

	msg.WriteString(timestamp)

	defer r.Body.Close()
	var body bytes.Buffer

	// Copy the original body back into the request after finishing.
	defer func() {
		r.Body = io.NopCloser(&body)
	}()

	// Copy body into buffers
	_, err = io.Copy(&msg, io.TeeReader(r.Body, &body))
	if err != nil {
		return false
	}

	return ed25519.Verify(key, msg.Bytes(), sig)
}

func parseCommandsToDiscordObjects(commands map[string]map[string]Command, whitelist []string, reverseMode bool) []Command {
	list := make([]Command, len(commands))
	var itx uint32 = 0

	for _, tree := range commands {
		command := tree["-"]

		for key, subCommand := range tree {
			if key == "-" {
				continue
			}

			command.Options = append(command.Options, Option{
				Name:        subCommand.Name,
				Description: subCommand.Description,
				Type:        OPTION_SUB_COMMAND,
				Options:     subCommand.Options,
			})
		}

		list[itx] = command
		itx++
	}

	wls := len(whitelist)
	if wls == 0 {
		return list
	}

	itx = 0

	// Work as blacklist
	if reverseMode {
		filteredList := make([]Command, len(commands)-wls)

		for itx, command := range list {
			blocked := false
			for _, cmdName := range whitelist {
				if command.Name == cmdName {
					blocked = true
					break
				}
			}

			if blocked {
				continue
			}

			filteredList[itx] = command
		}

		return filteredList
	}

	// Work as whitelist
	filteredList := make([]Command, wls)

	for _, command := range list {
		for _, cmdName := range whitelist {
			if command.Name == cmdName {
				filteredList[itx] = command
				itx++
			}
		}
	}

	return filteredList
}

func terminateCommandInteraction(w http.ResponseWriter) {
	body, err := json.Marshal(Response{
		Type: CHANNEL_MESSAGE_WITH_SOURCE_RESPONSE,
		Data: &ResponseData{
			Content: "Oh snap! It looks like you tried to trigger (/) command which is not registered within local cache. Please report this bug to my master.",
		},
	})

	if err != nil {
		panic("failed to parse json payload")
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(body)
}

func acknowledgeComponentInteraction(w http.ResponseWriter) {
	body, err := json.Marshal(Response{
		Type: DEFERRED_UPDATE_MESSAGE_RESPONSE,
	})

	if err != nil {
		panic("failed to parse json payload")
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(body)
}
