package tempest

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/ratelimit"
	"github.com/amatsagu/tempest/rest"
)

// syncHarness builds a Client whose REST calls hit a local server that decodes
// each request body into commands.
func syncHarness(t *testing.T, cfg Config) (*Client, *[]discord.Command) {
	t.Helper()

	got := new([]discord.Command)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(server.Close)

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.rest = rest.New("t", rest.Options{
		BaseURL: server.URL,
		Limiter: ratelimit.New(ratelimit.Options{GlobalRate: 1e9}),
	})
	return c, got
}

// TestSyncCommandsFilter covers the three CommandFilter shapes: nil sends every
// command, Names alone is a whitelist, and Exclude flips it into a blacklist.
func TestSyncCommandsFilter(t *testing.T) {
	commands := []discord.Command{
		{Name: "keep"},
		{Name: "drop"},
		{Name: "other"},
	}

	cases := []struct {
		name   string
		filter *CommandFilter
		want   []string
	}{
		{"nil sends all", nil, []string{"keep", "drop", "other"}},
		{"whitelist", &CommandFilter{Names: []string{"keep"}}, []string{"keep"}},
		{"blacklist", &CommandFilter{Names: []string{"drop"}, Exclude: true}, []string{"keep", "other"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, got := syncHarness(t, Config{Token: "x", ApplicationID: discord.AppID{Snowflake: 123}})

			if err := c.SyncCommands(context.Background(), discord.GuildID{}, commands, tc.filter); err != nil {
				t.Fatalf("SyncCommands: %v", err)
			}

			names := make([]string, len(*got))
			for i, cmd := range *got {
				names[i] = cmd.Name
			}

			if !slices.Equal(names, tc.want) {
				t.Fatalf("synced %v, want %v", names, tc.want)
			}
		})
	}
}

// TestSyncCommandsDefaultContexts proves a command without Contexts inherits
// Config.DefaultContexts while one with its own keeps them.
func TestSyncCommandsDefaultContexts(t *testing.T) {
	defaults := []discord.InteractionContextType{discord.InteractionContextPrivateChannel}
	own := []discord.InteractionContextType{discord.InteractionContextGuild}

	c, got := syncHarness(t, Config{
		Token:           "x",
		ApplicationID:   discord.AppID{Snowflake: 123},
		DefaultContexts: defaults,
	})

	commands := []discord.Command{
		{Name: "inherit"},
		{Name: "own", Contexts: own},
	}

	if err := c.SyncCommands(context.Background(), discord.GuildID{}, commands, nil); err != nil {
		t.Fatalf("SyncCommands: %v", err)
	}

	if len(*got) != 2 {
		t.Fatalf("sent %d commands, want 2", len(*got))
	}

	if !slices.Equal((*got)[0].Contexts, defaults) {
		t.Errorf("contexts of %q = %v, want the default %v", (*got)[0].Name, (*got)[0].Contexts, defaults)
	}

	if !slices.Equal((*got)[1].Contexts, own) {
		t.Errorf("contexts of %q = %v, want its own %v", (*got)[1].Name, (*got)[1].Contexts, own)
	}
}
