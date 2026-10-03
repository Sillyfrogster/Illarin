"use client";

import { useRouter } from "next/navigation";
import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";

export type Candidate = { workId: string; version: number };

export const DRAFTED_CHANGES_SAVED = "illarin:drafted-changes-saved";

export const DRAFTED_CHANGES_STALE = "illarin:drafted-changes-stale";

/** The newest drafted-changes version each work reached from this tab, kept across mounts because Back replays an older page */
const savedVersions = new Map<string, number>();

const DraftedChangesContext = createContext<Candidate | null>(null);

/** Re-reads the page while it is older than this tab's last save, and holds the editor until it catches up */
export function DraftedChangesProvider({
  workId,
  version = 0,
  children,
}: {
  workId: string;
  version: number | undefined;
  children: ReactNode;
}) {
  const router = useRouter();
  const current = !isOlderThanSaved(workId, version);
  const [candidate, setCandidate] = useState<Candidate | null>(() =>
    current ? { workId, version } : null,
  );
  if (!candidate && current) setCandidate({ workId, version });
  useEffect(() => {
    if (!current) router.refresh();
  }, [current, router]);
  if (!candidate) return null;
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

/** Whether a page read carries drafted changes older than ones this tab has already saved. */
export function isOlderThanSaved(workId: string, version: number): boolean {
  return version < (savedVersions.get(workId) ?? 0);
}

export function acceptCandidateVersion(
  candidate: Candidate,
  response: Response,
) {
  if (!response.ok) return;
  const version = Number(response.headers.get("X-Drafted-Changes-Version"));
  if (!Number.isSafeInteger(version)) return;
  if (!isOlderThanSaved(candidate.workId, version))
    savedVersions.set(candidate.workId, version);
  if (version > candidate.version) {
    candidate.version = version;
    window.dispatchEvent(new Event(DRAFTED_CHANGES_SAVED));
  }
}

export function reportStaleDraftedChanges(refusal: unknown) {
  const detail = refusal as { code?: unknown } | undefined;
  if (detail?.code !== "drafted_changes_conflict") return;
  window.dispatchEvent(new Event(DRAFTED_CHANGES_STALE));
}
