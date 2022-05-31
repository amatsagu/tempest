export type HTTPMethod = "GET" | "PUT" | "POST" | "HEAD" | "PATCH" | "TRACE" | "DELETE" | "OPTIONS" | "CONNECT";

export interface Rest {
  token: string;
  cooldownOffset: number;
  /** Contains all bucket hashes (received from Discord API) that are waiting on cooldown. Mapped with timestamp after which bucket will return from cooldown. */
  buckets: Map<string, number>;
  request: (method: HTTPMethod, route: string, content?: Record<string, unknown>, skipResponse?: boolean) => Promise<Record<string, unknown>>;
}
