package tempest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"
)

func TestNewClientConfigAndPanics(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubHex := hex.EncodeToString(pub)

	r := NewRest(RestClientConfig{Token: testBotToken("111")})
	if r == nil {
		t.Fatal("NewRest returned nil")
	}

	var logBuf bytes.Buffer
	client := NewClient(ClientConfig{
		Context:                    t.Context(),
		Token:                      testBotToken("111"),
		PublicKey:                  pubHex,
		TraceLogger:                log.New(&logBuf, "[TEST] ", 0),
		Trace:                      true,
		DefaultInteractionContexts: []discord.InteractionContextType{discord.BotDMContextType},
		MaxConcurrentHandlers:      -1,
		Rest:                       r,
	})
	t.Cleanup(func() {
		_ = client.Shutdown(t.Context())
	})

	if client.Rest != r {
		t.Fatal("expected pre-built Rest client to be reused")
	}
	if !client.trace {
		t.Fatal("expected trace enabled")
	}

	_ = NewClient(ClientConfig{
		Token:     testBotToken("111"),
		PublicKey: pubHex,
		Trace:     true,
	})

	assertPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected %s to panic", name)
			}
		}()
		fn()
	}

	assertPanic("bad token", func() {
		NewClient(ClientConfig{Token: "bad", PublicKey: pubHex})
	})
	assertPanic("bad public key hex", func() {
		NewClient(ClientConfig{Token: testBotToken("111"), PublicKey: "zz"})
	})
	assertPanic("wrong public key length", func() {
		NewClient(ClientConfig{Token: testBotToken("111"), PublicKey: "abcd"})
	})
}

func TestClientRESTHelpersAndCommandSync(t *testing.T) {
	t.Parallel()

	type seenReq struct {
		method string
		path   string
		query  string
	}
	var seen []seenReq

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		seen = append(seen, seenReq{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery})

		w.Header().Set("Content-Type", ContentTypeJSON)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/users/@me/channels":
			_, _ = w.Write([]byte(`{"id":"555"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/users/42":
			_, _ = w.Write([]byte(`{"id":"42","username":"alice","global_name":"Alice"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/guilds/10/members/42":
			_, _ = w.Write([]byte(`{"user":{"id":"42","username":"alice","global_name":"Alice"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/applications/123456789012345678/entitlements":
			_, _ = w.Write([]byte(`[{"id":"900"}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/applications/123456789012345678/entitlements/900":
			_, _ = w.Write([]byte(`{"id":"900"}`))
		case strings.HasPrefix(r.URL.Path, "/channels/"):
			_, _ = w.Write([]byte(`{"id":"777","channel_id":"555"}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	client, _ := newTestClientPair(t, ClientHooks{})
	client.Rest = rest.NewRestClient(rest.RestClientConfig{
		Token:   testBotToken("123456789012345678"),
		BaseURL: srv.URL,
	})

	ctx := t.Context()
	chID := discord.ChannelID{Snowflake: 100}
	msgID := discord.MessageID{Snowflake: 200}
	userID := discord.UserID{Snowflake: 42}
	guildID := discord.GuildID{Snowflake: 10}
	entID := discord.EntitlementID{Snowflake: 900}

	if _, err := client.SendLinearMessage(ctx, chID, "hello"); err != nil {
		t.Fatalf("SendLinearMessage: %v", err)
	}
	if msg, err := client.SendPrivateMessage(ctx, userID, discord.Message{Content: "dm"}, nil); err != nil || msg.ChannelID.Snowflake != 555 {
		t.Fatalf("SendPrivateMessage = (%+v, %v)", msg, err)
	}
	if err := client.EditMessage(ctx, chID, msgID, discord.Message{Content: "edited"}); err != nil {
		t.Fatalf("EditMessage: %v", err)
	}
	if err := client.DeleteMessage(ctx, chID, msgID); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
	if err := client.CrosspostMessage(ctx, chID, msgID); err != nil {
		t.Fatalf("CrosspostMessage: %v", err)
	}
	if u, err := client.FetchUser(ctx, userID); err != nil || u.ID != userID {
		t.Fatalf("FetchUser = (%+v, %v)", u, err)
	}
	if m, err := client.FetchMember(ctx, guildID, userID); err != nil || m.User == nil || m.User.ID != userID {
		t.Fatalf("FetchMember = (%+v, %v)", m, err)
	}
	if ents, err := client.FetchEntitlementsPage(ctx, "user_id=42"); err != nil || len(ents) != 1 {
		t.Fatalf("FetchEntitlementsPage = (%+v, %v)", ents, err)
	}
	if _, err := client.FetchEntitlement(ctx, entID); err != nil {
		t.Fatalf("FetchEntitlement: %v", err)
	}
	if err := client.ConsumeEntitlement(ctx, entID); err != nil {
		t.Fatalf("ConsumeEntitlement: %v", err)
	}
	if err := client.CreateTestEntitlement(ctx, discord.TestEntitlementPayload{}); err != nil {
		t.Fatalf("CreateTestEntitlement: %v", err)
	}
	if err := client.DeleteTestEntitlement(ctx, entID); err != nil {
		t.Fatalf("DeleteTestEntitlement: %v", err)
	}

	if err := client.RegisterCommand(Command{Name: "alpha", Description: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterCommand(Command{Name: "beta", Description: "B"}); err != nil {
		t.Fatal(err)
	}

	if err := client.SyncCommands(ctx, nil, []string{"alpha"}, false); err != nil {
		t.Fatalf("SyncCommands whitelist: %v", err)
	}
	if err := client.SyncCommands(ctx, []discord.GuildID{guildID}, []string{"beta"}, true); err != nil {
		t.Fatalf("SyncCommands guild exclude: %v", err)
	}

	if len(seen) < 14 {
		t.Fatalf("expected >= 14 REST calls, got %d", len(seen))
	}
}

func TestRegistryEdgeCasesAndSubcommandGroups(t *testing.T) {
	t.Parallel()

	client, _ := newTestClientPair(t, ClientHooks{})

	if err := client.RegisterCommand(Command{}); err == nil {
		t.Fatal("expected empty command name error")
	}
	if err := client.RegisterSubCommand(Command{Name: "sub", Description: "d"}, "missing"); err == nil {
		t.Fatal("expected missing parent error")
	}
	if err := client.RegisterSubCommand(Command{Name: "sub", Description: "d"}); err == nil {
		t.Fatal("expected empty parentPath error")
	}
	if err := client.RegisterSubCommand(Command{Name: "", Description: "d"}, "p"); err == nil {
		t.Fatal("expected empty sub name error")
	}
	if err := client.RegisterSubCommand(Command{Name: "s", Description: "d"}, "a@b"); err == nil {
		t.Fatal("expected invalid @ in parentPath error")
	}
	if err := client.RegisterComponent("btn", nil); err == nil {
		t.Fatal("expected nil component handler error")
	}
	if err := client.RegisterModal("mdl", nil); err == nil {
		t.Fatal("expected nil modal handler error")
	}

	if err := client.RegisterCommand(Command{Name: "cfg", Description: "Config"}); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterCommand(Command{Name: "cfg", Description: "Dup"}); err == nil {
		t.Fatal("expected duplicate command error")
	}
	if err := client.RegisterSubCommand(Command{
		Name: "set", Description: "Set",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			return nil
		},
	}, "cfg", "role"); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterSubCommand(Command{Name: "set", Description: "Dup"}, "cfg", "role"); err == nil {
		t.Fatal("expected duplicate subcommand error")
	}

	if _, ok := client.FindCommand("cfg@role@set"); !ok {
		t.Fatal("expected FindCommand to locate grouped subcommand")
	}

	var cmdKeys []string
	for k := range client.RegisteredCommands() {
		cmdKeys = append(cmdKeys, k)
	}
	if !slices.Contains(cmdKeys, "cfg@role@set") {
		t.Fatalf("RegisteredCommands missing cfg@role@set: %v", cmdKeys)
	}

	if !client.UnregisterCommand("cfg@role@set") || client.UnregisterCommand("cfg@role@set") {
		t.Fatal("unexpected UnregisterCommand results")
	}

	_ = client.RegisterComponent("c1", func(ctx context.Context, itx *ComponentInteraction) error { return nil })
	_ = client.RegisterModal("m1", func(ctx context.Context, itx *ModalInteraction) error { return nil })

	var cmps, mdls []string
	for k := range client.RegisteredComponents() {
		cmps = append(cmps, k)
	}
	for k := range client.RegisteredModals() {
		mdls = append(mdls, k)
	}
	if !slices.Equal(cmps, []string{"c1"}) || !slices.Equal(mdls, []string{"m1"}) {
		t.Fatalf("unexpected registered sets: cmps=%v mdls=%v", cmps, mdls)
	}

	if !client.UnregisterComponent("c1") || !client.UnregisterModal("m1") {
		t.Fatal("expected UnregisterComponent/Modal to return true")
	}
	client.ClearComponents()
	client.ClearModals()
}

func TestInteractionDispatchEdgeCasesAndHelpers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", ContentTypeJSON)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/gw-tok") {
			_, _ = w.Write([]byte(`{"id":"888"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	var skipped atomic.Bool
	client, priv := newTestClientPair(t, ClientHooks{
		PreCommand: func(ctx context.Context, cmd Command, itx *CommandInteraction) bool {
			if cmd.Name == "skip" {
				skipped.Store(true)
				return false
			}
			return true
		},
		PreComponent: func(ctx context.Context, itx *ComponentInteraction) bool {
			return itx.Data.CustomID != "cmp-skip"
		},
		PreModal: func(ctx context.Context, itx *ModalInteraction) bool {
			return itx.Data.CustomID != "mdl-skip"
		},
	})
	client.Rest = rest.NewRestClient(rest.RestClientConfig{
		Token:   testBotToken("123456789012345678"),
		BaseURL: srv.URL,
	})

	_ = client.RegisterCommand(Command{
		Name: "skip", Description: "Skip",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			t.Error("skipped command ran unexpectedly")
			return nil
		},
	})
	_ = client.RegisterCommand(Command{
		Name: "admin", Description: "Admin",
	})
	_ = client.RegisterSubCommand(Command{
		Name: "ban", Description: "Ban",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			if err := itx.SendReply(ctx, discord.ResponseMessageData{Content: "file"}, true, []File{{Name: "a.txt", Reader: strings.NewReader("hi")}}); err != nil {
				return err
			}
			if !itx.Responded() {
				return errors.New("expected Responded() == true")
			}
			if !errors.Is(itx.SendModal(ctx, discord.ResponseModalData{CustomID: "m", Title: "t"}), ErrAlreadyDeferred) {
				return errors.New("expected ErrAlreadyDeferred on SendModal after reply")
			}
			if err := itx.EditLinearReply(ctx, "edited", true); err != nil {
				return err
			}
			if _, err := itx.SendLinearFollowUp(ctx, "f1", true); err != nil {
				return err
			}
			if msg, err := itx.SendLinearFollowUpWithFiles(ctx, "f2", true, []File{{Name: "b.txt", Reader: strings.NewReader("x")}}); err != nil || msg.ID.Snowflake != 888 {
				return errors.New("followup with files failed")
			}
			if err := itx.EditLinearFollowUp(ctx, discord.MessageID{Snowflake: 888}, "f3"); err != nil {
				return err
			}
			if err := itx.DeleteFollowUp(ctx, discord.MessageID{Snowflake: 888}); err != nil {
				return err
			}
			return itx.DeleteReply(ctx)
		},
		AutoCompleteHandler: func(ctx context.Context, itx *CommandInteraction) ([]discord.CommandOptionChoice, error) {
			return nil, errors.New("ac fail")
		},
	}, "admin", "mod")

	_ = client.RegisterComponent("cmp-skip", func(ctx context.Context, itx *ComponentInteraction) error {
		t.Error("skipped component ran unexpectedly")
		return nil
	})
	_ = client.RegisterComponent("cmp-ops", func(ctx context.Context, itx *ComponentInteraction) error {
		if err := itx.Defer(ctx, true); err != nil {
			return err
		}
		if err := itx.EditLinearOriginal(ctx, "orig"); err != nil {
			return err
		}
		if _, err := itx.SendLinearFollowUp(ctx, "cf", true); err != nil {
			return err
		}
		if _, err := itx.SendFollowUpWithFiles(ctx, discord.ResponseMessageData{Content: "cf2"}, false, []File{{Name: "f.txt", Reader: strings.NewReader("1")}}); err != nil {
			return err
		}
		if err := itx.EditFollowUp(ctx, discord.MessageID{Snowflake: 888}, discord.ResponseMessageData{Content: "ce"}); err != nil {
			return err
		}
		if err := itx.DeleteFollowUp(ctx, discord.MessageID{Snowflake: 888}); err != nil {
			return err
		}
		return itx.DeleteOriginal(ctx)
	})
	_ = client.RegisterModal("mdl-skip", func(ctx context.Context, itx *ModalInteraction) error {
		t.Error("skipped modal ran unexpectedly")
		return nil
	})
	_ = client.RegisterModal("mdl-ops", func(ctx context.Context, itx *ModalInteraction) error {
		if err := itx.Defer(ctx, true); err != nil {
			return err
		}
		if err := itx.EditLinearOriginal(ctx, "m-orig"); err != nil {
			return err
		}
		if _, err := itx.SendLinearFollowUp(ctx, "mf", true); err != nil {
			return err
		}
		if _, err := itx.SendFollowUpWithFiles(ctx, discord.ResponseMessageData{Content: "mf2"}, false, []File{{Name: "m.txt", Reader: strings.NewReader("2")}}); err != nil {
			return err
		}
		if err := itx.EditFollowUp(ctx, discord.MessageID{Snowflake: 888}, discord.ResponseMessageData{Content: "me"}); err != nil {
			return err
		}
		if err := itx.DeleteFollowUp(ctx, discord.MessageID{Snowflake: 888}); err != nil {
			return err
		}
		return itx.DeleteOriginal(ctx)
	})

	// SubCommandGroup over HTTP with files (auto-defer + EditReply + FollowUps).
	rec := httptest.NewRecorder()
	groupCmd := `{"id":"1","application_id":"123456789012345678","type":2,"token":"gw-tok","app_permissions":"0","member":{"user":{"id":"5","username":"u"}},"guild_id":"9","data":{"id":"2","name":"admin","type":1,"options":[{"name":"mod","type":2,"options":[{"name":"ban","type":1}]}]}}`
	client.ServeHTTP(rec, signedInteractionRequest(t, priv, groupCmd))
	if rec.Code != http.StatusOK {
		t.Fatalf("group command HTTP status = %d", rec.Code)
	}

	// Unknown command HTTP & Gateway, plus skipped/unsupported payloads.
	recUnknown := httptest.NewRecorder()
	client.ServeHTTP(recUnknown, signedInteractionRequest(t, priv, `{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"nope","type":1}}`))
	if !bytes.Equal(recUnknown.Body.Bytes(), bodyUnknownCommandResponse) {
		t.Fatalf("expected unknown command response, got %s", recUnknown.Body.String())
	}

	for _, badBody := range []string{`not json`, `{}`, `{"type":"bad"}`, `{"type":99}`, `{"type":2,"data":123}`, `{"type":3,"data":123}`, `{"type":4,"data":123}`, `{"type":5,"data":123}`} {
		recBad := httptest.NewRecorder()
		client.ServeHTTP(recBad, signedInteractionRequest(t, priv, badBody))
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want 400", badBody, recBad.Code)
		}
	}

	if err := client.HandleInteraction(t.Context(), []byte(`{"type":1}`)); err != nil {
		t.Fatalf("HandleInteraction(Ping) = %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"type":99}`)); err == nil {
		t.Fatal("expected unsupported type error on HandleInteraction")
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":2,"token":"gw-tok","app_permissions":"0","data":{"id":"2","name":"nope","type":1}}`)); !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":3,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"missing","component_type":2}}`)); !errors.Is(err, ErrUnknownComponent) {
		t.Fatalf("expected ErrUnknownComponent, got %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":5,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"missing","components":[]}}`)); !errors.Is(err, ErrUnknownModal) {
		t.Fatalf("expected ErrUnknownModal, got %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":2,"token":"gw-tok","app_permissions":"0","data":{"id":"2","name":"skip","type":1}}`)); err != nil || !skipped.Load() {
		t.Fatalf("skip command failed: err=%v skipped=%t", err, skipped.Load())
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":3,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"cmp-skip","component_type":2}}`)); err != nil {
		t.Fatalf("cmp-skip: %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":5,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"mdl-skip","components":[]}}`)); err != nil {
		t.Fatalf("mdl-skip: %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":3,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"cmp-ops","component_type":2}}`)); err != nil {
		t.Fatalf("cmp-ops: %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":5,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"mdl-ops","components":[]}}`)); err != nil {
		t.Fatalf("mdl-ops: %v", err)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":4,"token":"gw-tok","app_permissions":"0","data":{"id":"2","name":"admin","type":1,"options":[{"name":"mod","type":2,"options":[{"name":"ban","type":1}]}]}}`)); err == nil {
		t.Fatal("expected autocomplete error")
	}

	// Verify ServeHTTP AckTimeout and clean ErrAckTimeout when handler responds too late.
	pub2, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	slowClient := NewClient(ClientConfig{
		Token:                 testBotToken("123456789012345678"),
		PublicKey:             hex.EncodeToString(pub2),
		AckTimeout:            20 * time.Millisecond,
		MaxConcurrentHandlers: 1,
	})
	defer func() { _ = slowClient.Shutdown(t.Context()) }()

	lateErr := make(chan error, 1)
	_ = slowClient.RegisterCommand(Command{
		Name: "slow", Description: "Slow",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			time.Sleep(60 * time.Millisecond)
			lateErr <- itx.SendLinearReply(ctx, "too late", true)
			return nil
		},
	})

	recSlow := httptest.NewRecorder()
	slowClient.ServeHTTP(recSlow, signedInteractionRequest(t, priv2, `{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"slow","type":1}}`))
	if recSlow.Code != http.StatusNoContent {
		t.Fatalf("slow status = %d, want 204", recSlow.Code)
	}
	if got := <-lateErr; !errors.Is(got, ErrAckTimeout) {
		t.Fatalf("late SendLinearReply = %v, want ErrAckTimeout", got)
	}
}

func TestConcurrencyBackpressureAndRemainingResponses(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(ClientConfig{
		Token:                 testBotToken("123456789012345678"),
		PublicKey:             hex.EncodeToString(pub),
		AckTimeout:            400 * time.Millisecond,
		MaxConcurrentHandlers: 1,
	})
	defer func() { _ = client.Shutdown(t.Context()) }()

	hold := make(chan struct{})
	started := make(chan struct{})
	_ = client.RegisterCommand(Command{
		Name: "hold", Description: "Hold",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			close(started)
			<-hold
			return itx.SendModal(ctx, discord.ResponseModalData{CustomID: "m:1", Title: "Title"})
		},
	})
	_ = client.RegisterComponent("open-mdl:{id}", func(ctx context.Context, itx *ComponentInteraction) error {
		if _, ok := itx.Param("id"); !ok {
			return errors.New("missing id param")
		}
		if err := itx.AcknowledgeWithModal(ctx, discord.ResponseModalData{CustomID: "m:2", Title: "T"}); err != nil {
			return err
		}
		if !errors.Is(itx.Acknowledge(ctx), ErrAlreadyDeferred) {
			return errors.New("expected ErrAlreadyDeferred")
		}
		return nil
	})
	_ = client.RegisterModal("upd:{id}", func(ctx context.Context, itx *ModalInteraction) error {
		id, ok := itx.Param("id")
		if !ok {
			return errors.New("missing id param")
		}
		if err := itx.AcknowledgeWithLinearUpdate(ctx, "upd:"+id); err != nil {
			return err
		}
		if !errors.Is(itx.Acknowledge(ctx), ErrAlreadyDeferred) {
			return errors.New("expected ErrAlreadyDeferred")
		}
		return nil
	})

	type reqKey struct{}
	baseReq := signedInteractionRequest(t, priv, `{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"hold","type":1}}`)
	baseReq = baseReq.WithContext(context.WithValue(baseReq.Context(), reqKey{}, "val"))

	rec1 := httptest.NewRecorder()
	done1 := make(chan struct{})
	go func() {
		client.ServeHTTP(rec1, baseReq)
		close(done1)
	}()

	<-started

	// Second concurrent HTTP request exceeds semaphoreWaitBudget and gets 503.
	recBusy := httptest.NewRecorder()
	client.ServeHTTP(recBusy, signedInteractionRequest(t, priv, `{"id":"2","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"hold","type":1}}`))
	if recBusy.Code != http.StatusServiceUnavailable {
		t.Fatalf("busy status = %d, want 503", recBusy.Code)
	}

	// Peer cancellation prior to initial ack cancels pre-ack HTTP waiter cleanly.
	peerCtx, peerCancel := context.WithCancel(t.Context())
	peerCancel()
	if err := client.HandleInteraction(peerCtx, []byte(`{"id":"3","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"hold","type":1}}`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled HandleInteraction = %v, want context.Canceled", err)
	}

	close(hold)
	<-done1
	if rec1.Code != http.StatusOK {
		t.Fatalf("rec1 status = %d, want 200", rec1.Code)
	}

	recCmp := httptest.NewRecorder()
	client.ServeHTTP(recCmp, signedInteractionRequest(t, priv, `{"id":"4","application_id":"123456789012345678","type":3,"token":"t","app_permissions":"0","data":{"custom_id":"open-mdl:7","component_type":2}}`))
	if recCmp.Code != http.StatusOK {
		t.Fatalf("recCmp status = %d, want 200", recCmp.Code)
	}

	recMdl := httptest.NewRecorder()
	client.ServeHTTP(recMdl, signedInteractionRequest(t, priv, `{"id":"5","application_id":"123456789012345678","type":5,"token":"t","app_permissions":"0","data":{"custom_id":"upd:9","components":[]}}`))
	if recMdl.Code != http.StatusOK || !strings.Contains(recMdl.Body.String(), "upd:9") {
		t.Fatalf("recMdl = (%d, %s), want upd:9", recMdl.Code, recMdl.Body.String())
	}

	// Verify FindInteractiveComponent Section and Label traversal.
	components := []discord.MessageComponent{
		discord.SectionComponent{
			Type:      discord.SectionComponentType,
			Accessory: discord.ButtonComponent{Type: discord.ButtonComponentType, CustomID: "sec-btn"},
		},
	}
	if sec, ok := FindInteractiveComponent(components, func(b discord.ButtonComponent) bool { return b.CustomID == "sec-btn" }); !ok || sec.CustomID != "sec-btn" {
		t.Fatalf("expected sec-btn, got (%+v, %t)", sec, ok)
	}
	modalComponents := []discord.ModalComponent{
		discord.LabelComponent{
			Type:      discord.LabelComponentType,
			Component: discord.TextInputComponent{Type: discord.TextInputComponentType, CustomID: "lbl-in"},
		},
	}
	if ti, ok := FindInteractiveComponent(modalComponents, func(in discord.TextInputComponent) bool { return in.CustomID == "lbl-in" }); !ok || ti.CustomID != "lbl-in" {
		t.Fatalf("expected lbl-in, got (%+v, %t)", ti, ok)
	}
}
