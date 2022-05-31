import { REST_URL } from "../constants.ts";
import { HTTPMethod } from "../typings/rest.d.ts";

export function sendRequest(token: string) {
  return async function (method: HTTPMethod, route: string, content?: Record<string, unknown>, skipResponse?: boolean): Promise<Record<string, unknown> | undefined> {
    const res = await fetch(REST_URL + route, {
      method,
      headers: {
        "User-Agent": "DiscordApp https://github.com/Amatsagu/tempest",
        "Content-Type": "application/json",
        Authorization: token
      },
      body: content && JSON.stringify(content)
    });

    const localTime = new Date().valueOf() / 1000;
    const discordTime = ~~res.headers.get("x-ratelimit-reset")!;
    console.log(`Local: ${localTime}\nDiscord: ${discordTime}\nReset in: ${discordTime - localTime}s`);
    return await res.json();
  };
}
