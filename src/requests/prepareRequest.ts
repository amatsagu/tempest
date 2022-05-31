import { REST_URL } from "../constants.ts";
import { HTTPMethod, Rest } from "../typings/rest.d.ts";

export async function prepareRequest(rest: Rest, method: HTTPMethod, route: string, content?: Record<string, unknown>, skipResponse?: boolean): Promise<Record<string, unknown>> {
  console.log(`PROCESSING: ${route}`);

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
  if (bucketResetAt) {
    const currentTime = new Date().valueOf() / 1000;
    if (bucketResetAt > currentTime) {
      await new Promise(function (resolve) {
        setTimeout(resolve, ~~res.headers.get("x-ratelimit-reset-after")! * 1000 + rest.cooldownOffset);
      });
      return prepareRequest(rest, method, route, content, skipResponse);
    }
  }

  const remaining = res.headers.get("x-ratelimit-remaining");
  if (remaining && ~~remaining == 0 && bucketHash) {
    console.log(`Hit request limit for bucket: ${bucketHash} (${~~res.headers.get("x-ratelimit-reset-after")!}s) at ${route}`);
    rest.buckets.set(bucketHash, ~~res.headers.get("x-ratelimit-reset")!);
  }

  return {};
}
