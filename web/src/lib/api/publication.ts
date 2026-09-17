import type {
  AddedPublicationDestination,
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
  PublicationCategory,
  PublicationDestination,
  PublicationEvent,
  PublicationGrant,
  PublicationWorkspace,
  RotatedPublicationSecret,
} from "@/lib/api/query";
import { ask } from "./request";

export type DiscordRepair = {
  requestId: string;
  action: "edit" | "delete" | "correction";
  messageId: string;
  text: string;
};

export type DiscordRepairResult = {
  action: string;
  messageId: string;
  state: "completed" | "refused" | "unconfirmed";
  detail: string;
};

export function repairDiscordAnnouncement(id: string, repair: DiscordRepair) {
  return ask<DiscordRepairResult>(
    "POST",
    `/publication/deliveries/${id}/repair`,
    { body: repair },
  );
}

export function readCategories() {
  return ask<{ categories: PublicationCategory[] }>(
    "GET",
    "/publication/categories",
  );
}

export function updateCategory(
  id: string,
  change: { label?: string; retired?: boolean },
) {
  return ask<PublicationCategory>("PATCH", `/publication/categories/${id}`, {
    body: change,
  });
}

export function orderCategories(categoryIds: string[]) {
  return ask<{ categories: PublicationCategory[] }>(
    "PUT",
    "/publication/categories",
    { body: { categoryIds } },
  );
}

export function readGrants() {
  return ask<{ grants: PublicationGrant[] }>("GET", "/publication/grants");
}

export function approveContributor(approval: {
  handle: string;
  appId: string;
  categoryIds: string[];
  defaultCategoryId: string;
}) {
  return ask<PublicationGrant>("POST", "/publication/grants", {
    body: approval,
  });
}

export function updateGrant(
  id: string,
  change: { categoryIds?: string[]; defaultCategoryId?: string },
) {
  return ask<PublicationGrant>("PATCH", `/publication/grants/${id}`, {
    body: change,
  });
}

export function revokeGrant(id: string) {
  return ask<void>("DELETE", `/publication/grants/${id}`);
}

export function readDestinations() {
  return ask<{ destinations: PublicationDestination[] }>(
    "GET",
    "/publication/destinations",
  );
}

export function addDestination(endpoint: {
  name: string;
  address: string;
  events: PublicationEvent[];
}) {
  return ask<AddedPublicationDestination>("POST", "/publication/destinations", {
    body: endpoint,
  });
}

export function updateDestination(
  id: string,
  change: { name?: string; address?: string; events?: PublicationEvent[] },
) {
  return ask<PublicationDestination>(
    "PATCH",
    `/publication/destinations/${id}`,
    { body: change },
  );
}

export function addChannel(channel: {
  name: string;
  address: string;
  roleId: string;
  roleName: string;
}) {
  return ask<PublicationDestination>("POST", "/publication/channels", {
    body: channel,
  });
}

export function updateChannel(
  id: string,
  change: { name: string; address?: string; roleId: string; roleName: string },
) {
  return ask<PublicationDestination>("PATCH", `/publication/channels/${id}`, {
    body: change,
  });
}

export function verifyDestination(id: string) {
  return ask<PublicationDestination>(
    "POST",
    `/publication/destinations/${id}/verification`,
  );
}

export function disableDestination(id: string) {
  return ask<PublicationDestination>(
    "DELETE",
    `/publication/destinations/${id}/verification`,
  );
}

export function rotateDestinationSecret(id: string) {
  return ask<RotatedPublicationSecret>(
    "POST",
    `/publication/destinations/${id}/secret`,
  );
}

export function readDeliveries(state?: PostDeliveryState) {
  const narrowed = state ? `?state=${state}` : "";
  return ask<{ deliveries: PostDelivery[] }>(
    "GET",
    `/publication/deliveries${narrowed}`,
  );
}

export function readDeliveryAttempts(id: string) {
  return ask<{ attempts: PostDeliveryAttempt[] }>(
    "GET",
    `/publication/deliveries/${id}/attempts`,
  );
}

export function replayDelivery(id: string) {
  return ask<PostDelivery>("POST", `/publication/deliveries/${id}/replay`);
}

export function removeDestination(id: string) {
  return ask<void>("DELETE", `/publication/destinations/${id}`);
}

export function setGrantDestinations(
  grantId: string,
  policy: {
    destinationIds: string[] | null;
    defaultDestinationIds: string[];
  },
) {
  return ask<PublicationGrant>(
    "PUT",
    `/publication/grants/${grantId}/destinations`,
    { body: policy },
  );
}

export function readWorkspace() {
  return ask<PublicationWorkspace>("GET", "/publication/workspace");
}
