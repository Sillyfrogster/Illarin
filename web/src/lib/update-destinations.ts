import type { AssetUpdateDestination } from "@/lib/api/asset-destinations";
import { readableDate } from "@/lib/dates";

export function destinationRotating(one: AssetUpdateDestination): boolean {
  return Boolean(
    one.previousSecretUntil && new Date(one.previousSecretUntil) > new Date(),
  );
}

export function destinationStanding(one: AssetUpdateDestination): string {
  if (destinationRotating(one) && one.previousSecretUntil) {
    return `Both signing secrets are accepted until ${readableDate(one.previousSecretUntil)}.`;
  }
  if (one.state === "disabled") {
    return "Switched off. Verify it again to announce here.";
  }
  if (one.state !== "active" || !one.verifiedAt) {
    return one.kind === "discord"
      ? "Discord has not confirmed this channel yet."
      : "Your endpoint has to answer a signed challenge before Illarin sends to it.";
  }
  if (one.kind === "discord") {
    return `Discord confirmed the channel on ${readableDate(one.verifiedAt)}.`;
  }
  return `Answered the challenge on ${readableDate(one.verifiedAt)}.`;
}

export function destinationWhere(one: AssetUpdateDestination): string {
  if (one.kind === "discord") return "Discord channel";
  return one.host;
}
