import { User } from "../../typings/target.d.ts";
import { CDN_URL } from "../../constants.ts";
import { bigIntToHash, hashToBigInt } from "../../util.ts";

export function processUser(payload: Record<string, any>): User {
  const id = BigInt(payload.id);
  const avatar = payload.avatar && hashToBigInt(payload.avatar);
  const banner = payload.banner && hashToBigInt(payload.banner);

  return {
    id: id,
    username: payload.username,
    discriminator: BigInt(payload.discriminator),
    isBot: !!payload.bot,
    publicFlags: BigInt(payload.public_flags ?? 0),
    fetchAvatarUrl() {
      const hash = bigIntToHash(avatar);
      return hash && `${CDN_URL}/avatars/${id}/${hash}${hash[0] == "a" ? ".gif" : ""}`;
    },
    fetchBannerUrl() {
      const hash = bigIntToHash(banner);
      return hash && `${CDN_URL}/banners/${id}/${hash}${hash[0] == "a" ? ".gif" : ""}`;
    },
    fetchDynamicAvatarUrl(format: "png" | "jpg" | "gif" | "webp", size: 32 | 64 | 128 | 256 | 512 | 1024) {
      return avatar && `${CDN_URL}/avatars/${id}/${bigIntToHash(avatar)}.${format}?size=${size}`;
    }
  };
}
