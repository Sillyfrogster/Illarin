"use client";

import {
  createContext,
  type ReactNode,
  type RefObject,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { fetchWorkUpdates, type RecordedVersion } from "@/lib/api/query";

export type Versions =
  | { state: "unread" }
  | { state: "reading" }
  | { state: "read"; versions: RecordedVersion[] }
  | { state: "refused" };

type Stage = {
  card: RefObject<HTMLDivElement | null>;
  turned: boolean;
  turn: (next: boolean) => void;
  versions: Versions;
  readVersions: () => Promise<void>;
  viewing: RecordedVersion | null;
  view: (version: RecordedVersion | null) => void;
  peek: RecordedVersion | null;
  setPeek: (version: RecordedVersion | null) => void;
  collecting: boolean;
  setCollecting: (collecting: boolean) => void;
  backSlot: HTMLElement | null;
  setBackSlot: (slot: HTMLElement | null) => void;
};

const StageContext = createContext<Stage | null>(null);

/** CardStage holds what the work card and the controls around it share: which side shows, which version, and whether an app has yet to collect a send. */
export function CardStage({
  children,
  workId,
  recorded,
}: {
  children: ReactNode;
  workId: string;
  recorded: boolean;
}) {
  const card = useRef<HTMLDivElement>(null);
  const [turned, turn] = useState(false);
  const [versions, setVersions] = useState<Versions>({ state: "unread" });
  const [viewing, setViewing] = useState<RecordedVersion | null>(null);
  const [peek, setPeek] = useState<RecordedVersion | null>(null);
  const [collecting, setCollecting] = useState(false);
  const [backSlot, setBackSlot] = useState<HTMLElement | null>(null);

  const readVersions = useCallback(async () => {
    setVersions({ state: "reading" });
    const items = await fetchWorkUpdates(workId);
    setVersions(
      items ? { state: "read", versions: items } : { state: "refused" },
    );
  }, [workId]);

  useEffect(() => {
    if (recorded) void readVersions();
  }, [recorded, readVersions]);

  const view = useCallback((version: RecordedVersion | null) => {
    setViewing(version);
    setPeek(null);
  }, []);

  return (
    <StageContext.Provider
      value={{
        card,
        turned,
        turn,
        versions,
        readVersions,
        viewing,
        view,
        peek,
        setPeek,
        collecting,
        setCollecting,
        backSlot,
        setBackSlot,
      }}
    >
      {children}
    </StageContext.Provider>
  );
}

export function useCardStage(): Stage {
  const stage = useContext(StageContext);
  if (!stage) throw new Error("useCardStage needs a CardStage above it");
  return stage;
}

/** listedVersions is the read version list, newest first, or nothing while it loads. */
export function listedVersions(versions: Versions): RecordedVersion[] {
  return versions.state === "read" ? versions.versions : [];
}
