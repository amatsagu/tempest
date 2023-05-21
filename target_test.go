package tempest

import (
	"testing"

	"github.com/sugawarayuuta/sonnet"
)

func TestUser(t *testing.T) {
	const exampleUser = `{
		"id": "80351110224678912",
		"username": "Nelly",
		"discriminator": "1337",
		"avatar": "8342729096ea3675442027381ff50dfe",
		"verified": true,
		"email": "nelly@discord.com",
		"flags": 64,
		"banner": "a_06c16474723fe537c283b8efa61a30c8",
		"accent_color": 16711680,
		"premium_type": 1,
		"public_flags": 64
	}`

	var user User
	if err := sonnet.Unmarshal([]byte(exampleUser), &user); err != nil {
		panic("failed to parse example user (json) object")
	}

	if user.ID != 80351110224678912 {
		panic("parsed user has invalid ID")
	}

	if user.Username != "Nelly" {
		panic("parsed user has invalid username")
	}

	if user.AvatarHash == "" {
		panic("parsed user avatar hash data is lost")
	}

	validAvatarURL := DISCORD_CDN_URL + "/avatars/" + user.ID.String() + "/" + user.AvatarHash
	if user.AvatarURL() != validAvatarURL {
		panic("parsed user has invalid avatar url")
	}

	if user.BannerHash == "" {
		panic("parsed user banner hash data is lost")
	}

	validBannerURL := DISCORD_CDN_URL + "/banners/" + user.ID.String() + "/" + user.BannerHash + ".gif"
	if user.BannerURL() != validBannerURL {
		panic("parsed user has invalid banner url")
	}

	if user.AccentColor != 16711680 {
		panic("parsed user has invalid accent color")
	}

	if user.PremiumType != CLASSIC_NITRO_TYPE {
		panic("parsed user has invalid premium (nitro) type")
	}

	if user.PublicFlags == 0 {
		panic("parsed user (public) flags data is lost")
	}

	if user.Mention() != "<@"+user.ID.String()+">" {
		panic("parsed user couldn't be @mentioned")
	}
}

func TestMember(t *testing.T) {
	const exampleMember = `{
		"user": {},
		"nick": "Mike",
		"avatar": null,
		"roles": [],
		"joined_at": "2015-04-26T06:26:56.936000+00:00",
		"deaf": false,
		"mute": false
	}`

	var member Member
	if err := sonnet.Unmarshal([]byte(exampleMember), &member); err != nil {
		panic("failed to parse example member (json) object")
	}

	if member.Nickname != "Mike" {
		panic("parsed member has invalid nickname")
	}

	if member.GuildAvatarHash != "" {
		panic("parsed member guild avatar hash data is invalid")
	}

	if member.JoinedAt.IsZero() {
		panic("parsed member joined at date is invalid")
	}
}
