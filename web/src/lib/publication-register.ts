import type {
  PostDelivery,
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationGrant,
  PublicationToken,
} from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import { deliveryState, EVENT_WORDS } from "@/lib/publication-delivery";

/** One of the things the publication authority keeps, and the page shows one at a time. */
export type Register =
  | "contributors"
  | "apps"
  | "categories"
  | "destinations"
  | "deliveries";

/** The order the registers are offered in, everywhere they are offered. */
export const REGISTERS: Register[] = [
  "contributors",
  "apps",
  "categories",
  "destinations",
  "deliveries",
];

const NAMES: Record<Register, string> = {
  contributors: "Contributors",
  apps: "Apps",
  categories: "Categories",
  destinations: "Destinations",
  deliveries: "Announcements",
};

export function registerName(register: Register): string {
  return NAMES[register];
}

/** What a register carries, and whether that number is asking to be looked at. */
export type RegisterStanding = { count: number | null; attention: boolean };

/** Counts what each register still uses; announcements count only what stopped short. */
export function registerStandings(held: {
  grants: PublicationGrant[];
  apps: PublicationApp[];
  categories: PublicationCategory[];
  destinations: PublicationDestination[];
  stopped: number;
}): Record<Register, RegisterStanding> {
  return {
    contributors: kept(held.grants.filter((one) => one.active).length),
    apps: kept(held.apps.filter((one) => !one.retired).length),
    categories: kept(held.categories.filter((one) => !one.retired).length),
    destinations: kept(held.destinations.length),
    deliveries: {
      attention: held.stopped > 0,
      count: held.stopped > 0 ? held.stopped : null,
    },
  };
}

function kept(count: number): RegisterStanding {
  return { attention: false, count };
}

/** What an empty register says, which is what to do about it rather than that it is empty. */
export function nothingIn(
  register: Register,
  held: { apps: PublicationApp[] },
): string {
  if (register === "contributors") {
    return held.apps.filter((one) => !one.retired).length === 0
      ? "Add an app first. An approval binds one person to one app."
      : "Nobody is approved to publish. Illarin's own writing still works.";
  }
  if (register === "apps") {
    return "No app is configured, so nobody can be approved yet.";
  }
  if (register === "categories") {
    return "The blog has no categories, so no post can be filed.";
  }
  if (register === "destinations") {
    return "Nowhere is set up to receive an announcement, so every post publishes quietly.";
  }
  return "No post has announced anywhere yet.";
}

/** What an empty delivery listing says, which depends on what was looked for. */
export function nothingDelivered(view: string): string {
  if (view === "failed") return "Nothing has stopped short.";
  if (view === "pending") return "Nothing is on its way.";
  if (view === "delivered") return "Nothing has arrived yet.";
  if (view === "unconfirmed") {
    return "Every announcement Discord took, it confirmed.";
  }
  return "No post has announced anywhere yet.";
}

/** One line naming the app a contributor writes for and what they may file it as. */
export function grantAllowance(grant: PublicationGrant): string {
  const app = grant.app.retired
    ? `${grant.app.name} (retired)`
    : grant.app.name;
  const categories = grant.categories.map((one) => one.label).join(", ");
  return `${app} · ${categories} · ${grant.defaultCategory.label} by default`;
}

/** One line saying what a destination is doing and since when. */
export function destinationStanding(one: PublicationDestination): string {
  if (
    one.previousSecretUntil &&
    new Date(one.previousSecretUntil) > new Date()
  ) {
    return `Both signing secrets are accepted until ${readableDate(one.previousSecretUntil)}.`;
  }
  if (one.state === "disabled") {
    return "Switched off. Verify it again to start sending here.";
  }
  if (one.state !== "active" || !one.verifiedAt) {
    return "Waiting to prove it is listening. Nothing is sent until it does.";
  }
  if (one.channel) {
    return `Announcing as ${one.channel.webhookName || "the name Discord gives it"}. Discord confirmed the channel on ${readableDate(one.verifiedAt)}.`;
  }
  return `Receiving. Proved it was listening on ${readableDate(one.verifiedAt)}.`;
}

/** What a destination receives, and the role it may mention. */
export function destinationTakes(one: PublicationDestination): string {
  if (one.channel) {
    const role = one.channel.roleName;
    return role ? `First publication · @${role}` : "First publication";
  }
  if (one.events.length === 0) return "Takes nothing.";
  return one.events.map((event) => EVENT_WORDS[event].word).join(" · ");
}

/** Which of the three standing changes a destination is open to right now. */
export function destinationActions(one: PublicationDestination): {
  rotate: boolean;
  switchOff: boolean;
  verify: boolean;
} {
  return {
    rotate: !one.channel,
    switchOff: one.state === "active",
    verify: one.state !== "active",
  };
}

/** Only an announcement Illarin gave up on, to a destination that still exists, can be sent again. */
export function canReplay(one: PostDelivery): boolean {
  return deliveryState(one) === "gaveUp" && !one.removed;
}

/** What one token has done, in the order that matters when you are choosing one to revoke. */
export function tokenStanding(one: PublicationToken): string {
  const said = [`Made ${readableDate(one.createdAt)}`];
  said.push(
    one.lastUsedAt ? `last used ${readableDate(one.lastUsedAt)}` : "never used",
  );
  if (one.expiresAt && one.active) {
    said.push(`expires ${readableDate(one.expiresAt)}`);
  }
  return said.join(" · ");
}

/** How a token that no longer works ended. */
export function tokenEnded(one: PublicationToken): string {
  if (one.revokedAt) return `Revoked ${readableDate(one.revokedAt)}`;
  if (one.expiresAt) return `Expired ${readableDate(one.expiresAt)}`;
  return "Spent";
}
