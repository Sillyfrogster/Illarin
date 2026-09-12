import { type Answer, ask } from "./request";
import type { components } from "./schema";

export type AssetUpdateDestination =
  components["schemas"]["AssetUpdateDestination"];
export type AssetUpdateDestinationChoice =
  components["schemas"]["AssetUpdateDestinationChoice"];
export type AssetUpdateAnnouncement =
  components["schemas"]["AssetUpdateAnnouncement"];
type AddedDestination = components["schemas"]["AddedAssetUpdateDestination"];
type NewDestination = components["schemas"]["AddAssetUpdateDestinationRequest"];
type DestinationChange =
  components["schemas"]["UpdateAssetUpdateDestinationRequest"];

const base = "/account/update-destinations";

async function request<T>(
  path: string,
  method: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<Answer<T>> {
  try {
    return await ask<T>(
      path,
      {
        method,
        signal,
        headers: body ? { "Content-Type": "application/json" } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      },
      (response) => response.json() as Promise<T>,
    );
  } catch {
    return { error: "We could not read Illarin's response. Try again." };
  }
}

export function readUpdateDestinations(signal?: AbortSignal) {
  return request<{ destinations: AssetUpdateDestination[] }>(
    base,
    "GET",
    undefined,
    signal,
  );
}

export function addUpdateDestination(body: NewDestination) {
  return request<AddedDestination>(base, "POST", body);
}

export function changeUpdateDestination(id: string, body: DestinationChange) {
  return request<AssetUpdateDestination>(`${base}/${id}`, "PATCH", body);
}

export function verifyUpdateDestination(id: string) {
  return request<AssetUpdateDestination>(`${base}/${id}/verification`, "POST");
}

export function disableUpdateDestination(id: string) {
  return request<AssetUpdateDestination>(
    `${base}/${id}/verification`,
    "DELETE",
  );
}

export function rotateUpdateDestinationSecret(id: string) {
  return request<AddedDestination>(`${base}/${id}/secret`, "POST");
}

export function removeUpdateDestination(id: string) {
  return ask<null>(`${base}/${id}`, { method: "DELETE" }, async () => null);
}

export function readAssetUpdateDestinationChoices(
  assetId: string,
  signal?: AbortSignal,
) {
  return request<{ destinations: AssetUpdateDestinationChoice[] }>(
    `/assets/${assetId}/update-destinations`,
    "GET",
    undefined,
    signal,
  );
}

export function readAssetUpdateAnnouncements(
  assetId: string,
  signal?: AbortSignal,
) {
  return request<{ announcements: AssetUpdateAnnouncement[] }>(
    `/assets/${assetId}/announcements`,
    "GET",
    undefined,
    signal,
  );
}
