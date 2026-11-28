package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amatsagu/tempest"
	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"

	"github.com/amatsagu/tempest-example/feature/calc"
	"github.com/amatsagu/tempest-example/feature/commands"
	"github.com/amatsagu/tempest-example/feature/components"
	"github.com/amatsagu/tempest-example/feature/contextmenu"
	"github.com/amatsagu/tempest-example/feature/image"
	"github.com/amatsagu/tempest-example/feature/patterns"
	"github.com/amatsagu/tempest-example/feature/premium"
	"github.com/amatsagu/tempest-example/feature/showcase"
	"github.com/amatsagu/tempest-example/feature/stats"
	"github.com/amatsagu/tempest-example/feature/tags"
)

func signedWebhookReq(t *testing.T, priv ed25519.PrivateKey, body string) *http.Request {
	t.Helper()

	ts := "1700000000"
	sig := ed25519.Sign(priv, append([]byte(ts), body...))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	req.Header.Set("X-Signature-Timestamp", ts)
	return req
}

type testResponseData struct {
	Poll       *discord.Poll        `json:"poll,omitzero"`
	Content    string               `json:"content,omitzero"`
	Components []jsontext.Value     `json:"components,omitzero"`
	Flags      discord.MessageFlags `json:"flags,omitzero"`
}

type testResponse struct {
	Data *testResponseData    `json:"data,omitzero"`
	Type discord.ResponseType `json:"type"`
}

func postSigned(t *testing.T, client *tempest.Client, priv ed25519.PrivateKey, body string) testResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	client.ServeHTTP(rec, signedWebhookReq(t, priv, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s) for body %s", rec.Code, rec.Body.String(), body)
	}

	var resp testResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response (%s): %v", rec.Body.String(), err)
	}
	return resp
}

func TestExampleBotEndToEndAndHooks(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer apiSrv.Close()

	token := base64.RawURLEncoding.EncodeToString([]byte("123456789012345678")) + ".test.sig"
	botStats := stats.NewStats()

	client := tempest.NewClient(tempest.ClientConfig{
		Context:   t.Context(),
		Token:     token,
		PublicKey: hex.EncodeToString(pub),
		Rest: rest.NewRestClient(rest.RestClientConfig{
			Token:   token,
			BaseURL: apiSrv.URL,
		}),
		Hooks: tempest.ClientHooks{
			PreCommand:  botStats.PreCommand,
			PostCommand: botStats.PostCommand,
			CommandError: func(ctx context.Context, cmd tempest.Command, itx *tempest.CommandInteraction, err error) {
				botStats.RecordCommandError(ctx, cmd, itx, err)
				if !itx.Responded() {
					_ = itx.SendLinearReply(ctx, "Command failed: "+err.Error(), true)
				}
			},
			PreComponent:  botStats.PreComponent,
			PostComponent: botStats.PostComponent,
			ComponentError: func(ctx context.Context, itx *tempest.ComponentInteraction, err error) {
				botStats.RecordComponentError(ctx, itx, err)
				if !itx.Responded() {
					_ = itx.AcknowledgeWithLinearMessage(ctx, "Component failed: "+err.Error(), true)
				}
			},
			PreModal:  botStats.PreModal,
			PostModal: botStats.PostModal,
			ModalError: func(ctx context.Context, itx *tempest.ModalInteraction, err error) {
				botStats.RecordModalError(ctx, itx, err)
				if !itx.Responded() {
					_ = itx.AcknowledgeWithLinearMessage(ctx, "Modal failed: "+err.Error(), true)
				}
			},
		},
	})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = client.Shutdown(shutdownCtx)
	}()

	registers := []func(*tempest.Client) ([]tempest.Command, error){
		commands.Register,
		contextmenu.Register,
		components.Register,
		patterns.Register,
		showcase.Register,
		image.Register,
		tags.Register,
		calc.Register,
		premium.Register,
		func(c *tempest.Client) ([]tempest.Command, error) {
			return stats.Register(c, botStats)
		},
	}
	for _, register := range registers {
		if _, err := register(client); err != nil {
			t.Fatalf("feature register failed: %v", err)
		}
	}

	// Failing component route exercises ComponentError hook too (first register & unregister test route).
	if err := client.RegisterComponent("test:err", func(ctx context.Context, itx *tempest.ComponentInteraction) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	client.UnregisterComponent("test:err")
	if err := client.RegisterComponent("test:err", func(ctx context.Context, itx *tempest.ComponentInteraction) error {
		_ = itx.AcknowledgeWithLinearMessage(ctx, "error hook test", true)
		return context.DeadlineExceeded
	}); err != nil {
		t.Fatal(err)
	}

	// 1. /ping
	pingResp := postSigned(t, client, priv, `{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"ping","type":1}}`)
	if pingResp.Data == nil || pingResp.Data.Content != "Pong." {
		t.Fatalf("/ping = %+v, want Pong.", pingResp)
	}

	// 2. /action autocomplete + /action command + pattern buttons + patterned modal.
	recAC := httptest.NewRecorder()
	client.ServeHTTP(recAC, signedWebhookReq(t, priv, `{"id":"2","application_id":"123456789012345678","type":4,"token":"t","app_permissions":"0","data":{"id":"3","name":"action","type":1,"options":[{"name":"note","type":3,"value":"spam","focused":true}]}}`))
	if recAC.Code != http.StatusOK || !strings.Contains(recAC.Body.String(), "Spamming links") {
		t.Fatalf("/action autocomplete = (%d, %s)", recAC.Code, recAC.Body.String())
	}

	actionResp := postSigned(t, client, priv, `{"id":"3","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"3","name":"action","type":1,"options":[{"name":"target","type":6,"value":"42"},{"name":"note","type":3,"value":"guild:9:case:12"}]}}`)
	if actionResp.Data == nil || len(actionResp.Data.Components) == 0 {
		t.Fatalf("/action response missing components: %+v", actionResp)
	}

	// Exact route "act:help"
	helpResp := postSigned(t, client, priv, `{"id":"4","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"act:help","component_type":2}}`)
	if helpResp.Data == nil || !strings.Contains(helpResp.Data.Content, "act/{user_id}/mute/{duration}") {
		t.Fatalf("act:help = %+v", helpResp)
	}

	// Multi-wildcard precedence: "act/42/mute/10m" must hit handleMute before "act/{user_id}/{verb}"
	muteResp := postSigned(t, client, priv, `{"id":"5","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"act/42/mute/10m","component_type":2}}`)
	if muteResp.Type != discord.UpdateMessageResponseType || muteResp.Data == nil || len(muteResp.Data.Components) == 0 {
		t.Fatalf("mute pattern = %+v", muteResp)
	}

	// Single wildcard "act/42/warn" opens patterned modal "act:modal:42:warn"
	recModalOpen := httptest.NewRecorder()
	client.ServeHTTP(recModalOpen, signedWebhookReq(t, priv, `{"id":"6","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"act/42/warn","component_type":2}}`))
	var modalOpenResp struct {
		Data *struct {
			CustomID string `json:"custom_id"`
			Title    string `json:"title"`
		} `json:"data"`
		Type discord.ResponseType `json:"type"`
	}
	if err := json.Unmarshal(recModalOpen.Body.Bytes(), &modalOpenResp); err != nil {
		t.Fatal(err)
	}
	if modalOpenResp.Type != discord.ModalResponseType || modalOpenResp.Data == nil || modalOpenResp.Data.CustomID != "act:modal:42:warn" {
		t.Fatalf("expected patterned modal act:modal:42:warn, got %+v", modalOpenResp)
	}

	// Modal submit hits "act:modal:{user_id}:{kind}"
	modalSubmitResp := postSigned(t, client, priv, `{"id":"7","application_id":"123456789012345678","type":5,"token":"t","app_permissions":"0","data":{"custom_id":"act:modal:42:warn","components":[{"type":18,"label":"Reason","component":{"type":4,"style":1,"custom_id":"act:modal:reason","value":"repeated flood"}}]}}`)
	if modalSubmitResp.Data == nil || !strings.Contains(modalSubmitResp.Data.Content, `kind="warn" user=<@42> reason="repeated flood"`) {
		t.Fatalf("modal submit = %+v", modalSubmitResp)
	}

	// Catch-all {trail...}
	trailResp := postSigned(t, client, priv, `{"id":"8","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"act:note:guild:9:case:12","component_type":2}}`)
	if trailResp.Data == nil || !strings.Contains(trailResp.Data.Content, `guild:9:case:12`) {
		t.Fatalf("catch-all note = %+v", trailResp)
	}

	// 3. /poll + poll/{owner_id}/close
	pollResp := postSigned(t, client, priv, `{"id":"9","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","user":{"id":"99","username":"bob"},"data":{"id":"4","name":"poll","type":1,"options":[{"name":"question","type":3,"value":"Best lang?"},{"name":"options","type":3,"value":"Go, Rust"}]}}`)
	if pollResp.Data == nil || pollResp.Data.Poll == nil || len(pollResp.Data.Poll.Answers) != 2 {
		t.Fatalf("/poll = %+v", pollResp)
	}

	pollClose := postSigned(t, client, priv, `{"id":"10","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","user":{"id":"99","username":"bob"},"data":{"custom_id":"poll/99/close","component_type":2}}`)
	if pollClose.Type != discord.UpdateMessageResponseType || pollClose.Data == nil || !strings.Contains(pollClose.Data.Content, "closed by owner") {
		t.Fatalf("poll close = %+v", pollClose)
	}

	// 4. /calc + pattern-matched key press "calc:{session}:{key}"
	calcResp := postSigned(t, client, priv, `{"id":"11","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"5","name":"calc","type":1}}`)
	calcCmps := make([]discord.MessageComponent, 0, len(calcResp.Data.Components))
	for _, rawCmp := range calcResp.Data.Components {
		cmp, err := discord.UnmarshalComponent(rawCmp)
		if err != nil {
			t.Fatalf("unmarshal calc component: %v", err)
		}
		if mc, ok := cmp.(discord.MessageComponent); ok {
			calcCmps = append(calcCmps, mc)
		}
	}
	btn, found := tempest.FindInteractiveComponent(calcCmps, func(b discord.ButtonComponent) bool {
		return strings.HasPrefix(b.CustomID, "calc:") && strings.HasSuffix(b.CustomID, ":7")
	})
	if !found {
		t.Fatal("expected calc button :7")
	}
	calcPress := postSigned(t, client, priv, `{"id":"12","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"`+btn.CustomID+`","component_type":2}}`)
	if calcPress.Type != discord.UpdateMessageResponseType || calcPress.Data == nil || calcPress.Data.Content != "7" {
		t.Fatalf("calc press = %+v, want display 7", calcPress)
	}

	// 5. /tag add + /tag show (missing triggers CommandError) + /tag admin clear (two-level group)
	_ = postSigned(t, client, priv, `{"id":"13","application_id":"123456789012345678","type":2,"token":"t","guild_id":"77","app_permissions":"0","data":{"id":"6","name":"tag","type":1,"options":[{"name":"add","type":1,"options":[{"name":"name","type":3,"value":"k"},{"name":"value","type":3,"value":"v"}]}]}}`)
	showErr := postSigned(t, client, priv, `{"id":"14","application_id":"123456789012345678","type":2,"token":"t","guild_id":"77","app_permissions":"0","data":{"id":"6","name":"tag","type":1,"options":[{"name":"show","type":1,"options":[{"name":"name","type":3,"value":"missing"}]}]}}`)
	if showErr.Data == nil || !strings.Contains(showErr.Data.Content, "tag") {
		t.Fatalf("expected error reply for missing tag: %+v", showErr)
	}
	clearResp := postSigned(t, client, priv, `{"id":"15","application_id":"123456789012345678","type":2,"token":"t","guild_id":"77","app_permissions":"0","data":{"id":"6","name":"tag","type":1,"options":[{"name":"admin","type":2,"options":[{"name":"clear","type":1}]}]}}`)
	if clearResp.Data == nil || !strings.Contains(clearResp.Data.Content, "Cleared all tags") {
		t.Fatalf("/tag admin clear = %+v", clearResp)
	}

	// 6. Runtime RegisterComponent + UnregisterComponent via /ticket
	ticketResp := postSigned(t, client, priv, `{"id":"1600","application_id":"123456789012345678","type":2,"token":"t","user":{"id":"99","username":"bob"},"app_permissions":"0","data":{"id":"7","name":"ticket","type":1}}`)
	if ticketResp.Data == nil || len(ticketResp.Data.Components) == 0 {
		t.Fatalf("/ticket = %+v", ticketResp)
	}
	ticketClick := postSigned(t, client, priv, `{"id":"17","application_id":"123456789012345678","type":3,"token":"t","user":{"id":"99","username":"bob"},"app_permissions":"0","data":{"custom_id":"ticket:1600","component_type":2}}`)
	if ticketClick.Type != discord.UpdateMessageResponseType || ticketClick.Data == nil || ticketClick.Data.Content != "Ticket created." {
		t.Fatalf("ticket click = %+v", ticketClick)
	}

	// Trigger test:err component hook too.
	_ = postSigned(t, client, priv, `{"id":"18","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"test:err","component_type":2}}`)

	// 7. /stats and verify all ClientHooks updated counters.
	statsResp := postSigned(t, client, priv, `{"id":"19","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"8","name":"stats","type":1}}`)
	if statsResp.Data == nil || len(statsResp.Data.Components) == 0 {
		t.Fatalf("/stats = %+v", statsResp)
	}

	snap := botStats.Snapshot()
	if snap.CommandsStarted < 9 || snap.CommandsCompleted < 9 {
		t.Fatalf("command hook counts too low: %+v", snap)
	}
	if snap.ComponentsStarted < 7 || snap.ComponentsCompleted < 7 {
		t.Fatalf("component hook counts too low: %+v", snap)
	}
	if snap.ModalsStarted != 1 || snap.ModalsCompleted != 1 {
		t.Fatalf("modal hook counts = (%d, %d), want (1, 1)", snap.ModalsStarted, snap.ModalsCompleted)
	}
	if snap.Errors < 1 {
		t.Fatalf("expected hook errors >= 1, got %d", snap.Errors)
	}
}
