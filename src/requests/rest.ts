import { Rest } from "../typings/rest.d.ts";
import { prepareRequest } from "./prepareRequest.ts";

export function createRest(token: string): Rest {
  const rest: Rest = {
    token: token.startsWith("Bot") ? token : `Bot ${token}`,
    /**
     * Extra time offset (in ms) to wait for API cooldown.
     * Set it above 1s (1000) if you experience rate limit problems.
     * @default 1000
     */
    cooldownOffset: 1000,
    buckets: new Map<string, number>(),
    request: undefined!
  };

  rest.request = prepareRequest.bind(undefined, rest);
  return rest;
}
