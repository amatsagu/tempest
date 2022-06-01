import { REST_URL } from "../constants.ts";
import { HTTPMethod, Rest } from "../typings/rest.d.ts";

let reqs = 0;
let resetAt = 0;
let timeOffset = 0;

export async function createRequest(rest: Rest, method: HTTPMethod, route: string, content?: Record<string, unknown>, skipResponse?: boolean): Promise<Record<string, unknown>> {
  if (timeOffset > 0) {
    return new Promise<Promise<Record<string, unknown>>>(function (resolve) {
      console.log(`[REST] Awaiting route: ${route}`);
      setTimeout(function () {
        timeOffset = 0;
        resolve(createRequest(rest, method, route, content, skipResponse));
      }, timeOffset);
    });
  }

  const res = await fetch(REST_URL + route, {
    method,
    headers: {
      "User-Agent": "DiscordApp https://github.com/Amatsagu/tempest",
      "Content-Type": "application/json",
      Authorization: rest.token
    },
    body: content && JSON.stringify(content)
  });

  const bucketHash = res.headers.get("x-ratelimit-bucket");
  const bucketResetAt = bucketHash ? rest.buckets.get(bucketHash) : undefined;
  const currentTime = new Date().valueOf() / 1000;

  if (res.status == 429 || (bucketResetAt && bucketResetAt > currentTime)) timeOffset += ~~res.headers.get("x-ratelimit-reset-after")! * 1000 + rest.cooldownOffset;
  if (res.headers.get("X-RateLimit-Global")) {
    console.log(`[REST (GLOBAL)] Hit global request limit for bucket: ${bucketHash} (${~~res.headers.get("x-ratelimit-reset-after")!}s) at ${route}`);
    console.log(await res.text());
    Deno.exit(1);
  }

  reqs++;
  if (reqs >= rest.globalRequestLimit) {
    console.log(`[REST] Hit max requests per second limit!`);
    if (currentTime < resetAt) {
      timeOffset += 4500 + rest.cooldownOffset;
      reqs = 0;
    } else resetAt = currentTime + 4500 + rest.cooldownOffset;
  }

  const remaining = res.headers.get("x-ratelimit-remaining");
  if (remaining && ~~remaining == 0 && bucketHash) {
    console.log(`[REST] Hit request limit for bucket: ${bucketHash} (${~~res.headers.get("x-ratelimit-reset-after")!}s) at ${route}`);
    rest.buckets.set(bucketHash, ~~res.headers.get("x-ratelimit-reset")!);
  }

  // @ts-ignore Return nothing as requested.
  if (skipResponse || res.status == 204) return;
  return res.json();
}
