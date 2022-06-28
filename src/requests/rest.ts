import { Rest, RestRequestMethod } from "../typings/rest.d.ts";
import { createRequest } from "./request.ts";

/**
 * @param maxRequestsBeforeSweep  The maximum amount of requests to process before sweeping rest cache. Setting it too low may cause extra lag and too high lead to unnecessary memory usage. It's recommended to set somewhere between 100 and 5000 based on how many and how frequently you use it.
 */
export function createRest(token: string, maxRequestsBeforeSweep = 100): Rest {
  const rest: Rest = {
    token: token.startsWith("Bot") ? token : `Bot ${token}`,
    cooldownOffset: 2500,
    globalRequestLimit: 50,
    maxRequestsBeforeSweep: maxRequestsBeforeSweep,
    buckets: new Map<string, number>(),
    request: undefined!
  };

  rest.request = createRequest.bind(undefined, rest) as RestRequestMethod;
  return rest;
}
