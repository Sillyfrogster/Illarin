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
  type AssetBlock,
  type AssetElement,
  saveAssetBlock,
  saveAssetIdentity,
} from "@/lib/api/query";
import { useWorkingCopy, WORKING_COPY_STALE } from "@/lib/working-copy";
import {
  type AllowedApp,
  hasSealedPrompts,
  NO_ALLOWED_APP,
} from "../SealedPolicy";
import { unsealedPrompts } from "../UnsealConfirmation";
import {
  blockSaveRequest,
  changedBlockIds,
  replaceBlock,
  replaceElement,
} from "./save";

export type Identity = { name: string; isNsfw: boolean | null };

export type SaveState =
  | "saving"
  | "unsaved"
  | "failed"
  | "private"
  | "published";

export type Pane =
  | { kind: "access" }
  | { kind: "publication" }
  | { kind: "conflict" }
  | { kind: "element"; blockId: string; elementId: string };

type Unsealing = { prompts: string[]; keepsASeal: boolean };

type Workspace = {
  assetId: string;
  isOwner: boolean;
  isDraft: boolean;
  editing: boolean;
  sweep: number;
  blocks: AssetBlock[];
  identity: Identity;
  allowedApps: AllowedApp[];
  eligibleApps: AllowedApp[];
  cursor: string | null;
  chosenItems: Record<string, string>;
  dirty: boolean;
  pane: Pane | null;
  saveState: SaveState;
  message: string;
  busy: boolean;
  unsealing: Unsealing | null;
  startEditing: () => void;
  stopEditing: () => void;
  setCursor: (cursor: string | null) => void;
  chooseItem: (elementId: string, key: string | null) => void;
  setBlocks: (blocks: AssetBlock[]) => void;
  applyServerBlocks: (blocks: AssetBlock[]) => void;
  editBlockList: (change: (blocks: AssetBlock[]) => AssetBlock[]) => void;
  writeElement: (blockId: string, element: AssetElement) => void;
  writeIdentity: (identity: Identity) => void;
  setAllowedApps: (apps: AllowedApp[]) => void;
  openPane: (pane: Pane) => void;
  closePane: () => void;
  say: (message: string) => void;
  save: () => void;
  confirmUnseal: () => void;
  cancelUnseal: () => void;
};

const WorkspaceContext = createContext<Workspace | null>(null);

export function useWorkspace() {
  const workspace = useContext(WorkspaceContext);
  if (!workspace) throw new Error("This page has no editing workspace.");
  return workspace;
}

export function AssetWorkspace({
  assetId,
  isOwner,
  isDraft,
  blocks,
  identity,
  allowedApps,
  eligibleApps,
  unpublishedChanges,
  children,
}: {
  assetId: string;
  isOwner: boolean;
  isDraft: boolean;
  blocks: AssetBlock[];
  identity: Identity;
  allowedApps: AllowedApp[];
  eligibleApps: AllowedApp[];
  unpublishedChanges: boolean;
  children: ReactNode;
}) {
  const candidate = useWorkingCopy();
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [sweep, setSweep] = useState(0);
  const [draft, setDraft] = useState(blocks);
  const [saved, setSaved] = useState(blocks);
  const [draftIdentity, setDraftIdentity] = useState(identity);
  const [savedIdentity, setSavedIdentity] = useState(identity);
  const [apps, setApps] = useState<AllowedApp[]>(allowedApps);
  const [cursor, setCursor] = useState<string | null>(null);
  const [chosenItems, setChosenItems] = useState<Record<string, string>>({});
  const [pane, setPane] = useState<Pane | null>(null);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const [message, setMessage] = useState("");
  const [unsealing, setUnsealing] = useState<Unsealing | null>(null);
  const exposeConfirmed = useRef(false);
  const lastCursor = useRef<string | null>(null);
  const savedBlocks = useRef(blocks);

  useEffect(() => {
    savedBlocks.current = saved;
  }, [saved]);

  useEffect(() => {
    const stale = () => setPane({ kind: "conflict" });
    window.addEventListener(WORKING_COPY_STALE, stale);
    return () => window.removeEventListener(WORKING_COPY_STALE, stale);
  }, []);

  useEffect(() => {
    if (!message) return;
    const timer = window.setTimeout(() => setMessage(""), 7000);
    return () => window.clearTimeout(timer);
  }, [message]);

  const changed = useMemo(() => changedBlockIds(draft, saved), [draft, saved]);
  const identityChanged =
    draftIdentity.name !== savedIdentity.name ||
    draftIdentity.isNsfw !== savedIdentity.isNsfw;
  const dirty = changed.length > 0 || identityChanged;

  const saveState: SaveState = busy
    ? "saving"
    : failed
      ? "failed"
      : dirty
        ? "unsaved"
        : isDraft || unpublishedChanges
          ? "private"
          : "published";

  /** A save refused for want of an allowed app opens the element that asks for one. */
  const openSealedElement = useCallback((pages: AssetBlock[]) => {
    for (const block of pages) {
      const asking = block.elements.find((element) =>
        hasSealedPrompts([element]),
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
            if (was) exposed.push(...unsealedPrompts(was, element));
          }
        }
        if (exposed.length > 0) {
          const keepsASeal = draft.some((block) =>
            hasSealedPrompts(block.elements),
          );
          setUnsealing({ prompts: exposed, keepsASeal });
          return;
        }
      }

      const sealed = draft.some((block) => hasSealedPrompts(block.elements));
      if (sealed && apps.length === 0) {
        setFailed(true);
        setMessage(NO_ALLOWED_APP);
        openSealedElement(draft);
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
            const result = await saveAssetBlock(
              candidate,
              assetId,
              block.id,
              blockSaveRequest(block, {
                exposeProtected: expose || exposeConfirmed.current || undefined,
                allowedApps: sealed
                  ? apps
                  : allowedApps.length > 0
                    ? []
                    : undefined,
              }),
            );
            written = replaceBlock(written, result);
          }
          if (identityChanged) {
            await saveAssetIdentity(candidate, assetId, draftIdentity);
            setSavedIdentity(draftIdentity);
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
      assetId,
      busy,
      candidate,
      dirty,
      draft,
      draftIdentity,
      identityChanged,
      isDraft,
      openSealedElement,
      router,
      saved,
    ],
  );

  // Writing the creator has not saved yet is kept over the server's answer.
  const applyServerBlocks = useCallback((incoming: AssetBlock[]) => {
    setDraft((current) => {
      const changed = new Set(changedBlockIds(current, savedBlocks.current));
      const held = new Map(current.map((block) => [block.id, block]));
      return incoming.map((block) => {
        const mine = changed.has(block.id) ? held.get(block.id) : undefined;
        return mine ? { ...block, elements: mine.elements } : block;
      });
    });
    setSaved(incoming);
  }, []);

  useEffect(() => {
    applyServerBlocks(blocks);
  }, [applyServerBlocks, blocks]);

  useEffect(() => {
    const answer = { isNsfw: identity.isNsfw, name: identity.name };
    setDraftIdentity(answer);
    setSavedIdentity(answer);
  }, [identity.isNsfw, identity.name]);

  /** Adding or removing a block changes the saved copy as well, so it reads as saved. */
  const editBlockList = useCallback(
    (change: (blocks: AssetBlock[]) => AssetBlock[]) => {
      setDraft(change);
      setSaved(change);
    },
    [],
  );

  const value: Workspace = {
    assetId,
    isOwner,
    isDraft,
    editing,
    sweep,
    blocks: draft,
    identity: draftIdentity,
    allowedApps: apps,
    eligibleApps,
    cursor,
    chosenItems,
    dirty,
    pane,
    saveState,
    message,
    busy,
    unsealing,
    startEditing: () => {
      setEditing(true);
      setSweep((count) => count + 1);
      setCursor(lastCursor.current);
    },
    stopEditing: () => {
      lastCursor.current = cursor;
      setEditing(false);
      setCursor(null);
      setPane(null);
    },
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
    applyServerBlocks,
    editBlockList,
    writeElement: (blockId, element) =>
      setDraft((current) =>
        current.map((block) =>
          block.id === blockId ? replaceElement(block, element) : block,
        ),
      ),
    writeIdentity: setDraftIdentity,
    setAllowedApps: setApps,
    openPane: (next) => {
      setPane(next);
      setCursor(null);
    },
    closePane: () => setPane(null),
    say: setMessage,
    save: () => save(),
    confirmUnseal: () => {
      exposeConfirmed.current = true;
      setUnsealing(null);
      save(true);
    },
    cancelUnseal: () => setUnsealing(null),
  };

  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  );
}
