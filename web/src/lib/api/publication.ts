import type {
  AddedPublicationDestination,
  IssuedPublicationToken,
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationEvent,
  PublicationGrant,
  PublicationToken,
  PublicationWorkspace,
  RotatedPublicationSecret,
} from "@/lib/api/query";
import { ask, json } from "./distinctions";

export function readApps() {
  return json<{ apps: PublicationApp[] }>("/publication/apps", "GET");
}

export function configureApp(slug: string, name: string, home: string) {
  return json<PublicationApp>("/publication/apps", "POST", {
    slug,
    name,
    home,
  });
}

export function updateApp(
  id: string,
  change: { slug?: string; name?: string; home?: string; retired?: boolean },
) {
  return json<PublicationApp>(`/publication/apps/${id}`, "PATCH", change);
}

export function orderApps(appIds: string[]) {
  return json<{ apps: PublicationApp[] }>("/publication/apps", "PUT", {
    appIds,
  });
}

export function uploadAppMark(id: string, file: File) {
  const body = new FormData();
  body.append("file", file);
  return ask<PublicationApp>(
    `/publication/apps/${id}/mark`,
    { method: "PUT", body },
    (response) => response.json() as Promise<PublicationApp>,
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

export function setAppDestinations(
  appId: string,
  policy: { destinationIds: string[]; defaultDestinationIds: string[] },
) {
  return json<PublicationApp>(
    `/publication/apps/${appId}/destinations`,
    "PUT",
    policy,
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

export function readTokens(grantId: string) {
  return json<{ tokens: PublicationToken[] }>(
    `/publication/grants/${grantId}/tokens`,
    "GET",
  );
}

export function issueToken(
  grantId: string,
  token: { name: string; expiresAt?: string },
) {
  return json<IssuedPublicationToken>(
    `/publication/grants/${grantId}/tokens`,
    "POST",
    token,
  );
}

export function revokeToken(id: string) {
  return ask<null>(
    `/publication/tokens/${id}`,
    { method: "DELETE" },
    async () => null,
  );
}
