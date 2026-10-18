package tempest

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
)

// TestSyncCommandsDefaultsZeroType proves a zero Type is sent as CommandChatInput.
func TestSyncCommandsDefaultsZeroType(t *testing.T) {
	var (
		got  []discord.Command
		path string
		logs []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	c, err := New(Config{
		Token:         "x",
		ApplicationID: discord.AppID{Snowflake: 123},
		Debug:         true,
		Logf:          func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.rest = rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})

	commands := []discord.Command{
		{Name: "plain", Description: "no type"},
		{Name: "user", Type: discord.CommandUser},
	}

	if err := c.SyncCommands(context.Background(), discord.GuildID{}, commands, nil); err != nil {
		t.Fatalf("SyncCommands: %v", err)
	}

	if path != "/applications/123/commands" {
		t.Fatalf("path = %s", path)
	}

	if len(got) != 2 {
		t.Fatalf("sent %d commands, want 2", len(got))
	}

	if got[0].Type != discord.CommandChatInput {
		t.Errorf("zero type sent as %d, want CommandChatInput", got[0].Type)
	}

	if got[1].Type != discord.CommandUser {
		t.Errorf("explicit type = %d, want CommandUser", got[1].Type)
	}

	if !strings.Contains(strings.Join(logs, "\n"), `command "plain" had no type`) {
		t.Errorf("missing debug log, got %v", logs)
	}
}
