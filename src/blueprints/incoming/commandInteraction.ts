import { CommandInteraction } from "../../typings/interaction.d.ts";
import { Content } from "../../typings/message.d.ts";
import { RestRequestMethod } from "../../typings/rest.d.ts";
import { processContent } from "../outgoing/message.ts";
import { processMember } from "./member.ts";
import { processUser } from "./user.ts";

// Don't convert id into bigint because it has too short lifespan! (Not worth)
export function processCommandInteraction(payload: Record<string, any>, applicationId: bigint, request: RestRequestMethod): CommandInteraction {
  const options: Record<string, string | number> = {};
  let subCommand!: string;
  let acknowledged = false;

  if (payload.data.options) {
    if (payload.data.options[0].type == 1) {
      subCommand = payload.data.options[0].name;
      for (const opt of payload.data.options[0].options) options[opt.name] = opt.value;
    } else {
      for (const opt of payload.data.options) options[opt.name] = opt.value;
    }
  }

  return {
    secret: payload.id,
    channelId: BigInt(payload.channel_id),
    guildId: payload.guild_id && BigInt(payload.guild_id),
    target: payload.guild_id ? processMember(payload.member, payload.guild_id) : processUser(payload.user),
    command: payload.data.name,
    subCommand: subCommand,
    options: options,
    async defer(ephemeral?: boolean) {
      if (acknowledged) return;
      acknowledged = true;
      await request("POST", `/interactions/${payload.id}/${payload.token}/callback`, { type: 5, data: { flags: ephemeral ? 64 : 0 } });
    },
    async sendReply(content: Content, ephemeral?: boolean) {
      const data = processContent(content);
      if (!data.flags && ephemeral) data.flags = 64;

      if (acknowledged) {
        data.wait = true;
        await request("POST", `/webhooks/${applicationId}/${payload.token}`, data, true);
      } else {
        acknowledged = true;
        await request("POST", `/interactions/${payload.id}/${payload.token}/callback`, { type: 4, data }, true);
      }
    },
    async editReply(content: Content, ephemeral?: boolean) {
      if (!acknowledged) throw new Error("This interaction needs to be acknowledged first to edit it.");

      const data = processContent(content);
      if (!data.flags && ephemeral) data.flags = 64;
      await request("PATCH", `/webhooks/${applicationId}/${payload.token}/messages/@original`, data, true);
    },
    async deleteReply() {
      if (!acknowledged) throw new Error("This interaction needs to be acknowledged first to later delete it.");
      await request("DELETE", `/webhooks/${applicationId}/${payload.token}/messages/@original`, undefined, true);
    },
    async sendFollowUp(content: Content, ephemeral?: boolean) {
      if (!acknowledged) throw new Error("This interaction needs to be acknowledged first to send follow up message.");
      const data = processContent(content);
      data.wait = true;
      if (!data.flags && ephemeral) data.flags = 64;
      await request("POST", `/webhooks/${applicationId}/${payload.token}`, data, true);
    }
  };
}
