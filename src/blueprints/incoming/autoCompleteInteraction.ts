import { AutoCompleteInteraction } from "../../typings/interaction.d.ts";
import { CommandChoice } from "../../typings/command.d.ts";
import { RestRequestMethod } from "../../typings/rest.d.ts";

// Don't convert id into bigint because it has too short lifespan! (Not worth)
export function processAutoCompleteInteraction(payload: Record<string, any>, request: RestRequestMethod): AutoCompleteInteraction {
  let optionName!: string;
  let optionValue!: string | number;
  let subCommand!: string;
  let acknowledged = false;

  if (payload.data.options) {
    if (payload.data.options[0].options) subCommand = payload.data.options[0].name;
    const iter = subCommand ? payload.data.options[0].options : payload.data.options;

    for (const option of iter) {
      if (option.focused) {
        optionName = option.name;
        optionValue = option.value;
        break;
      }
    }
  }

  return {
    command: payload.data.name,
    subCommand: subCommand,
    option: optionName,
    value: optionValue,
    async suggest(choices: CommandChoice[]) {
      if (acknowledged) return;
      acknowledged = true;
      await request("POST", `/interactions/${payload.id}/${payload.token}/callback`, { type: 8, data: { choices } }, true);
    }
  };
}
