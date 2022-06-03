import { ButtonInteraction } from "../../typings/interaction.d.ts";
import { RestRequestMethod } from "../../typings/rest.d.ts";
import { processMember } from "./member.ts";
import { processUser } from "./user.ts";

// Don't convert id into bigint because it has too short lifespan! (Not worth)
export function processButtonInteraction(payload: Record<string, any>, request: RestRequestMethod): ButtonInteraction {
  let acknowledged = false;

  return {
    id: payload.data.custom_id,
    target: payload.guild_id ? processMember(payload.member, payload.guild_id) : processUser(payload.user),
    async acknowledge() {
      if (acknowledged) return;
      acknowledged = true;
      await request("POST", `/interactions/${payload.id}/${payload.token}/callback`, { type: 6 }, true);
    }
  };
}
