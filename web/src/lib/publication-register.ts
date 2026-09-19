import type {
  BlogAnnouncementAttempt,
  BlogIntegration,
  PublicationCategory,
  PublicationGrant,
} from "@/lib/api/query";
import { attemptState } from "@/lib/attempt-standing";
import { ANNOUNCEMENT_WORDS } from "@/lib/blog-announcement-attempt";
import { readableDate } from "@/lib/dates";

export type Register =
  | "contributors"
  | "categories"
  | "integrations"
  | "attempts";

export const REGISTERS: Register[] = [
  "contributors",
  "categories",
  "integrations",
  "attempts",
];

const NAMES: Record<Register, string> = {
  contributors: "Contributors",
  categories: "Categories",
  integrations: "Integrations",
  attempts: "Announcements",
};

export function registerName(register: Register): string {
  return NAMES[register];
}

export type RegisterStanding = { count: number | null; attention: boolean };

export function registerStandings(held: {
  grants: PublicationGrant[];
  categories: PublicationCategory[];
  integrations: BlogIntegration[];
  stopped: number;
}): Record<Register, RegisterStanding> {
  return {
    contributors: kept(held.grants.filter((one) => one.active).length),
    categories: kept(held.categories.filter((one) => !one.retired).length),
    integrations: kept(held.integrations.length),
    attempts: {
      attention: held.stopped > 0,
      count: held.stopped > 0 ? held.stopped : null,
    },
  };
}

function kept(count: number): RegisterStanding {
  return { attention: false, count };
}

export function nothingIn(register: Register): string {
  if (register === "contributors") {
    return "No app contributors approved. Team publishing remains available.";
  }
  if (register === "categories") {
    return "The blog has no categories, so no post can be filed.";
  }
  if (register === "integrations") {
    return "No announcement integrations configured. Posts can still be published.";
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

export function integrationStanding(one: BlogIntegration): string {
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

export function integrationTakes(one: BlogIntegration): string {
  if (one.channel) {
    const role = one.channel.roleName;
    return role ? `First publication · @${role}` : "First publication";
  }
  if (one.announcements.length === 0) return "No announcements chosen.";
  return one.announcements
    .map((event) => ANNOUNCEMENT_WORDS[event].word)
    .join(" · ");
}

export function integrationActions(one: BlogIntegration): {
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

export function canReplay(one: BlogAnnouncementAttempt): boolean {
  return attemptState(one) === "gaveUp" && !one.removed;
}
