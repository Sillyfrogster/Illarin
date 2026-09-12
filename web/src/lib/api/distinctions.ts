import type {
  Distinction,
  DistinctionAssignment,
  DistinctionForm,
} from "@/lib/api/query";
import { ask, json } from "./request";

export function readDefinitions() {
  return json<{ definitions: Distinction[] }>("/distinctions", "GET");
}

export function defineDistinction(
  form: DistinctionForm,
  name: string,
  explanation: string,
) {
  return json<Distinction>("/distinctions", "POST", {
    form,
    name,
    explanation,
  });
}

export function updateDistinction(
  id: string,
  change: { name?: string; explanation?: string; retired?: boolean },
) {
  return json<Distinction>(`/distinctions/${id}`, "PATCH", change);
}

export function orderDistinctions(
  form: DistinctionForm,
  distinctionIds: string[],
) {
  return json<{ definitions: Distinction[] }>("/distinctions", "PUT", {
    form,
    distinctionIds,
  });
}

export function uploadMark(id: string, file: File) {
  const body = new FormData();
  body.append("file", file);
  return ask<Distinction>(
    `/distinctions/${id}/mark`,
    { method: "PUT", body },
    (response) => response.json() as Promise<Distinction>,
  );
}

export function clearMark(id: string) {
  return json<Distinction>(`/distinctions/${id}/mark`, "DELETE");
}

export function readAccountDistinctions(handle: string) {
  return json<{ handle: string; assignments: DistinctionAssignment[] }>(
    `/accounts/${encodeURIComponent(handle)}/distinctions`,
    "GET",
  );
}

export function giveDistinction(handle: string, distinctionId: string) {
  return json<DistinctionAssignment>(
    `/accounts/${encodeURIComponent(handle)}/distinctions`,
    "POST",
    { distinctionId },
  );
}

export function takeBackDistinction(handle: string, assignmentId: string) {
  return ask<null>(
    `/accounts/${encodeURIComponent(handle)}/distinctions/${assignmentId}`,
    { method: "DELETE" },
    async () => null,
  );
}

export function orderAccountDistinctions(
  handle: string,
  assignmentIds: string[],
) {
  return json<{ handle: string; assignments: DistinctionAssignment[] }>(
    `/accounts/${encodeURIComponent(handle)}/distinctions`,
    "PUT",
    { assignmentIds },
  );
}
