import type {
  IssuedPublicationToken,
  PublicationApp,
  PublicationCategory,
  PublicationGrant,
  PublicationToken,
  PublicationWorkspace,
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
