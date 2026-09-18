import { ask } from "./request";

import type {
  AddedWorkUpdateDestination,
  AddWorkUpdateDestinationRequest,
  UpdateWorkUpdateDestinationRequest,
  WorkUpdateAnnouncement,
  WorkUpdateDestination,
  WorkUpdateDestinationChoice,
} from "./shapes";
export type {
  WorkUpdateAnnouncement,
  WorkUpdateDestination,
  WorkUpdateDestinationChoice,
};

type AddedDestination = AddedWorkUpdateDestination;
type NewDestination = AddWorkUpdateDestinationRequest;
type DestinationChange = UpdateWorkUpdateDestinationRequest;

const base = "/account/update-destinations";

export function readUpdateDestinations(signal?: AbortSignal) {
  return ask<{ destinations: WorkUpdateDestination[] }>("GET", base, {
    signal,
  });
}

export function addUpdateDestination(body: NewDestination) {
  return ask<AddedDestination>("POST", base, { body });
}

export function changeUpdateDestination(id: string, body: DestinationChange) {
  return ask<WorkUpdateDestination>("PATCH", `${base}/${id}`, { body });
}

export function verifyUpdateDestination(id: string) {
  return ask<WorkUpdateDestination>("POST", `${base}/${id}/verification`);
}

export function disableUpdateDestination(id: string) {
  return ask<WorkUpdateDestination>("DELETE", `${base}/${id}/verification`);
}

export function rotateUpdateDestinationSecret(id: string) {
  return ask<AddedDestination>("POST", `${base}/${id}/secret`);
}

export function removeUpdateDestination(id: string) {
  return ask<void>("DELETE", `${base}/${id}`);
}

export function readWorkUpdateDestinationChoices(
  workId: string,
  signal?: AbortSignal,
) {
  return ask<{ destinations: WorkUpdateDestinationChoice[] }>(
    "GET",
    `/works/${workId}/update-destinations`,
    { signal },
  );
}

export function readWorkUpdateAnnouncements(
  workId: string,
  signal?: AbortSignal,
) {
  return ask<{ announcements: WorkUpdateAnnouncement[] }>(
    "GET",
    `/works/${workId}/announcements`,
    { signal },
  );
}
