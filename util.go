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

func parseCommandsToDiscordObjects(client *client, commandsToInclude []string) []Command {
	var list []Command

	for _, tree := range client.commands {
		root := tree["-"]

		for key, subCommand := range tree {
			if key == "-" {
				continue
			}

			sub := Reshape[Command](subCommand)
			sub.Type = CommandType(OPTION_SUB_COMMAND)
			root.Options = append(root.Options, Reshape[Option](subCommand))
		}

		list = append(list, root)
	}

	if len(commandsToInclude) != 0 {
		var fList []Command

		for _, command := range list {
			if CheckInSlice(&commandsToInclude, command.Name) {
				fList = append(fList, command)
			}
		}

		return fList
	}

	return list
}

func CheckInSlice[T comparable](slice *[]T, item T) bool {
	for _, value := range *slice {
		if value == item {
			return true
		}
	}
	return false
}

// Makes Go's compiler thing it's whatever type you want. It's a trick for lazy people but should be avoid...
func Reshape[T any](s any) T {
	return s.(T)
}
