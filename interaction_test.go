package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
)

func testBotToken(id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(id)) + ".test.sig"
}

func newTestClientPair(t *testing.T, hooks ClientHooks) (*Client, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(ClientConfig{
		Token:      testBotToken("123456789012345678"),
		PublicKey:  hex.EncodeToString(pub),
		AckTimeout: 500 * time.Millisecond,
		Hooks:      hooks,
	})
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = client.Shutdown(shutdownCtx)
	})

	return client, priv
}

func signedInteractionRequest(t *testing.T, priv ed25519.PrivateKey, body string) *http.Request {
	t.Helper()

	timestamp := "1700000000"
	msg := append([]byte(timestamp), body...)
	sig := ed25519.Sign(priv, msg)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	req.Header.Set("X-Signature-Timestamp", timestamp)
	return req
}

func TestClientHTTPInteractions(t *testing.T) {
	t.Parallel()

	var (
		preCmdCount  atomic.Int32
		postCmdCount atomic.Int32
		preCmpCount  atomic.Int32
		postCmpCount atomic.Int32
		preMdlCount  atomic.Int32
		postMdlCount atomic.Int32
	)

	client, priv := newTestClientPair(t, ClientHooks{
		PreCommand: func(ctx context.Context, cmd Command, itx *CommandInteraction) bool {
			preCmdCount.Add(1)
			return true
		},
		PostCommand: func(ctx context.Context, cmd Command, itx *CommandInteraction) {
			postCmdCount.Add(1)
		},
		PreComponent: func(ctx context.Context, itx *ComponentInteraction) bool {
			preCmpCount.Add(1)
			return true
		},
		PostComponent: func(ctx context.Context, itx *ComponentInteraction) {
			postCmpCount.Add(1)
		},
		PreModal: func(ctx context.Context, itx *ModalInteraction) bool {
			preMdlCount.Add(1)
			return true
		},
		PostModal: func(ctx context.Context, itx *ModalInteraction) {
			postMdlCount.Add(1)
		},
	})

	err := client.RegisterCommand(Command{
		Command: discord.Command{
			Name:        "greet",
			Description: "Greet someone",
			Options: []discord.CommandOption{
				{Name: "name", Description: "Who", Type: discord.StringOptionType, AutoComplete: true},
			},
		},
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			name, _ := itx.GetOptionValue("name")
			return itx.SendLinearReply(ctx, "Hi "+name.(string), true)
		},
		AutoCompleteHandler: func(ctx context.Context, itx *CommandInteraction) ([]discord.CommandOptionChoice, error) {
			_, val := itx.GetFocusedValue()
			return []discord.CommandOptionChoice{
				{Name: "Alice", Value: val.(string) + "alice"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterCommand: %v", err)
	}

	if err := client.RegisterComponent("btn:ok", func(ctx context.Context, itx *ComponentInteraction) error {
		return itx.AcknowledgeWithLinearUpdate(ctx, "updated")
	}); err != nil {
		t.Fatalf("RegisterComponent exact: %v", err)
	}

	if err := client.RegisterComponent("ban/{user_id}/{reason}", func(ctx context.Context, itx *ComponentInteraction) error {
		uid := itx.PathValue("user_id")
		reason, ok := itx.Param("reason")
		if !ok {
			return errors.New("missing reason")
		}
		return itx.AcknowledgeWithLinearMessage(ctx, "banned:"+uid+":"+reason, true)
	}); err != nil {
		t.Fatalf("RegisterComponent pattern: %v", err)
	}

	if err := client.RegisterModal("mdl:{form_id}", func(ctx context.Context, itx *ModalInteraction) error {
		formID := itx.PathValue("form_id")
		val := itx.GetInputValue("field:1")
		return itx.AcknowledgeWithLinearMessage(ctx, formID+":"+val, true)
	}); err != nil {
		t.Fatalf("RegisterModal pattern: %v", err)
	}

	t.Run("Ping", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := signedInteractionRequest(t, priv, `{"type":1}`)
		client.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), bodyPingResponse) {
			t.Fatalf("ping response = (%d, %s), want (200, %s)", rec.Code, rec.Body.String(), bodyPingResponse)
		}
	})

	t.Run("SlashCommand", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":2,"token":"tok","app_permissions":"0","data":{"id":"2","name":"greet","type":1,"options":[{"name":"name","type":3,"value":"World"}]}}`
		client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

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
		client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

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

	t.Run("ComponentExact", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":3,"token":"tok","app_permissions":"0","data":{"custom_id":"btn:ok","component_type":2}}`
		client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

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

	t.Run("ComponentPattern", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":3,"token":"tok","app_permissions":"0","data":{"custom_id":"ban/42/spam","component_type":2}}`
		client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var resp discord.ResponseMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal pattern component response: %v", err)
		}
		if resp.Data == nil || resp.Data.Content != "banned:42:spam" {
			t.Fatalf("unexpected pattern response: %+v", resp)
		}
	})

	t.Run("ModalPatternSubmit", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"id":"1","application_id":"123456789012345678","type":5,"token":"tok","app_permissions":"0","data":{"custom_id":"mdl:profile","components":[{"type":18,"label":"L","component":{"type":4,"style":1,"custom_id":"field:1","value":"abc"}}]}}`
		client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var resp discord.ResponseMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal modal response: %v", err)
		}
		if resp.Type != discord.ChannelMessageWithSourceResponseType || resp.Data == nil || resp.Data.Content != "profile:abc" {
			t.Fatalf("unexpected modal response: %+v", resp)
		}
	})

	if err := client.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if preCmdCount.Load() != 1 || postCmdCount.Load() != 1 {
		t.Fatalf("command hooks = (%d, %d), want (1, 1)", preCmdCount.Load(), postCmdCount.Load())
	}
	if preCmpCount.Load() != 2 || postCmpCount.Load() != 2 {
		t.Fatalf("component hooks = (%d, %d), want (2, 2)", preCmpCount.Load(), postCmpCount.Load())
	}
	if preMdlCount.Load() != 1 || postMdlCount.Load() != 1 {
		t.Fatalf("modal hooks = (%d, %d), want (1, 1)", preMdlCount.Load(), postMdlCount.Load())
	}
}

func TestBuildSyncCommandsAndFindInteractiveComponent(t *testing.T) {
	t.Parallel()

	client, _ := newTestClientPair(t, ClientHooks{})

	if err := client.RegisterCommand(Command{
		Command: discord.Command{Name: "tag", Description: "Root"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterSubCommand(Command{
		Command: discord.Command{Name: "add", Description: "Add"},
	}, "tag"); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterSubCommand(Command{
		Command: discord.Command{Name: "grant", Description: "Grant role"},
	}, "tag", "admin"); err != nil {
		t.Fatal(err)
	}

	parsed := client.buildSyncCommands(nil, false)
	if len(parsed) != 1 || parsed[0].Name != "tag" || len(parsed[0].Options) != 2 {
		t.Fatalf("parsed commands = %+v, want 1 root command with 2 options (1 sub + 1 group)", parsed)
	}
	if parsed[0].Options[0].Type != discord.SubCommandOptionType || parsed[0].Options[0].Name != "add" {
		t.Fatalf("first option = %+v, want SubCommandOptionType add", parsed[0].Options[0])
	}
	if parsed[0].Options[1].Type != discord.SubCommandGroupOptionType || parsed[0].Options[1].Name != "admin" {
		t.Fatalf("second option = %+v, want SubCommandGroupOptionType admin", parsed[0].Options[1])
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
