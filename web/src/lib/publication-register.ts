import type {
  PostDelivery,
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationGrant,
  PublicationToken,
} from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import { deliveryState } from "@/lib/delivery-standing";
import { EVENT_WORDS } from "@/lib/publication-delivery";

export type Register =
  | "contributors"
  | "apps"
  | "categories"
  | "destinations"
  | "deliveries";

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

export type RegisterStanding = { count: number | null; attention: boolean };

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

export function nothingIn(
  register: Register,
  held: { apps: PublicationApp[] },
): string {
  if (register === "contributors") {
    return held.apps.filter((one) => !one.retired).length === 0
      ? "Add an app first. An approval binds one person to one app."
      : "No app contributors approved. Team publishing remains available.";
  }
  if (register === "apps") {
    return "No apps configured. Add an app before approving a contributor.";
  }
  if (register === "categories") {
    return "The blog has no categories, so no post can be filed.";
  }
  if (register === "destinations") {
    return "No announcement destinations configured. Posts can still be published.";
  }
  return "No announcements sent yet.";
}

export function nothingDelivered(view: string): string {
  if (view === "failed") return "No failed announcements.";
  if (view === "pending") return "No pending announcements.";
  if (view === "delivered") return "No delivered announcements.";
  if (view === "unconfirmed") {
    return "No unconfirmed Discord announcements.";
  }
  return "No announcements sent yet.";
}

export function grantAllowance(grant: PublicationGrant): string {
  const app = grant.app.retired
    ? `${grant.app.name} (retired)`
    : grant.app.name;
  const categories = grant.categories.map((one) => one.label).join(", ");
  return `${app} · ${categories} · ${grant.defaultCategory.label} by default`;
}

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
    return "Endpoint verification required before sending announcements.";
  }
  if (one.channel) {
    return `Announcing as ${one.channel.webhookName || "the name Discord gives it"}. Discord confirmed the channel on ${readableDate(one.verifiedAt)}.`;
  }
  return `Verified on ${readableDate(one.verifiedAt)}.`;
}

export function destinationTakes(one: PublicationDestination): string {
  if (one.channel) {
    const role = one.channel.roleName;
    return role ? `First publication · @${role}` : "First publication";
  }
  if (one.events.length === 0) return "No events selected.";
  return one.events.map((event) => EVENT_WORDS[event].word).join(" · ");
}

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

export function canReplay(one: PostDelivery): boolean {
  return deliveryState(one) === "gaveUp" && !one.removed;
}

export function tokenStanding(one: PublicationToken): string {
  const said = [`Created ${readableDate(one.createdAt)}`];
  said.push(
    one.lastUsedAt ? `last used ${readableDate(one.lastUsedAt)}` : "never used",
  );
  if (one.expiresAt && one.active) {
    said.push(`expires ${readableDate(one.expiresAt)}`);
  }
  return said.join(" · ");
}

export function tokenEnded(one: PublicationToken): string {
  if (one.revokedAt) return `Revoked ${readableDate(one.revokedAt)}`;
  if (one.expiresAt) return `Expired ${readableDate(one.expiresAt)}`;
  return "Inactive";
}
