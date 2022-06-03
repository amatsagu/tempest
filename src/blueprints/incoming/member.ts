import { Member } from "../../typings/target.d.ts";
import { CDN_URL } from "../../constants.ts";
import { bigIntToHash, hashToBigInt } from "../../util.ts";

export function processMember(payload: Record<string, any>, guildId: string | bigint): Member {
  const id = BigInt(payload.user.id);
  const avatar = payload.user.avatar && hashToBigInt(payload.user.avatar);
  const banner = payload.user.banner && hashToBigInt(payload.user.banner);
  const guildAvatar = payload.avatar && hashToBigInt(payload.avatar);

  return {
    id: id,
    username: payload.user.username,
    discriminator: payload.user.discriminator,
    isBot: !!payload.user.bot,
    publicFlags: BigInt(payload.user.public_flags ?? 0),
    guildNickname: payload.nick,
    nitroSince: payload.premium_since && BigInt(Date.parse(payload.premium_since)),
    roleIds: payload.roles?.map((role: number) => BigInt(role)),
    permissionFlags: BigInt(payload.permissions),
    memberSince: payload.joined_at && BigInt(Date.parse(payload.joined_at)),
    guildId: BigInt(guildId),
    fetchAvatarUrl() {
      const hash = bigIntToHash(avatar);
      return hash && `${CDN_URL}/avatars/${id}/${hash}${hash[0] == "a" ? ".gif" : ""}`;
    },
    fetchBannerUrl() {
      const hash = bigIntToHash(banner);
      return hash && `${CDN_URL}/banners/${id}/${hash}${hash[0] == "a" ? ".gif" : ""}`;
    },
    fetchGuildAvatarUrl() {
      const hash = bigIntToHash(guildAvatar);
      return hash && `${CDN_URL}/guilds/${this.guildId}/users/${id}/avatars/${hash}${hash[0] == "a" ? ".gif" : ""}`;
    },
    fetchDynamicAvatarUrl(format: "png" | "jpg" | "gif" | "webp", size: 32 | 64 | 128 | 256 | 512 | 1024) {
      return avatar && `${CDN_URL}/avatars/${id}/${bigIntToHash(avatar)}.${format}?size=${size}`;
    }
  };
}
