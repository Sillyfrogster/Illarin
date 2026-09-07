"use client";

import { createContext, type ReactNode, useContext, useState } from "react";

export type Candidate = { version: number };

const WorkingCopyContext = createContext<Candidate | null>(null);

export function WorkingCopyProvider({
  version,
  children,
}: {
  version: number | undefined;
  children: ReactNode;
}) {
  const [candidate] = useState<Candidate>(() => ({ version: version ?? 0 }));
  return (
    <WorkingCopyContext.Provider value={candidate}>
      {children}
    </WorkingCopyContext.Provider>
  );
}

export function useWorkingCopy() {
  const candidate = useContext(WorkingCopyContext);
  if (!candidate)
    throw new Error("The working copy is unavailable. Reload the page.");
  return candidate;
}

export function acceptCandidateVersion(
  candidate: Candidate,
  response: Response,
) {
  if (!response.ok) return;
  const version = Number(response.headers.get("X-Working-Copy-Version"));
  if (Number.isSafeInteger(version) && version > candidate.version)
    candidate.version = version;
}
