"use client";

import { createContext, type ReactNode, useContext, useState } from "react";

export type Candidate = { version: number };

export const DRAFTED_CHANGES_SAVED = "illarin:drafted-changes-saved";

export const DRAFTED_CHANGES_STALE = "illarin:drafted-changes-stale";

const DraftedChangesContext = createContext<Candidate | null>(null);

export function DraftedChangesProvider({
  version,
  children,
}: {
  version: number | undefined;
  children: ReactNode;
}) {
  const [candidate] = useState<Candidate>(() => ({ version: version ?? 0 }));
  return (
    <DraftedChangesContext.Provider value={candidate}>
      {children}
    </DraftedChangesContext.Provider>
  );
}

export function useDraftedChanges() {
  const candidate = useContext(DraftedChangesContext);
  if (!candidate)
    throw new Error("The drafted changes are unavailable. Reload the page.");
  return candidate;
}

export function acceptCandidateVersion(
  candidate: Candidate,
  response: Response,
) {
  if (!response.ok) return;
  const version = Number(response.headers.get("X-Drafted-Changes-Version"));
  if (Number.isSafeInteger(version) && version > candidate.version) {
    candidate.version = version;
    window.dispatchEvent(new Event(DRAFTED_CHANGES_SAVED));
  }
}

export function reportStaleDraftedChanges(refusal: unknown) {
  const detail = refusal as { code?: unknown } | undefined;
  if (detail?.code !== "drafted_changes_conflict") return;
  window.dispatchEvent(new Event(DRAFTED_CHANGES_STALE));
}
