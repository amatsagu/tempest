package discord

import (
	json "encoding/json/v2"
	"testing"
	"time"
)

func TestDiscordCDNBaseURL(t *testing.T) {
	original := DiscordCDNBaseURL()
	t.Cleanup(func() {
		UpdateDiscordCDNBaseURL(original)
	})

	if original != "https://cdn.discordapp.com" {
		t.Fatalf("default CDN URL = %q, want https://cdn.discordapp.com", original)
	}

	UpdateDiscordCDNBaseURL("https://cdn.example.test")
	if got := DiscordCDNBaseURL(); got != "https://cdn.example.test" {
		t.Fatalf("updated CDN URL = %q, want https://cdn.example.test", got)
	}
}

func TestGuildURLs(t *testing.T) {
	empty := Guild{ID: GuildID{Snowflake: 12345}}
	if empty.IconURL() != "" || empty.SplashURL() != "" || empty.DiscoverySplashURL() != "" || empty.BannerURL() != "" {
		t.Fatal("expected empty URLs when hashes are empty")
	}

	static := Guild{
		ID:                  GuildID{Snowflake: 12345},
		IconHash:            "iconhash",
		SplashHash:          "splashhash",
		DiscoverySplashHash: "dsplashhash",
		BannerHash:          "bannerhash",
	}
	base := DiscordCDNBaseURL()
	if got, want := static.IconURL(), base+"/icons/12345/iconhash"; got != want {
		t.Fatalf("IconURL = %q, want %q", got, want)
	}
	if got, want := static.SplashURL(), base+"/splashes/12345/splashhash"; got != want {
		t.Fatalf("SplashURL = %q, want %q", got, want)
	}
	if got, want := static.DiscoverySplashURL(), base+"/discovery-splashes/12345/dsplashhash"; got != want {
		t.Fatalf("DiscoverySplashURL = %q, want %q", got, want)
	}
	if got, want := static.BannerURL(), base+"/banners/12345/bannerhash"; got != want {
		t.Fatalf("BannerURL = %q, want %q", got, want)
	}

	animated := Guild{
		ID:         GuildID{Snowflake: 12345},
		IconHash:   "a_icon",
		BannerHash: "a_banner",
	}
	if got, want := animated.IconURL(), base+"/icons/12345/a_icon.webp?animated=true"; got != want {
		t.Fatalf("animated IconURL = %q, want %q", got, want)
	}
	if got, want := animated.BannerURL(), base+"/banners/12345/a_banner.webp?animated=true"; got != want {
		t.Fatalf("animated BannerURL = %q, want %q", got, want)
	}
}

func TestMentionableHelpers(t *testing.T) {
	base := DiscordCDNBaseURL()

	t.Run("avatar_decoration", func(t *testing.T) {
		empty := &AvatarDecoration{}
		if got := empty.DecorationURL(); got != "" {
			t.Fatalf("empty DecorationURL = %q, want empty", got)
		}

		dec := &AvatarDecoration{AssetHash: "v_abc"}
		if got, want := dec.DecorationURL(), base+"/avatar-decoration-presets/v_abc"; got != want {
			t.Fatalf("DecorationURL = %q, want %q", got, want)
		}

		animDec := &AvatarDecoration{AssetHash: "a_decor"}
		if got, want := animDec.DecorationURL(), base+"/avatar-decoration-presets/a_decor.webp?animated=true"; got != want {
			t.Fatalf("animated DecorationURL = %q, want %q", got, want)
		}
	})

	t.Run("user", func(t *testing.T) {
		defaultUser := User{
			ID:       UserID{Snowflake: 1159846879854792704},
			Username: "amatsagu",
		}
		if got := defaultUser.Mention(); got != "<@1159846879854792704>" {
			t.Fatalf("Mention = %q", got)
		}
		if got := defaultUser.BannerURL(); got != "" {
			t.Fatalf("empty BannerURL = %q, want empty", got)
		}
		if got := defaultUser.AvatarURL(); got == "" || got[:len(base)] != base {
			t.Fatalf("unexpected default AvatarURL = %q", got)
		}

		staticUser := User{
			ID:         UserID{Snowflake: 1001},
			Username:   "dev",
			GlobalName: "Dev User",
			AvatarHash: "avhash",
			BannerHash: "bnhash",
		}
		if got, want := staticUser.AvatarURL(), base+"/avatars/1001/avhash"; got != want {
			t.Fatalf("static AvatarURL = %q, want %q", got, want)
		}
		if got, want := staticUser.BannerURL(), base+"/banners/1001/bnhash"; got != want {
			t.Fatalf("static BannerURL = %q, want %q", got, want)
		}

		animUser := User{
			ID:         UserID{Snowflake: 1001},
			AvatarHash: "a_av",
			BannerHash: "a_bn",
		}
		if got, want := animUser.AvatarURL(), base+"/avatars/1001/a_av.webp?animated=true"; got != want {
			t.Fatalf("animated AvatarURL = %q, want %q", got, want)
		}
		if got, want := animUser.BannerURL(), base+"/banners/1001/a_bn.webp?animated=true"; got != want {
			t.Fatalf("animated BannerURL = %q, want %q", got, want)
		}
	})

	t.Run("member", func(t *testing.T) {
		noCustom := Member{GuildID: GuildID{Snowflake: 777}, User: &User{ID: UserID{Snowflake: 1001}}}
		if noCustom.GuildAvatarURL() != "" || noCustom.GuildBannerURL() != "" {
			t.Fatal("expected empty guild avatar/banner URL when hashes empty")
		}

		nilUser := Member{GuildID: GuildID{Snowflake: 777}, GuildAvatarHash: "av"}
		if nilUser.GuildAvatarURL() != "" {
			t.Fatal("expected empty GuildAvatarURL when User is nil")
		}

		noGuild := Member{GuildAvatarHash: "av", GuildBannerHash: "bn", User: &User{ID: UserID{Snowflake: 1001}}}
		assertPanics(t, "GuildAvatarURL without GuildID", func() { _ = noGuild.GuildAvatarURL() })
		assertPanics(t, "GuildBannerURL without GuildID", func() { _ = noGuild.GuildBannerURL() })

		member := Member{
			GuildID:         GuildID{Snowflake: 777},
			User:            &User{ID: UserID{Snowflake: 1001}},
			GuildAvatarHash: "gav",
			GuildBannerHash: "gbn",
		}
		if got, want := member.GuildAvatarURL(), base+"/guilds/777/users/1001/avatars/gav"; got != want {
			t.Fatalf("GuildAvatarURL = %q, want %q", got, want)
		}
		if got, want := member.GuildBannerURL(), base+"/guilds/777/users/1001/banners/gbn"; got != want {
			t.Fatalf("GuildBannerURL = %q, want %q", got, want)
		}

		animMember := Member{
			GuildID:         GuildID{Snowflake: 777},
			User:            &User{ID: UserID{Snowflake: 1001}},
			GuildAvatarHash: "a_gav",
			GuildBannerHash: "a_gbn",
		}
		if got, want := animMember.GuildAvatarURL(), base+"/guilds/777/users/1001/avatars/a_gav.webp?animated=true"; got != want {
			t.Fatalf("animated GuildAvatarURL = %q, want %q", got, want)
		}
		if got, want := animMember.GuildBannerURL(), base+"/guilds/777/users/1001/banners/a_gbn.webp?animated=true"; got != want {
			t.Fatalf("animated GuildBannerURL = %q, want %q", got, want)
		}
	})

	t.Run("role", func(t *testing.T) {
		role := Role{ID: RoleID{Snowflake: 555}, Name: "Mod"}
		if got := role.Mention(); got != "<@&555>" {
			t.Fatalf("Role.Mention = %q, want <@&555>", got)
		}
		if got := role.IconURL(); got != "" {
			t.Fatalf("empty Role.IconURL = %q, want empty", got)
		}

		role.IconHash = "icon123"
		if got, want := role.IconURL(), base+"/role-icons/555/icon123.png"; got != want {
			t.Fatalf("Role.IconURL = %q, want %q", got, want)
		}

		role.IconHash = "a_icon"
		if got, want := role.IconURL(), base+"/role-icons/555/a_icon.webp?animated=true"; got != want {
			t.Fatalf("animated Role.IconURL = %q, want %q", got, want)
		}
	})
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: expected panic, did not panic", name)
		}
	}()
	fn()
}

func BenchmarkUser_JSON(b *testing.B) {
	u := User{
		ID:         UserID{Snowflake: 1159846879854792704},
		Username:   "amatsagu",
		GlobalName: "Amatsagu Dev",
		AvatarHash: "abcdef123456",
		Locale:     "en-US",
		Bot:        true,
	}
	data, err := json.Marshal(&u)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed User
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMember_JSON(b *testing.B) {
	joined := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	m := Member{
		JoinedAt:        joined,
		Nickname:        "Gopher",
		GuildAvatarHash: "gav123",
		RoleIDs:         []RoleID{{Snowflake: 10}, {Snowflake: 20}, {Snowflake: 30}},
		PermissionFlags: AllTextPermissionFlags,
		User: &User{
			ID:       UserID{Snowflake: 1001},
			Username: "gopher",
		},
	}
	data, err := json.Marshal(&m)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Member
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGuild_JSON(b *testing.B) {
	g := Guild{
		ID:                        GuildID{Snowflake: 123456789012345678},
		Name:                      "Tempest Dev",
		OwnerID:                   UserID{Snowflake: 987654321098765432},
		IconHash:                  "icon123",
		BannerHash:                "banner123",
		PreferredLocale:           "en-US",
		AFKChannelTimeout:         300,
		ApproximateMemberCount:    420,
		Features:                  []string{"COMMUNITY", "NEWS"},
		VerificationLevel:         MediumVerificationLevel,
		PremiumTier:               Boost2PremiumTier,
		ExplicitContentFilter:     AllMembersContentFilter,
		SystemChannelFlags:        SuppressJoinNotificationsSystemFlag,
		Roles:                     []Role{{ID: RoleID{Snowflake: 1}, Name: "@everyone"}},
		Emojis:                    []Emoji{{ID: EmojiID{Snowflake: 2}, Name: "fast"}},
		SystemChannelID:           ChannelID{Snowflake: 333},
		PublicUpdatesChannelID:    ChannelID{Snowflake: 444},
		PremiumProgressBarEnabled: true,
	}
	data, err := json.Marshal(&g)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Guild
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&g); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRole_JSON(b *testing.B) {
	secondary := uint32(0x112233)
	r := Role{
		ID:              RoleID{Snowflake: 555666777888},
		Name:            "Maintainer",
		PermissionFlags: AdministratorPermissionFlag,
		Colors:          RoleColors{PrimaryColor: 0x5865F2, SecondaryColor: &secondary},
		Position:        5,
		Hoist:           true,
		Mentionable:     true,
		Tags:            &RoleTag{BotID: UserID{Snowflake: 1001}},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var parsed Role
		if err := json.Unmarshal(data, &parsed); err != nil {
			b.Fatal(err)
		}
		if _, err := json.Marshal(&r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCDNURLs(b *testing.B) {
	guild := Guild{ID: GuildID{Snowflake: 123456789012345678}, IconHash: "a_iconhash", BannerHash: "bannerhash"}
	user := User{ID: UserID{Snowflake: 987654321098765432}, AvatarHash: "avatarhash"}
	userDefault := User{ID: UserID{Snowflake: 987654321098765432}}
	member := Member{GuildID: guild.ID, User: &user, GuildAvatarHash: "gavhash"}
	role := Role{ID: RoleID{Snowflake: 555666777888}, IconHash: "roleicon"}
	dec := AvatarDecoration{AssetHash: "v_decor"}

	b.ReportAllocs()
	for b.Loop() {
		_ = guild.IconURL()
		_ = guild.BannerURL()
		_ = user.AvatarURL()
		_ = userDefault.AvatarURL()
		_ = member.GuildAvatarURL()
		_ = role.IconURL()
		_ = dec.DecorationURL()
	}
}
