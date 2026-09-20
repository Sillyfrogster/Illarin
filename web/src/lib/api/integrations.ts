import { ask } from "./request";

import type {
  AddedWorkIntegration,
  AddWorkIntegrationRequest,
  UpdateWorkIntegrationRequest,
  WorkAnnouncementAttempt,
  WorkAnnouncementAttemptList,
  WorkIntegration,
  WorkIntegrationChoice,
} from "./shapes";
export type { WorkAnnouncementAttempt, WorkIntegration, WorkIntegrationChoice };

type AddedIntegration = AddedWorkIntegration;
type NewIntegration = AddWorkIntegrationRequest;
type IntegrationChange = UpdateWorkIntegrationRequest;

const base = "/account/integrations";

export function readIntegrations(signal?: AbortSignal) {
  return ask<{ integrations: WorkIntegration[] }>("GET", base, {
    signal,
  });
}

export function addIntegration(body: NewIntegration) {
  return ask<AddedIntegration>("POST", base, { body });
}

export function changeIntegration(id: string, body: IntegrationChange) {
  return ask<WorkIntegration>("PATCH", `${base}/${id}`, { body });
}

export function verifyIntegration(id: string) {
  return ask<WorkIntegration>("POST", `${base}/${id}/verification`);
}

export function disableIntegration(id: string) {
  return ask<WorkIntegration>("DELETE", `${base}/${id}/verification`);
}

export function rotateIntegrationSecret(id: string) {
  return ask<AddedIntegration>("POST", `${base}/${id}/secret`);
}

export function removeIntegration(id: string) {
  return ask<void>("DELETE", `${base}/${id}`);
}

export function readWorkIntegrationChoices(
  workId: string,
  signal?: AbortSignal,
) {
  return ask<{ integrations: WorkIntegrationChoice[] }>(
    "GET",
    `/works/${workId}/integrations`,
    { signal },
  );
}

export function readWorkAnnouncementAttempts(
  workId: string,
  signal?: AbortSignal,
) {
  return ask<WorkAnnouncementAttemptList>(
    "GET",
    `/works/${workId}/announcement-attempts`,
    { signal },
  );
}
