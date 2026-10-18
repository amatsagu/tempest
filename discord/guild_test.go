package discord

import (
	"encoding/json/v2"
	"testing"
)

// TestGuildParsesGatewayPayload decodes a GUILD_CREATE payload, whose guild fields
// are promoted onto the event data, and checks the derived helpers.
func TestGuildParsesGatewayPayload(t *testing.T) {
	raw := `{
		"id": "41771983423143937",
		"name": "Test Guild",
		"icon": "a_1234567890abcdef",
		"splash": null,
		"discovery_splash": "fedcba",
		"banner": "abcdef",
		"owner_id": "80351110224678912",
		"afk_channel_id": null,
		"afk_timeout": 300,
		"widget_enabled": false,
		"widget_channel_id": "123",
		"verification_level": 2,
		"default_message_notifications": 1,
		"explicit_content_filter": 2,
		"mfa_level": 1,
		"application_id": null,
		"system_channel_id": "456",
		"system_channel_flags": 5,
		"rules_channel_id": "789",
		"public_updates_channel_id": null,
		"safety_alerts_channel_id": null,
		"preferred_locale": "en-US",
		"nsfw_level": 3,
		"premium_tier": 2,
		"premium_subscription_count": 4,
		"premium_progress_bar_enabled": true,
		"max_members": 1000,
		"max_presences": 50,
		"max_video_channel_users": 25,
		"approximate_member_count": 900,
		"approximate_presence_count": 100,
		"vanity_url_code": "test",
		"description": "A test guild",
		"features": ["COMMUNITY", "DISCOVERABLE"],
		"roles": [{"id": "1", "name": "everyone"}],
		"emojis": [],
		"stickers": [],
		"joined_at": "2026-01-02T03:04:05.000000+00:00",
		"member_count": 900,
		"large": false,
		"unavailable": false,
		"members": [{"joined_at": "2026-01-02T03:04:05.000000+00:00", "nick": "tester"}],
		"channels": [{"id": "456", "type": 0, "name": "general"}],
		"threads": []
	}`

	var data GuildCreateData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if data.Guild == nil {
		t.Fatal("embedded guild was not decoded")
	}

	if data.ID != (GuildID{Snowflake: 41771983423143937}) || data.Name != "Test Guild" {
		t.Fatalf("id/name = %s/%q", data.ID, data.Name)
	}

	if data.OwnerID != (UserID{Snowflake: 80351110224678912}) {
		t.Errorf("OwnerID = %s, want 80351110224678912", data.OwnerID)
	}

	if data.ApplicationID.Snowflake != 0 {
		t.Errorf("ApplicationID = %d, want 0 for null", data.ApplicationID)
	}

	if data.AFKChannelID.Snowflake != 0 || data.SystemChannelID != (ChannelID{Snowflake: 456}) {
		t.Errorf("channel ids = %s/%s", data.AFKChannelID, data.SystemChannelID)
	}

	if data.PreferredLocale != LocaleEnglishUS {
		t.Errorf("PreferredLocale = %q, want en-US", data.PreferredLocale)
	}

	// The enums and the bitfield, which are the part that has to be exactly right.
	if data.VerificationLevel != VerificationMedium {
		t.Errorf("VerificationLevel = %d, want medium", data.VerificationLevel)
	}

	if data.DefaultMessageNotifications != NotificationOnlyMentions {
		t.Errorf("DefaultMessageNotifications = %d, want only mentions", data.DefaultMessageNotifications)
	}

	if data.ExplicitContentFilter != ContentFilterAllMembers {
		t.Errorf("ExplicitContentFilter = %d, want all members", data.ExplicitContentFilter)
	}

	if data.MFALevel != MFAElevated {
		t.Errorf("MFALevel = %d, want elevated", data.MFALevel)
	}

	if data.NSFWLevel != NSFWLevelAgeRestricted {
		t.Errorf("NSFWLevel = %d, want age restricted", data.NSFWLevel)
	}

	if data.PremiumTier != PremiumTier2 {
		t.Errorf("PremiumTier = %d, want tier 2", data.PremiumTier)
	}

	// 5 is SUPPRESS_JOIN_NOTIFICATIONS (1 << 0) plus SUPPRESS_GUILD_REMINDER (1 << 2).
	if !data.SystemChannelFlags.Has(SystemChannelFlagSuppressJoinNotifications, SystemChannelFlagSuppressGuildReminderNotifications) {
		t.Errorf("SystemChannelFlags = %d, want join and reminder bits", data.SystemChannelFlags)
	}

	if !data.HasFeature("COMMUNITY") || data.HasFeature("PRIVATE_THREADS") {
		t.Errorf("HasFeature = %v/%v, want true/false", data.HasFeature("COMMUNITY"), data.HasFeature("PRIVATE_THREADS"))
	}

	// The asset urls, where an animated hash has to use webp.
	if got, want := data.IconURL(), CDNBaseURL+"/icons/41771983423143937/a_1234567890abcdef.webp?animated=true"; got != want {
		t.Errorf("IconURL = %q, want %q", got, want)
	}

	if got := data.SplashURL(); got != "" {
		t.Errorf("SplashURL = %q, want empty for a null splash", got)
	}

	if got, want := data.DiscoverySplashURL(), CDNBaseURL+"/discovery-splashes/41771983423143937/fedcba.png"; got != want {
		t.Errorf("DiscoverySplashURL = %q, want %q", got, want)
	}

	if got, want := data.BannerURL(), CDNBaseURL+"/banners/41771983423143937/abcdef.png"; got != want {
		t.Errorf("BannerURL = %q, want %q", got, want)
	}

	// The extra state Discord sends with the guild.
	if data.JoinedAt == nil {
		t.Error("JoinedAt = nil, want the joined timestamp")
	}

	if data.MemberCount != 900 || len(data.Members) != 1 || len(data.Channels) != 1 {
		t.Errorf("member count/members/channels = %d/%d/%d", data.MemberCount, len(data.Members), len(data.Channels))
	}

	if len(data.Roles) != 1 || data.Roles[0].Name != "everyone" {
		t.Errorf("roles = %+v", data.Roles)
	}
}

// TestGuildCreateUnavailable proves the unavailable shape, which carries only an id,
// still decodes into the same type.
func TestGuildCreateUnavailable(t *testing.T) {
	var data GuildCreateData
	if err := json.Unmarshal([]byte(`{"id":"1","unavailable":true}`), &data); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !data.Unavailable {
		t.Error("Unavailable = false, want true")
	}

	if data.Guild == nil || data.ID != (GuildID{Snowflake: 1}) {
		t.Fatalf("guild = %+v, want it to carry id 1", data.Guild)
	}

	if data.Name != "" {
		t.Errorf("Name = %q, want empty for an unavailable guild", data.Name)
	}
}

func TestUnavailableGuildRoundTrip(t *testing.T) {
	unavailable := UnavailableGuild{ID: GuildID{Snowflake: 7}, Unavailable: true}

	data, err := json.Marshal(unavailable)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var parsed UnavailableGuild
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if parsed != unavailable {
		t.Errorf("parsed = %+v, want %+v", parsed, unavailable)
	}
}

// TestGuildEnumValues pins the documented integers, since a wrong level silently
// changes what a bot permits.
func TestGuildEnumValues(t *testing.T) {
	notifications := map[MessageNotificationLevel]uint8{
		NotificationAllMessages:  0,
		NotificationOnlyMentions: 1,
	}

	for got, want := range notifications {
		if uint8(got) != want {
			t.Errorf("notification level = %d, want %d", got, want)
		}
	}

	filters := map[ExplicitContentFilter]uint8{
		ContentFilterDisabled:            0,
		ContentFilterMembersWithoutRoles: 1,
		ContentFilterAllMembers:          2,
	}

	for got, want := range filters {
		if uint8(got) != want {
			t.Errorf("content filter = %d, want %d", got, want)
		}
	}

	mfa := map[MFALevel]uint8{
		MFANone:     0,
		MFAElevated: 1,
	}

	for got, want := range mfa {
		if uint8(got) != want {
			t.Errorf("mfa level = %d, want %d", got, want)
		}
	}

	verification := map[VerificationLevel]uint8{
		VerificationNone:     0,
		VerificationLow:      1,
		VerificationMedium:   2,
		VerificationHigh:     3,
		VerificationVeryHigh: 4,
	}

	for got, want := range verification {
		if uint8(got) != want {
			t.Errorf("verification level = %d, want %d", got, want)
		}
	}

	nsfw := map[NSFWLevel]uint8{
		NSFWLevelDefault:       0,
		NSFWLevelExplicit:      1,
		NSFWLevelSafe:          2,
		NSFWLevelAgeRestricted: 3,
	}

	for got, want := range nsfw {
		if uint8(got) != want {
			t.Errorf("nsfw level = %d, want %d", got, want)
		}
	}

	premium := map[PremiumTier]uint8{
		PremiumTierNone: 0,
		PremiumTier1:    1,
		PremiumTier2:    2,
		PremiumTier3:    3,
	}

	for got, want := range premium {
		if uint8(got) != want {
			t.Errorf("premium tier = %d, want %d", got, want)
		}
	}

	systemFlags := map[SystemChannelFlags]uint8{
		SystemChannelFlagSuppressJoinNotifications:               0,
		SystemChannelFlagSuppressPremiumSubscriptions:            1,
		SystemChannelFlagSuppressGuildReminderNotifications:      2,
		SystemChannelFlagSuppressJoinNotificationReplies:         3,
		SystemChannelFlagSuppressRoleSubscriptionPurchase:        4,
		SystemChannelFlagSuppressRoleSubscriptionPurchaseReplies: 5,
	}

	checkBits(t, "system channel flag", systemFlags)
}
