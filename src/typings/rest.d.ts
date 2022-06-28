export type HTTPMethod = "GET" | "PUT" | "POST" | "HEAD" | "PATCH" | "TRACE" | "DELETE" | "OPTIONS" | "CONNECT";
export type RestRequestMethod = (method: HTTPMethod, route: string, content?: Record<string, unknown> | Record<string, unknown>[], skipResponse?: boolean) => Promise<Record<string, unknown>>;

export interface Rest {
  token: string;
  /**
   * Extra time offset (in ms) to wait for API cooldown.
   * Set it above 2.5s (2500) if you experience rate limit problems.
   * @default 2500
   */
  cooldownOffset: number;
  /**
   * Number of requests your app/bot can process per second before it gets rate limited by Discord API.
   * By default it's 50 but big bots (over 250k guilds) commonly receive increased limit.
   * @default 50
   */
  globalRequestLimit: number;
  /**
   * The maximum amount of commands to execute before sweeping cooldown cache. Setting it too low may cause extra lag and too high lead to unnecessary memory usage. It's recommended to set somewhere between 100 and 5000 based on how frequently your application is used.
   * @default 100
   */
  maxRequestsBeforeSweep: number;
  /** Contains all bucket hashes (received from Discord API) that are waiting on cooldown. Mapped with timestamp after which bucket will return from cooldown. */
  buckets: Map<string, number>;
  request: RestRequestMethod;
}
