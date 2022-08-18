package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
)

func PrettyStructPrint(v any) {
	str, _ := json.MarshalIndent(v, "", "  ")
	println(string(str))
}

func CheckInSlice[T comparable](slice *[]T, item T) bool {
	for _, value := range *slice {
		if value == item {
			return true
		}
	}
	return false
}

// Makes Go's compiler thing be whatever type you want. You should avoid relying on it!
func JsonReshape[T interface{}](object interface{}) T {
	var shape T

	raw, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(raw, shape)
	return shape
}

// Simply reflects any data to type you need. Returns second param "true" when data is safe to use. You should avoid relying on it!
func Reshape[T any](value any) (T, bool) {
	v, safe := value.(T)
	return v, safe
}

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

func parseCommandsToDiscordObjects(client *Client, commandsToInclude []string) []interface{} {
	list := make([]interface{}, len(client.commands))
	ip := 0

	for _, tree := range client.commands {
		root := tree["-"]

		for key, subCommand := range tree {
			if key == "-" {
				continue
			}

			root.Options = append(root.Options, Option{
				Name:        subCommand.Name,
				Description: subCommand.Description,
				Type:        OPTION_SUB_COMMAND,
				Options:     subCommand.Options,
			})
		}

		list[ip] = root
		ip++
	}

	if len(commandsToInclude) != 0 {
		fList := make([]interface{}, len(client.commands))
		ip := 0

		for _, command := range list {
			cmd := command.(Command)

			if CheckInSlice(&commandsToInclude, cmd.Name) {
				fList[ip] = cmd
				ip++
			}
		}

		return fList
	}

	return list
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
