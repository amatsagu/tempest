import { Rest } from "../typings/rest.d.ts";
import { createRequest } from "./createRequest.ts";

export function createRest(token: string): Rest {
  const rest: Rest = {
    token: token.startsWith("Bot") ? token : `Bot ${token}`,
    cooldownOffset: 1000,
    globalRequestLimit: 50,
    buckets: new Map<string, number>(),
    request: undefined!
  };

  // Sweep old buckets from rest cache (30s).
  setInterval(function () {
    const currentTime = new Date().valueOf() / 1000;
    for (const bucket of rest.buckets.entries()) {
      if (currentTime > bucket[1]) rest.buckets.delete(bucket[0]);
    }
  }, 30000);

  rest.request = createRequest.bind(undefined, rest);
  return rest;
}
