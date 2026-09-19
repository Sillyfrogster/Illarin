"use client";

import { useRouter } from "next/navigation";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  type AddableBlock,
  saveWorkBlock,
  saveWorkDetails,
  type WorkBlock,
  type WorkDetailsRequest,
  type WorkElement,
} from "@/lib/api/query";
import type { AppName } from "@/lib/api/shapes";
import {
  DRAFTED_CHANGES_STALE,
  useDraftedChanges,
} from "@/lib/drafted-changes";
import { promptsMadePublic } from "../MakePublicConfirmation";
import { hasPrivatePrompts, NO_ALLOWED_APP } from "../PrivatePromptsControl";
import { type Arrangement, useArrangement } from "./arrangement";
import { seatElements } from "./composition";
import { detailsHasChanged } from "./details";
import {
  blockSaveRequest,
  changedBlockIds,
  replaceBlock,
  replaceElement,
} from "./save";

export type Details = WorkDetailsRequest;

export type SaveState =
  | "saving"
  | "unsaved"
  | "failed"
  | "private"
  | "published";

export type Pane =
  | { kind: "access" }
  | { kind: "vault" }
  | { kind: "publication" }
  | { kind: "conflict" }
  | { kind: "add-block" }
  | { kind: "remove"; blockId: string }
  | { kind: "element"; blockId: string; elementId: string };

type MakingPublic = { prompts: string[]; keepsAPrivatePrompt: boolean };

type Workspace = {
  workId: string;
  isOwner: boolean;
  isDraft: boolean;
  editing: boolean;
  sweep: number;
  blocks: WorkBlock[];
  addableBlocks: AddableBlock[];
  arrangement: Arrangement;
  details: Details;
  allowedApps: AppName[];
  eligibleApps: AppName[];
  cursor: string | null;
  chosenItems: Record<string, string>;
  dirty: boolean;
  pane: Pane | null;
  saveState: SaveState;
  message: string;
  busy: boolean;
  makingPublic: MakingPublic | null;
  startEditing: () => void;
  stopEditing: () => void;
  setCursor: (cursor: string | null) => void;
  chooseItem: (elementId: string, key: string | null) => void;
  setBlocks: (blocks: WorkBlock[]) => void;
  writeBlock: (block: WorkBlock) => void;
  applyServerBlocks: (blocks: WorkBlock[]) => void;
  editBlockList: (change: (blocks: WorkBlock[]) => WorkBlock[]) => void;
  writeElement: (blockId: string, element: WorkElement) => void;
  writeDetails: (details: Details) => void;
  setAllowedApps: (apps: AppName[]) => void;
  openPane: (pane: Pane) => void;
  closePane: () => void;
  say: (message: string) => void;
  save: () => void;
  confirmMakePublic: () => void;
  cancelMakePublic: () => void;
};

const WorkspaceContext = createContext<Workspace | null>(null);

export function useWorkspace() {
  const workspace = useContext(WorkspaceContext);
  if (!workspace) throw new Error("This page has no editing workspace.");
  return workspace;
}

export function WorkspaceProvider({
  addableBlocks,
  workId,
  isOwner,
  isDraft,
  blocks,
  details,
  allowedApps,
  eligibleApps,
  unpublishedChanges,
  children,
}: {
  addableBlocks: AddableBlock[];
  workId: string;
  isOwner: boolean;
  isDraft: boolean;
  blocks: WorkBlock[];
  details: Details;
  allowedApps: AppName[];
  eligibleApps: AppName[];
  unpublishedChanges: boolean;
  children: ReactNode;
}) {
  const candidate = useDraftedChanges();
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [sweep, setSweep] = useState(0);
  const [draft, setDraft] = useState(blocks);
  const [saved, setSaved] = useState(blocks);
  const [draftDetails, setDraftDetails] = useState(details);
  const [savedDetails, setSavedDetails] = useState(details);
  const [apps, setApps] = useState<AppName[]>(allowedApps);
  const [cursor, setCursor] = useState<string | null>(null);
  const [chosenItems, setChosenItems] = useState<Record<string, string>>({});
  const [pane, setPane] = useState<Pane | null>(null);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const [message, setMessage] = useState("");
  const [makingPublic, setMakingPublic] = useState<MakingPublic | null>(null);
  const exposeConfirmed = useRef(false);
  const lastCursor = useRef<string | null>(null);
  const savedBlocks = useRef(blocks);

  useEffect(() => {
    savedBlocks.current = saved;
  }, [saved]);

  useEffect(() => {
    const stale = () =>
      setPane((open) =>
        open?.kind === "publication" ? open : { kind: "conflict" },
      );
    window.addEventListener(DRAFTED_CHANGES_STALE, stale);
    return () => window.removeEventListener(DRAFTED_CHANGES_STALE, stale);
  }, []);

  useEffect(() => {
    if (!message) return;
    const timer = window.setTimeout(() => setMessage(""), 7000);
    return () => window.clearTimeout(timer);
  }, [message]);

  const changed = useMemo(() => changedBlockIds(draft, saved), [draft, saved]);
  const hasDetailsChanges = detailsHasChanged(draftDetails, savedDetails);
  const dirty = changed.length > 0 || hasDetailsChanges;

  const saveState: SaveState = busy
    ? "saving"
    : failed
      ? "failed"
      : dirty
        ? "unsaved"
        : isDraft || unpublishedChanges
          ? "private"
          : "published";

  const openPrivatePromptElement = useCallback((pages: WorkBlock[]) => {
    for (const block of pages) {
      const asking = block.elements.find((element) =>
        hasPrivatePrompts([element]),
      );
      if (!asking) continue;
      setPane({ blockId: block.id, elementId: asking.id, kind: "element" });
      return;
    }
  }, []);

  const save = useCallback(
    (expose = false) => {
      if (busy || !dirty) return;
      const pending = changedBlockIds(draft, saved);
      const before = new Map(saved.map((block) => [block.id, block]));

      if (!expose && !exposeConfirmed.current) {
        const exposed: string[] = [];
        for (const id of pending) {
          const after = draft.find((block) => block.id === id);
          const original = before.get(id);
          if (!after || !original) continue;
          for (const element of after.elements) {
            const was = original.elements.find(
              (item) => item.id === element.id,
            );
            if (was) exposed.push(...promptsMadePublic(was, element));
          }
        }
        if (exposed.length > 0) {
          const keepsAPrivatePrompt = draft.some((block) =>
            hasPrivatePrompts(block.elements),
          );
          setMakingPublic({ prompts: exposed, keepsAPrivatePrompt });
          return;
        }
      }

      const keepsPrivatePrompts = draft.some((block) =>
        hasPrivatePrompts(block.elements),
      );
      if (keepsPrivatePrompts && apps.length === 0) {
        setFailed(true);
        setMessage(NO_ALLOWED_APP);
        openPrivatePromptElement(draft);
        return;
      }

      setBusy(true);
      setFailed(false);
      setMessage("");
      void (async () => {
        let written = draft;
        try {
          for (const id of pending) {
            const block = written.find((item) => item.id === id);
            if (!block) continue;
            const result = await saveWorkBlock(
              candidate,
              workId,
              block.id,
              blockSaveRequest(block, {
                makePromptsPublic:
                  expose || exposeConfirmed.current || undefined,
                allowedApps: keepsPrivatePrompts
                  ? apps.map((app) => app.id)
                  : allowedApps.length > 0
                    ? []
                    : undefined,
              }),
            );
            written = replaceBlock(written, result);
          }
          if (hasDetailsChanges) {
            await saveWorkDetails(candidate, workId, draftDetails);
            setSavedDetails(draftDetails);
          }
          exposeConfirmed.current = false;
          setDraft(written);
          setSaved(written);
          setMessage(
            isDraft
              ? "Saved. Only you can open this page."
              : "Saved privately. Readers still have the published version.",
          );
          router.refresh();
        } catch (error) {
          setFailed(true);
          setMessage(
            error instanceof Error
              ? error.message
              : "The save failed. Everything you wrote is still on the page.",
          );
        } finally {
          setBusy(false);
        }
      })();
    },
    [
      allowedApps.length,
      apps,
      workId,
      busy,
      candidate,
      dirty,
      draft,
      draftDetails,
      hasDetailsChanges,
      isDraft,
      openPrivatePromptElement,
      router,
      saved,
    ],
  );

  const applyServerBlocks = useCallback((incoming: WorkBlock[]) => {
    setDraft((current) => {
      const changed = new Set(changedBlockIds(current, savedBlocks.current));
      const held = new Map(current.map((block) => [block.id, block]));
      const before = new Map(savedBlocks.current.map((one) => [one.id, one]));
      return incoming.map((block) => {
        const mine = changed.has(block.id) ? held.get(block.id) : undefined;
        return mine ? keepWriting(block, mine, before.get(block.id)) : block;
      });
    });
    setSaved(incoming);
  }, []);

  useEffect(() => {
    applyServerBlocks(blocks);
  }, [applyServerBlocks, blocks]);

  useEffect(() => {
    const incomingDetails = {
      blurb: details.blurb,
      isNsfw: details.isNsfw,
      name: details.name,
    };
    setDraftDetails(incomingDetails);
    setSavedDetails(incomingDetails);
  }, [details.blurb, details.isNsfw, details.name]);

  const editBlockList = useCallback(
    (change: (blocks: WorkBlock[]) => WorkBlock[]) => {
      setDraft(change);
      setSaved(change);
    },
    [],
  );

  const draftBlocks = useRef(draft);
  useEffect(() => {
    draftBlocks.current = draft;
  }, [draft]);

  const arrangement = useArrangement({
    applyServerBlocks,
    workId,
    blocks: draftBlocks,
    candidate,
    editBlockList,
    savedBlocks,
    say: setMessage,
  });

  const stopEditing = useCallback(() => {
    lastCursor.current = cursor;
    setEditing(false);
    setCursor(null);
    setPane(null);
  }, [cursor]);

  useEffect(() => {
    if (!editing) return;
    const onEscape = (event: KeyboardEvent) => {
      if (
        event.key !== "Escape" ||
        event.defaultPrevented ||
        event.isComposing ||
        makingPublic
      )
        return;
      if (
        document.querySelector(
          'dialog[open], [role="dialog"], [role="menu"], [role="listbox"]',
        )
      )
        return;
      event.preventDefault();
      if (pane) setPane(null);
      else stopEditing();
    };
    window.addEventListener("keydown", onEscape);
    return () => window.removeEventListener("keydown", onEscape);
  }, [editing, pane, stopEditing, makingPublic]);

  const value: Workspace = {
    addableBlocks,
    arrangement,
    workId,
    isOwner,
    isDraft,
    editing,
    sweep,
    blocks: draft,
    details: draftDetails,
    allowedApps: apps,
    eligibleApps,
    cursor,
    chosenItems,
    dirty,
    pane,
    saveState,
    message,
    busy,
    makingPublic,
    startEditing: () => {
      setEditing(true);
      setSweep((count) => count + 1);
      setCursor(lastCursor.current);
    },
    stopEditing,
    setCursor,
    chooseItem: (elementId, key) =>
      setChosenItems((current) => {
        if (key === null) {
          const { [elementId]: _gone, ...rest } = current;
          return rest;
        }
        return { ...current, [elementId]: key };
      }),
    setBlocks: setDraft,
    writeBlock: (written) =>
      setDraft((current) =>
        current.map((block) => (block.id === written.id ? written : block)),
      ),
    applyServerBlocks,
    editBlockList,
    writeElement: (blockId, element) =>
      setDraft((current) =>
        current.map((block) =>
          block.id === blockId ? replaceElement(block, element) : block,
        ),
      ),
    writeDetails: setDraftDetails,
    setAllowedApps: setApps,
    openPane: (next) => {
      setPane(next);
      setCursor(null);
    },
    closePane: () => setPane(null),
    say: setMessage,
    save: () => save(),
    confirmMakePublic: () => {
      exposeConfirmed.current = true;
      setMakingPublic(null);
      save(true);
    },
    cancelMakePublic: () => setMakingPublic(null),
  };

  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  );
}

function keepWriting(
  server: WorkBlock,
  draft: WorkBlock,
  saved: WorkBlock | undefined,
): WorkBlock {
  const known = new Set(
    [...draft.elements, ...(saved?.elements ?? [])].map((one) => one.id),
  );
  const arrived = server.elements.filter((one) => !known.has(one.id));
  return {
    ...server,
    elements: seatElements(draft.layout, [...draft.elements, ...arrived]),
    layout: draft.layout,
    title: draft.title,
    titleIsDefault: draft.titleIsDefault,
    width: draft.width,
  };
}
