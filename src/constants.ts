import { Schema } from "../deps.ts";

export const CDN_URL = "https://cdn.discordapp.com";
export const REST_URL = "https://discord.com/api/v10";

export const COMMAND_SCHEMA: Schema = {
  name: {
    type: "string",
    min: 3,
    max: 32,
    match: /^[-_\p{L}\p{N}\p{sc=Deva}\p{sc=Thai}]{1,32}$/,
    filter: (arg) => arg.toLowerCase() === arg && !arg.includes(" "),
    required: true
  },
  description: {
    type: "string",
    max: 100,
    required: true
  },
  options: {
    type: "array",
    elementType: {
      type: "object",
      records: {
        type: {
          type: "string",
          match: /(string|int|boolean|user|channel|role|float)/,
          required: true
        },
        name: {
          type: "string",
          min: 3,
          max: 32,
          match: /^[-_\p{L}\p{N}\p{sc=Deva}\p{sc=Thai}]{1,32}$/,
          filter: (arg) => arg.toLowerCase() === arg && !arg.includes(" "),
          required: true
        },
        description: {
          type: "string",
          max: 100,
          required: true
        },
        required: {
          type: "boolean"
        },
        channelTypes: {
          type: "array",
          elementType: {
            type: "string",
            match: /(normal|category|crosspost)/
          },
          min: 1 // Require at least 1 if already defined.
        },
        minValue: {
          type: "float"
        },
        maxValue: {
          type: "float"
        },
        autoComplete: {
          type: "boolean"
        },
        choices: {
          type: "array",
          elementType: {
            type: "object",
            records: {
              name: {
                type: "string",
                max: 100,
                required: true
              },
              value: {
                type: "unknown",
                filter: (arg) => ["string", "number"].includes(typeof arg),
                required: true
              }
            }
          },
          max: 25
        }
      }
    },
    max: 25
  },
  autoComplete: {
    type: "function"
  },
  execute: {
    type: "function"
  }
};
