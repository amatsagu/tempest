export type HTTPMethod = "GET" | "PUT" | "POST" | "HEAD" | "PATCH" | "TRACE" | "DELETE" | "OPTIONS" | "CONNECT";
export type RestRequestMethod = (method: HTTPMethod, route: string, content?: Record<string, unknown> | Record<string, unknown>[], skipResponse?: boolean) => Promise<Record<string, unknown>>;

export interface Rest {
  token: string;
  /**
   * Extra time offset (in ms) to wait for API cooldown.
   * Set it above 1s (1000) if you experience rate limit problems.
   * @default 1000
   */
  cooldownOffset: number;
  /**
   * Number of requests your app/bot can process per second before it gets rate limited by Discord API.
   * By default it's 50 but big bots (over 250k guilds) commonly receive increased limit.
   * @default 50
   */
  globalRequestLimit: number;
  /** Contains all bucket hashes (received from Discord API) that are waiting on cooldown. Mapped with timestamp after which bucket will return from cooldown. */
  buckets: Map<string, number>;
  request: RestRequestMethod;
}
