export type Target = Member | User;

export interface User {
  id: bigint;
  username: string;
  discriminator: bigint;
  isBot: boolean;
  /**
   * Flags which hold info about user badges.
   * Check [Discord docs](https://discord.com/developers/docs/game-sdk/users#data-models-userflag-enum) for more details.
   * */
  publicFlags: bigint;
  fetchAvatarUrl(): string | undefined;
  fetchBannerUrl(): string | undefined;
  fetchDynamicAvatarUrl(format: "png" | "jpg" | "gif" | "webp", size: 32 | 64 | 128 | 256 | 512 | 1024): string | undefined;
}

export interface Member extends User {
  /** A custom nickname set only for this specific guild. */
  guildNickname?: string;
  /** A timestamp in milliseconds since member started boosting the guild. */
  nitroSince?: bigint;
  roleIds: bigint[];
  /**
   * Flags which hold info about member permissions.
   * Check [this calculator](https://finitereality.github.io/permissions-calculator/?v=9) for help.
   * For example, check for admin permissions:
   * @example if ((ctx.target.permissionFlags & 8n) !== 8n)
   */
  permissionFlags: bigint;
  /** A timestamp in milliseconds since member joined into guild. */
  memberSince: bigint;
  guildId: bigint;
  fetchGuildAvatarUrl(): string | undefined;
}
