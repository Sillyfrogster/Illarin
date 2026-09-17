import { ask } from "./request";

import type {
  AddAssetUpdateDestinationRequest,
  AddedAssetUpdateDestination,
  AssetUpdateAnnouncement,
  AssetUpdateDestination,
  AssetUpdateDestinationChoice,
  UpdateAssetUpdateDestinationRequest,
} from "./shapes";
export type {
  AssetUpdateAnnouncement,
  AssetUpdateDestination,
  AssetUpdateDestinationChoice,
};

type AddedDestination = AddedAssetUpdateDestination;
type NewDestination = AddAssetUpdateDestinationRequest;
type DestinationChange = UpdateAssetUpdateDestinationRequest;

const base = "/account/update-destinations";

export function readUpdateDestinations(signal?: AbortSignal) {
  return ask<{ destinations: AssetUpdateDestination[] }>("GET", base, {
    signal,
  });
}

export function addUpdateDestination(body: NewDestination) {
  return ask<AddedDestination>("POST", base, { body });
}

export function changeUpdateDestination(id: string, body: DestinationChange) {
  return ask<AssetUpdateDestination>("PATCH", `${base}/${id}`, { body });
}

export function verifyUpdateDestination(id: string) {
  return ask<AssetUpdateDestination>("POST", `${base}/${id}/verification`);
}

export function disableUpdateDestination(id: string) {
  return ask<AssetUpdateDestination>("DELETE", `${base}/${id}/verification`);
}

export function rotateUpdateDestinationSecret(id: string) {
  return ask<AddedDestination>("POST", `${base}/${id}/secret`);
}

export function removeUpdateDestination(id: string) {
  return ask<void>("DELETE", `${base}/${id}`);
}

export function readAssetUpdateDestinationChoices(
  assetId: string,
  signal?: AbortSignal,
) {
  return ask<{ destinations: AssetUpdateDestinationChoice[] }>(
    "GET",
    `/assets/${assetId}/update-destinations`,
    { signal },
  );
}

export function readAssetUpdateAnnouncements(
  assetId: string,
  signal?: AbortSignal,
) {
  return ask<{ announcements: AssetUpdateAnnouncement[] }>(
    "GET",
    `/assets/${assetId}/announcements`,
    { signal },
  );
}
