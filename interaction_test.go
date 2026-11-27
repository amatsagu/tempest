package tempest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
)

func testBotToken(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id)) + ".test.sig"
}

func signedInteractionRequest(t *testing.T, priv ed25519.PrivateKey, body string) *http.Request {
	t.Helper()

	timestamp := "1700000000"
	msg := append([]byte(timestamp), body...)
	sig := ed25519.Sign(priv, msg)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	req.Header.Set("X-Signature-Timestamp", timestamp)
	return req
}

func TestHTTPClientDiscordRequestHandler(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewHTTPClient(HTTPClientOptions{
		PublicKey: hex.EncodeToString(pub),
		BaseClientOptions: BaseClientOptions{
			Token: testBotToken("123456789012345678"),
		},
	})

	err = client.RegisterCommand(Command{
		Command: discord.Command{
			Name:        "greet",
			Description: "Greet someone",
			Options: []discord.CommandOption{
				{Name: "name", Description: "Who", Type: discord.StringOptionType, AutoComplete: true},
			},
		},
		SlashCommandHandler: func(itx *CommandInteraction) {
			name, _ := itx.GetOptionValue("name")
			_ = itx.SendLinearReply("Hi "+name.(string), true)
		},
		AutoCompleteHandler: func(itx CommandInteraction) []discord.CommandOptionChoice {
			_, val := itx.GetFocusedValue()
			return []discord.CommandOptionChoice{
				{Name: "Alice", Value: val.(string) + "alice"},
			}
		},
	})
	if err != nil {
		t.Fatalf("RegisterCommand: %v", err)
	}

	if err := client.RegisterComponent([]string{"btn:ok"}, func(itx ComponentInteraction) {
		_ = itx.AcknowledgeWithLinearUpdate("updated")
	}); err != nil {
		t.Fatalf("RegisterComponent: %v", err)
	}

	if err := client.RegisterModal("mdl:form", func(itx ModalInteraction) {
		val := itx.GetInputValue("field:1")
		_ = itx.AcknowledgeWithLinearMessage("got:"+val, true)
	}); err != nil {
		t.Fatalf("RegisterModal: %v", err)
	}

	t.Run("Ping", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := signedInteractionRequest(t, priv, `{"type":1}`)
		client.DiscordRequestHandler(rec, req)

		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), bodyPingResponse) {
			t.Fatalf("ping response = (%d, %s), want (200, %s)", rec.Code, rec.Body.String(), bodyPingResponse)
		}
	})

	t.Run("SlashCommand", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":2,"token":"tok","app_permissions":"0","data":{"id":"2","name":"greet","type":1,"options":[{"name":"name","type":3,"value":"World"}]}}`
		client.DiscordRequestHandler(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
		}

		var resp discord.ResponseMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Type != discord.ChannelMessageWithSourceResponseType || resp.Data == nil || resp.Data.Content != "Hi World" || resp.Data.Flags != discord.EphemeralMessageFlag {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("AutoComplete", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":4,"token":"tok","app_permissions":"0","data":{"id":"2","name":"greet","type":1,"options":[{"name":"name","type":3,"value":"a","focused":true}]}}`
		client.DiscordRequestHandler(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var resp discord.ResponseAutoComplete
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal autocomplete: %v", err)
		}
		if resp.Type != discord.AutocompleteResponseType || resp.Data == nil || len(resp.Data.Choices) != 1 || resp.Data.Choices[0].Value != "aalice" {
			t.Fatalf("unexpected autocomplete response: %+v", resp)
		}
	})

	t.Run("Component", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":3,"token":"tok","app_permissions":"0","data":{"custom_id":"btn:ok","component_type":2}}`
		client.DiscordRequestHandler(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var resp discord.ResponseMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal component response: %v", err)
		}
		if resp.Type != discord.UpdateMessageResponseType || resp.Data == nil || resp.Data.Content != "updated" {
			t.Fatalf("unexpected component response: %+v", resp)
		}
	})

	t.Run("ModalSubmit", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":5,"token":"tok","app_permissions":"0","data":{"custom_id":"mdl:form","components":[{"type":18,"label":"L","component":{"type":4,"style":1,"custom_id":"field:1","value":"abc"}}]}}`
		client.DiscordRequestHandler(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var resp discord.ResponseMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal modal response: %v", err)
		}
		if resp.Type != discord.ChannelMessageWithSourceResponseType || resp.Data == nil || resp.Data.Content != "got:abc" {
			t.Fatalf("unexpected modal response: %+v", resp)
		}
	})
}

func TestParseCommandsForDiscordAPIAndFindInteractiveComponent(t *testing.T) {
	t.Parallel()

	base := NewBaseClient(BaseClientOptions{Token: testBotToken("123456789012345678")})
	if err := base.RegisterCommand(Command{
		Command: discord.Command{Name: "tag", Description: "Root"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := base.RegisterSubCommand(Command{
		Command: discord.Command{Name: "add", Description: "Add"},
	}, "tag"); err != nil {
		t.Fatal(err)
	}

	parsed := parseCommandsForDiscordAPI(base.commands, nil, false)
	if len(parsed) != 1 || parsed[0].Name != "tag" || len(parsed[0].Options) != 1 {
		t.Fatalf("parsed commands = %+v, want 1 root command with 1 subcommand option", parsed)
	}
	if parsed[0].Options[0].Type != discord.SubCommandOptionType || parsed[0].Options[0].Name != "add" {
		t.Fatalf("subcommand option = %+v, want SubCommandOptionType add", parsed[0].Options[0])
	}

	tree := []discord.MessageComponent{
		discord.ContainerComponent{
			Type: discord.ContainerComponentType,
			Components: []discord.ContainerChildComponent{
				discord.SectionComponent{
					Type:      discord.SectionComponentType,
					Accessory: discord.ThumbnailComponent{Type: discord.ThumbnailComponentType},
				},
				discord.ActionRowComponent{
					Type: discord.ActionRowComponentType,
					Components: []discord.ActionRowChildComponent{
						discord.ButtonComponent{Type: discord.ButtonComponentType, CustomID: "target"},
					},
				},
			},
		},
	}

	btn, found := FindInteractiveComponent(tree, func(b discord.ButtonComponent) bool {
		return b.CustomID == "target"
	})
	if !found || btn.CustomID != "target" {
		t.Fatalf("FindInteractiveComponent = (%+v, %t), want target button", btn, found)
	}
}
