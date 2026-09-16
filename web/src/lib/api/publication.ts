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
import { ask, json } from "./request";

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
  return json<DiscordRepairResult>(
    `/publication/deliveries/${id}/repair`,
    "POST",
    repair,
  );
}

export function readCategories() {
  return json<{ categories: PublicationCategory[] }>(
    "/publication/categories",
    "GET",
  );
}

export function updateCategory(
  id: string,
  change: { label?: string; retired?: boolean },
) {
  return json<PublicationCategory>(
    `/publication/categories/${id}`,
    "PATCH",
    change,
  );
}

export function orderCategories(categoryIds: string[]) {
  return json<{ categories: PublicationCategory[] }>(
    "/publication/categories",
    "PUT",
    { categoryIds },
  );
}

export function readGrants() {
  return json<{ grants: PublicationGrant[] }>("/publication/grants", "GET");
}

export function approveContributor(approval: {
  handle: string;
  appId: string;
  categoryIds: string[];
  defaultCategoryId: string;
}) {
  return json<PublicationGrant>("/publication/grants", "POST", approval);
}

export function updateGrant(
  id: string,
  change: { categoryIds?: string[]; defaultCategoryId?: string },
) {
  return json<PublicationGrant>(`/publication/grants/${id}`, "PATCH", change);
}

export function revokeGrant(id: string) {
  return ask<null>(
    `/publication/grants/${id}`,
    { method: "DELETE" },
    async () => null,
  );
}

export function readDestinations() {
  return json<{ destinations: PublicationDestination[] }>(
    "/publication/destinations",
    "GET",
  );
}

export function addDestination(endpoint: {
  name: string;
  address: string;
  events: PublicationEvent[];
}) {
  return json<AddedPublicationDestination>(
    "/publication/destinations",
    "POST",
    endpoint,
  );
}

export function updateDestination(
  id: string,
  change: { name?: string; address?: string; events?: PublicationEvent[] },
) {
  return json<PublicationDestination>(
    `/publication/destinations/${id}`,
    "PATCH",
    change,
  );
}

export function addChannel(channel: {
  name: string;
  address: string;
  roleId: string;
  roleName: string;
}) {
  return json<PublicationDestination>("/publication/channels", "POST", channel);
}

export function updateChannel(
  id: string,
  change: { name: string; address?: string; roleId: string; roleName: string },
) {
  return json<PublicationDestination>(
    `/publication/channels/${id}`,
    "PATCH",
    change,
  );
}

export function verifyDestination(id: string) {
  return json<PublicationDestination>(
    `/publication/destinations/${id}/verification`,
    "POST",
  );
}

export function disableDestination(id: string) {
  return json<PublicationDestination>(
    `/publication/destinations/${id}/verification`,
    "DELETE",
  );
}

export function rotateDestinationSecret(id: string) {
  return json<RotatedPublicationSecret>(
    `/publication/destinations/${id}/secret`,
    "POST",
  );
}

export function readDeliveries(state?: PostDeliveryState) {
  const narrowed = state ? `?state=${state}` : "";
  return json<{ deliveries: PostDelivery[] }>(
    `/publication/deliveries${narrowed}`,
    "GET",
  );
}

export function readDeliveryAttempts(id: string) {
  return json<{ attempts: PostDeliveryAttempt[] }>(
    `/publication/deliveries/${id}/attempts`,
    "GET",
  );
}

export function replayDelivery(id: string) {
  return json<PostDelivery>(`/publication/deliveries/${id}/replay`, "POST");
}

export function removeDestination(id: string) {
  return ask<null>(
    `/publication/destinations/${id}`,
    { method: "DELETE" },
    async () => null,
  );
}

export function setGrantDestinations(
  grantId: string,
  policy: {
    destinationIds: string[] | null;
    defaultDestinationIds: string[];
  },
) {
  return json<PublicationGrant>(
    `/publication/grants/${grantId}/destinations`,
    "PUT",
    policy,
  );
}

export function readWorkspace() {
  return json<PublicationWorkspace>("/publication/workspace", "GET");
}
