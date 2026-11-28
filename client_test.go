package tempest

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"
)

type failingWriter struct{}

func (failingWriter) Header() http.Header { return http.Header{} }
func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
func (failingWriter) WriteHeader(int) {}

type failingCloseBody struct {
	io.Reader
}

func (failingCloseBody) Close() error { return errors.New("close failed") }

func TestClientGatewayHandleInteractionAndHooks(t *testing.T) {
	t.Parallel()

	var (
		cmdErrSeen   atomic.Pointer[error]
		cmpErrSeen   atomic.Pointer[error]
		modalErrSeen atomic.Pointer[error]
		callbackHits atomic.Int32
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		callbackHits.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, _ := newTestClientPair(t, ClientHooks{
		CommandError: func(ctx context.Context, cmd Command, itx *CommandInteraction, err error) {
			cmdErrSeen.Store(&err)
		},
		ComponentError: func(ctx context.Context, itx *ComponentInteraction, err error) {
			cmpErrSeen.Store(&err)
		},
		ModalError: func(ctx context.Context, itx *ModalInteraction, err error) {
			modalErrSeen.Store(&err)
		},
	})
	client.Rest = rest.NewRestClient(rest.RestClientConfig{
		Token:   testBotToken("123456789012345678"),
		BaseURL: srv.URL,
	})

	if err := client.RegisterCommand(Command{
		Name: "gw", Description: "GW cmd",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			if err := itx.SendLinearReply(ctx, "ok", true); err != nil {
				return err
			}
			return errors.New("cmd boom")
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := client.RegisterComponent("cmp:{id}", func(ctx context.Context, itx *ComponentInteraction) error {
		if itx.PathValue("id") == "fail" {
			return errors.New("cmp boom")
		}
		return itx.AcknowledgeWithLinearUpdate(ctx, "cmp ok")
	}); err != nil {
		t.Fatal(err)
	}

	if err := client.RegisterModal("mdl:{id}", func(ctx context.Context, itx *ModalInteraction) error {
		panic("modal panic")
	}); err != nil {
		t.Fatal(err)
	}

	// Gateway interaction replies go through Client.Rest (/interactions/{id}/{token}/callback).
	cmdPayload := `{"id":"100","application_id":"123456789012345678","type":2,"token":"gw-tok","app_permissions":"0","data":{"id":"2","name":"gw","type":1}}`
	if err := client.HandleInteraction(t.Context(), []byte(cmdPayload)); err == nil || !strings.Contains(err.Error(), "cmd boom") {
		t.Fatalf("HandleInteraction(cmd) = %v, want cmd boom", err)
	}
	if cmdErrSeen.Load() == nil {
		t.Fatal("expected CommandError hook to fire")
	}

	cmpPayload := `{"id":"101","application_id":"123456789012345678","type":3,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"cmp:fail","component_type":2}}`
	if err := client.HandleInteraction(t.Context(), []byte(cmpPayload)); err == nil || !strings.Contains(err.Error(), "cmp boom") {
		t.Fatalf("HandleInteraction(cmp) = %v, want cmp boom", err)
	}
	if cmpErrSeen.Load() == nil {
		t.Fatal("expected ComponentError hook to fire")
	}

	mdlPayload := `{"id":"102","application_id":"123456789012345678","type":5,"token":"gw-tok","app_permissions":"0","data":{"custom_id":"mdl:99","components":[]}}`
	if err := client.HandleInteraction(t.Context(), []byte(mdlPayload)); err == nil || !strings.Contains(err.Error(), "handler panic") {
		t.Fatalf("HandleInteraction(mdl) = %v, want recovered panic", err)
	}
	if modalErrSeen.Load() == nil {
		t.Fatal("expected ModalError hook to fire on recovered panic")
	}

	if callbackHits.Load() != 1 {
		t.Fatalf("callback REST hits = %d, want 1", callbackHits.Load())
	}
}

func TestClientDecoupledContextAndShutdown(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(ClientConfig{
		Token:                 testBotToken("123456789012345678"),
		PublicKey:             hex.EncodeToString(pub),
		MaxConcurrentHandlers: 2,
		AckTimeout:            500 * time.Millisecond,
	})

	continuedAfterHTTP := make(chan bool, 1)
	if err := client.RegisterCommand(Command{
		Name: "async", Description: "Defer and continue",
		SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
			if err := itx.Defer(ctx, true); err != nil {
				return err
			}
			time.Sleep(30 * time.Millisecond)
			continuedAfterHTTP <- ctx.Err() == nil
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	body := `{"id":"1","application_id":"123456789012345678","type":2,"token":"tok","app_permissions":"0","data":{"id":"2","name":"async","type":1}}`
	client.ServeHTTP(rec, signedInteractionRequest(t, priv, body))

	var resp discord.ResponseMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Type != discord.DeferredChannelMessageWithSourceResponseType {
		t.Fatalf("response type = %d, want defer", resp.Type)
	}

	if ok := <-continuedAfterHTTP; !ok {
		t.Fatal("handler ctx was canceled when ServeHTTP returned after Defer")
	}

	if err := client.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Rejects new interactions after Shutdown.
	rec2 := httptest.NewRecorder()
	client.ServeHTTP(rec2, signedInteractionRequest(t, priv, `{"type":1}`))
	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("status after Shutdown = %d, want 503", rec2.Code)
	}
	if err := client.HandleInteraction(t.Context(), []byte(`{"type":1}`)); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("HandleInteraction after Shutdown = %v, want ErrClientClosed", err)
	}
}

func TestClientEdgeBranches(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubHex := hex.EncodeToString(pub)

	t.Run("ShutdownTimeout", func(t *testing.T) {
		c := NewClient(ClientConfig{
			Token:      testBotToken("123456789012345678"),
			PublicKey:  pubHex,
			AckTimeout: 20 * time.Millisecond,
		})

		unblock := make(chan struct{})
		started := make(chan struct{})
		_ = c.RegisterCommand(Command{
			Name: "block", Description: "Block",
			SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
				close(started)
				<-unblock
				return nil
			},
		})

		go func() {
			rec := httptest.NewRecorder()
			c.ServeHTTP(rec, signedInteractionRequest(t, priv, `{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"block","type":1}}`))
		}()
		<-started

		shortCtx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()
		if err := c.Shutdown(shortCtx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Shutdown(short) = %v, want DeadlineExceeded", err)
		}

		close(unblock)
		if err := c.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown(background) after handler exit = %v", err)
		}
	})

	t.Run("RESTHelperErrors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"bad","code":50001}`))
		}))
		defer srv.Close()

		c := NewClient(ClientConfig{
			Token:     testBotToken("123456789012345678"),
			PublicKey: pubHex,
			Rest: rest.NewRestClient(rest.RestClientConfig{
				Token:      testBotToken("123456789012345678"),
				BaseURL:    srv.URL,
				MaxRetries: 1,
			}),
		})
		defer func() { _ = c.Shutdown(t.Context()) }()

		ctx := t.Context()
		chID := discord.ChannelID{Snowflake: 1}
		uID := discord.UserID{Snowflake: 2}
		gID := discord.GuildID{Snowflake: 3}
		eID := discord.EntitlementID{Snowflake: 4}

		if _, err := c.SendLinearMessage(ctx, chID, "hi"); err == nil {
			t.Error("expected SendLinearMessage error")
		}
		if _, err := c.SendPrivateMessage(ctx, uID, discord.Message{Content: "hi"}, nil); err == nil {
			t.Error("expected SendPrivateMessage error")
		}
		if _, err := c.FetchUser(ctx, uID); err == nil {
			t.Error("expected FetchUser error")
		}
		if _, err := c.FetchMember(ctx, gID, uID); err == nil {
			t.Error("expected FetchMember error")
		}
		if _, err := c.FetchEntitlementsPage(ctx, "?exclude_ended=true"); err == nil {
			t.Error("expected FetchEntitlementsPage(?...) error")
		}
		if _, err := c.FetchEntitlement(ctx, eID); err == nil {
			t.Error("expected FetchEntitlement error")
		}

		_ = c.RegisterCommand(Command{Name: "a", Description: "a"})
		if err := c.SyncCommands(ctx, nil, nil, false); err == nil {
			t.Error("expected global SyncCommands error")
		}
		if err := c.SyncCommands(ctx, []discord.GuildID{gID}, nil, false); err == nil {
			t.Error("expected guild SyncCommands error")
		}
	})

	t.Run("FindInteractiveComponentTopAndMiss", func(t *testing.T) {
		rowChildren := []discord.ActionRowChildComponent{
			discord.ButtonComponent{Type: discord.ButtonComponentType, CustomID: "top-btn"},
		}
		if b, ok := FindInteractiveComponent(rowChildren, func(b discord.ButtonComponent) bool { return b.CustomID == "top-btn" }); !ok || b.CustomID != "top-btn" {
			t.Fatalf("expected top-btn match, got (%+v, %t)", b, ok)
		}

		tree := []discord.MessageComponent{
			discord.ActionRowComponent{
				Type:       discord.ActionRowComponentType,
				Components: []discord.ActionRowChildComponent{discord.ButtonComponent{Type: discord.ButtonComponentType, CustomID: "row-btn"}},
			},
			discord.SectionComponent{
				Type:      discord.SectionComponentType,
				Accessory: nil,
			},
			discord.ContainerComponent{
				Type: discord.ContainerComponentType,
				Components: []discord.ContainerChildComponent{
					discord.TextDisplayComponent{Type: discord.TextDisplayComponentType, Content: "hi"},
				},
			},
		}

		if _, ok := FindInteractiveComponent(tree, func(b discord.ButtonComponent) bool { return b.CustomID == "missing" }); ok {
			t.Fatal("expected miss on missing button")
		}
	})

	t.Run("NilPoolsAndSessionEdgeCases", func(t *testing.T) {
		var zeroClient Client
		zeroClient.tracef("ignored %d", 1)

		timer := zeroClient.acquireTimer(10 * time.Millisecond)
		zeroClient.releaseTimer(timer)

		lr := zeroClient.acquireLimitedReader(strings.NewReader("abc"), 2)
		zeroClient.releaseLimitedReader(lr)

		buf := zeroClient.acquireBuffer()
		buf.Grow(maxPooledBodyBufferCap + 1024)
		zeroClient.releaseBuffer(nil)
		zeroClient.releaseBuffer(buf)

		if err := (&interactionContext{}).respond(t.Context(), &discord.Interaction{}, &Response{}); err == nil {
			t.Fatal("expected error with nil Client")
		}

		c, priv2 := newTestClientPair(t, ClientHooks{})
		ic := interactionContext{Client: c}
		canceledCtx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := ic.respond(canceledCtx, &discord.Interaction{}, &Response{}); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled respond = %v, want Canceled", err)
		}

		bigBuf := c.acquireBuffer()
		bigBuf.Grow(maxPooledBodyBufferCap + 4096)
		c.releaseBuffer(bigBuf)

		s := c.acquireSession()
		s.tryRespond([]byte("stale"), c.acquireBuffer())
		s.releaseRef()
		s.releaseRef()

		reused := c.acquireSession()
		reused.finishEmpty()
		reused.releaseRef()
		reused.releaseRef()

		type baseKey struct{}
		c.baseCtx = context.WithValue(c.baseCtx, baseKey{}, "base-v")
		type reqKey struct{}
		rCtx := context.WithValue(t.Context(), reqKey{}, "req-v")

		s2 := c.acquireSession()
		hCtx, hCancel, stop := c.interactionContext(rCtx, s2)
		if hCtx.Value(baseKey{}) != "base-v" || hCtx.Value(reqKey{}) != "req-v" {
			t.Fatalf("mergedValueCtx values failed: %v %v", hCtx.Value(baseKey{}), hCtx.Value(reqKey{}))
		}
		if s2.hasResponded() {
			t.Fatal("expected hasResponded false initially")
		}
		if stop != nil {
			stop()
		}
		hCancel()
		s2.releaseRef()
		s2.releaseRef()

		req := httptest.NewRequest(http.MethodPost, "/", failingCloseBody{Reader: strings.NewReader(`{"type":1}`)})
		msg := append([]byte("1700"), `{"type":1}`...)
		req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(ed25519.Sign(priv2, msg)))
		req.Header.Set("X-Signature-Timestamp", "1700")
		if _, bodyBuf, ok := c.verifyRequest(req, c.PublicKey, 1024); !ok {
			t.Fatal("expected verifyRequest to succeed even if Close errors")
		} else {
			c.releaseBuffer(bodyBuf)
		}

		var dst [4]byte
		if decodeHex(dst[:], "ab") {
			t.Fatal("expected decodeHex short input false")
		}
		if decodeHex(dst[:], "abcz0000") {
			t.Fatal("expected decodeHex bad low nibble false")
		}

		c.writeJSON(failingWriter{}, []byte(`{"ok":true}`))

		for _, raw := range []string{`{"a":`, `{"a":1`, `{"type":1e999}`} {
			if _, err := peekInteractionType([]byte(raw)); err == nil {
				t.Errorf("expected peekInteractionType(%q) error", raw)
			}
		}
	})

	t.Run("AlreadyRespondedAndDeferEdgeCases", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			if strings.HasSuffix(r.URL.Path, "/fail-tok") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		c, priv2 := newTestClientPair(t, ClientHooks{})
		c.Rest = rest.NewRestClient(rest.RestClientConfig{
			Token:      testBotToken("123456789012345678"),
			BaseURL:    srv.URL,
			MaxRetries: 1,
		})

		_ = c.RegisterCommand(Command{
			Name: "nil-handler", Description: "nil",
		})
		if err := c.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":2,"token":"t","app_permissions":"0","data":{"id":"2","name":"nil-handler","type":1}}`)); err == nil {
			t.Fatal("expected error for nil SlashCommandHandler")
		}

		for _, payload := range []string{
			`not-json`,
			`{"type":2,"data":123}`,
			`{"type":3,"data":123}`,
			`{"type":4,"data":123}`,
			`{"type":5,"data":123}`,
		} {
			if err := c.HandleInteraction(t.Context(), []byte(payload)); err == nil {
				t.Errorf("expected HandleInteraction(%q) error", payload)
			}
		}

		_ = c.RegisterCommand(Command{
			Name: "double", Description: "double",
			SlashCommandHandler: func(ctx context.Context, itx *CommandInteraction) error {
				if err := itx.Defer(ctx, true); err != nil {
					return err
				}
				if !errors.Is(itx.Defer(ctx, true), ErrAlreadyDeferred) {
					return errors.New("expected ErrAlreadyDeferred on second Defer")
				}
				if err := itx.SendLinearReply(ctx, "ok", false); err != nil {
					return err
				}
				if !errors.Is(itx.SendLinearReply(ctx, "again", false), ErrAlreadyResponded) {
					return errors.New("expected ErrAlreadyResponded on second SendReply")
				}
				return nil
			},
		})
		if err := c.HandleInteraction(t.Context(), []byte(`{"id":"1","application_id":"123456789012345678","type":2,"token":"ok-tok","app_permissions":"0","data":{"id":"2","name":"double","type":1}}`)); err != nil {
			t.Fatalf("double command = %v", err)
		}

		itxCmd := &CommandInteraction{
			Client: c,
			ID:     1, ApplicationID: discord.AppID{Snowflake: 1}, Token: "fail-tok",
		}
		if _, err := itxCmd.SendLinearFollowUp(t.Context(), "fail", true); err == nil {
			t.Fatal("expected SendLinearFollowUp error on 500 server")
		}

		itxCmp := &ComponentInteraction{
			Client: c, responded: true,
		}
		if !errors.Is(itxCmp.AcknowledgeWithLinearMessage(t.Context(), "x", true), ErrAlreadyResponded) {
			t.Fatal("expected ErrAlreadyResponded on ComponentInteraction.AcknowledgeWithLinearMessage")
		}
		if !errors.Is(itxCmp.AcknowledgeWithLinearUpdate(t.Context(), "x"), ErrAlreadyResponded) {
			t.Fatal("expected ErrAlreadyResponded on ComponentInteraction.AcknowledgeWithLinearUpdate")
		}
		if !errors.Is(itxCmp.Defer(t.Context(), true), ErrAlreadyDeferred) {
			t.Fatal("expected ErrAlreadyDeferred on ComponentInteraction.Defer")
		}
		if !errors.Is(itxCmp.AcknowledgeWithModal(t.Context(), discord.ResponseModalData{CustomID: "m", Title: "t"}), ErrAlreadyDeferred) {
			t.Fatal("expected ErrAlreadyDeferred on ComponentInteraction.AcknowledgeWithModal")
		}

		itxMdl := &ModalInteraction{
			Client: c, responded: true,
		}
		if !errors.Is(itxMdl.Defer(t.Context(), true), ErrAlreadyDeferred) {
			t.Fatal("expected ErrAlreadyDeferred on ModalInteraction.Defer")
		}
		if !errors.Is(itxMdl.AcknowledgeWithLinearMessage(t.Context(), "x", true), ErrAlreadyResponded) {
			t.Fatal("expected ErrAlreadyResponded on ModalInteraction.AcknowledgeWithLinearMessage")
		}
		if !errors.Is(itxMdl.AcknowledgeWithLinearUpdate(t.Context(), "x"), ErrAlreadyResponded) {
			t.Fatal("expected ErrAlreadyResponded on ModalInteraction.AcknowledgeWithLinearUpdate")
		}

		_ = c.RegisterCommand(Command{
			Name: "acfail", Description: "acfail",
			AutoCompleteHandler: func(ctx context.Context, itx *CommandInteraction) ([]discord.CommandOptionChoice, error) {
				panic(errors.New("err panic"))
			},
		})
		recAC := httptest.NewRecorder()
		c.ServeHTTP(recAC, signedInteractionRequest(t, priv2, `{"id":"1","application_id":"123456789012345678","type":4,"token":"t","app_permissions":"0","data":{"id":"2","name":"acfail","type":1}}`))
		if recAC.Code != http.StatusInternalServerError {
			t.Fatalf("autocomplete panic status = %d, want 500", recAC.Code)
		}

		pr := newPatternRouter[string]()
		_ = pr.register("a:{x}", "1")
		_ = pr.register("aa:{x}", "2")
		_ = pr.register("b:{x}:{y}", "3")
		_ = pr.register("ab:{x}", "4")
		_ = pr.register("ab:{x}", "dup")
		if pr.unregister("") || pr.unregister("{missing}") {
			t.Fatal("expected empty/missing unregister false")
		}
		for range pr.all() {
			break
		}
		for range c.RegisteredCommands() {
			break
		}
	})
}
