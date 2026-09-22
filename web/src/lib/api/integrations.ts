import { ask } from "./request";
import type { DiscordChannel } from "./shapes";

export type { DiscordChannel };

/** Whose Discord channel: the signed-in account's or the blog's. */
export type DiscordScope = "account" | "blog";

const path = (scope: DiscordScope) => `/${scope}/discord-channel`;

export function readDiscordChannel(scope: DiscordScope, signal?: AbortSignal) {
  return ask<DiscordChannel>("GET", path(scope), { signal });
}

export function connectDiscordChannel(scope: DiscordScope, address: string) {
  return ask<DiscordChannel>("PUT", path(scope), { body: { address } });
}

export function disconnectDiscordChannel(scope: DiscordScope) {
  return ask<DiscordChannel>("DELETE", path(scope));
}
