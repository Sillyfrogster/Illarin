"use client";

import { DndContext, DragOverlay } from "@dnd-kit/core";
import { useRouter } from "next/navigation";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  addMarkdownToShelf,
  addWorkImage,
  fetchShelf,
  letGoOfShelfImport,
  letGoOfShelfPiece,
  placeShelfImport,
  placeShelfPiece,
  type ShelfImport,
  type ShelfPiece,
  undoShelfPlacements,
} from "@/lib/api/query";
import { useDraftedChanges } from "@/lib/drafted-changes";
import { useMediaQuery } from "@/lib/use-media-query";
import { type Flight, ShelfFlight } from "./ShelfFlight";
import { LiftedSection, sectionName } from "./ShelfPiece";
import { ShelfToast, type Toast } from "./ShelfToast";
import { type ShelfTarget, useShelfDrag } from "./shelf-drag";
import { importCounts } from "./shelf-places";
import { useWorkspace } from "./state";

/** A section takes a position or a text element; a picture held by address takes an uploaded copy. */
type Placement = {
  position?: number;
  elementId?: string;
  blockId?: string;
  file?: File;
};

/** One import as the pane shows it: its pieces this session in the source's order, and where the placed ones went. */
export type ShownImport = ShelfImport & {
  placed: Record<string, string | null>;
};

type Shelf = {
  imports: ShownImport[];
  count: number;
  busy: string | null;
  canDrag: boolean;
  dragging: ShelfPiece | null;
  target: ShelfTarget | null;
  pending: { piece: ShelfPiece; target: ShelfTarget } | null;
  glowing: string | null;
  shifting: boolean;
  add: (markdown: string) => Promise<string>;
  place: (piece: ShelfPiece, placement?: Placement) => Promise<boolean>;
  placeAll: (held: ShelfImport) => void;
  letGo: (piece: ShelfPiece) => void;
  letGoOfImport: (held: ShelfImport) => void;
  goTo: (blockId: string) => void;
  arrivesQuietly: () => boolean;
};

const DRAG_QUERY = "(min-width: 1024px) and (pointer: fine)";

const SAVING_FIRST = "Your last edit is still saving. Try again in a moment.";
const GLOW_MS = 1800;
const SETTLE_WAIT_MS = 6000;

const ShelfContext = createContext<Shelf | null>(null);

export function useShelf(): Shelf {
  const shelf = useContext(ShelfContext);
  if (!shelf) throw new Error("This page has no shelf.");
  return shelf;
}

function without(imports: ShelfImport[], leaving: (id: string) => boolean) {
  return imports.map((held) => ({
    ...held,
    pieces: held.pieces.filter((piece) => !leaving(piece.id)),
  }));
}

export function ShelfProvider({ children }: { children: ReactNode }) {
  const workspace = useWorkspace();
  const candidate = useDraftedChanges();
  const router = useRouter();
  const canDrag = useMediaQuery(DRAG_QUERY);
  const [server, setServer] = useState<ShelfImport[]>([]);
  const [seen, setSeen] = useState<ShelfImport[]>([]);
  const [placed, setPlaced] = useState<Record<string, string | null>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [pending, setPending] = useState<Shelf["pending"]>(null);
  const [glowing, setGlowing] = useState<string | null>(null);
  const [flight, setFlight] = useState<Flight | null>(null);
  const [toast, setToast] = useState<Toast | null>(null);
  const quiet = useRef(false);
  const latest = useRef(workspace);
  useEffect(() => {
    latest.current = workspace;
  });
  const { workId, isOwner } = workspace;

  // The pane keeps every piece it has shown this session in the source's order, so placed ones can stay as a record
  const read = useCallback((next: ShelfImport[]) => {
    setServer(next);
    setSeen((known) => {
      const merged = next.map((held) => {
        const before = known.find((one) => one.id === held.id);
        if (!before) return held;
        const current = new Map(held.pieces.map((piece) => [piece.id, piece]));
        const fresh = held.pieces.filter(
          (piece) => !before.pieces.some((one) => one.id === piece.id),
        );
        return {
          ...held,
          pieces: [
            ...before.pieces.map((piece) => current.get(piece.id) ?? piece),
            ...fresh,
          ],
        };
      });
      const gone = known.filter(
        (one) => !next.some((held) => held.id === one.id),
      );
      return [...merged, ...gone];
    });
  }, []);

  const reload = useCallback(async () => {
    try {
      read(await fetchShelf(workId));
    } catch {
      /* The shelf keeps what it last read until the next read works. */
    }
  }, [read, workId]);

  useEffect(() => {
    if (isOwner) void reload();
  }, [isOwner, reload]);

  useEffect(() => {
    if (!glowing) return;
    const timer = window.setTimeout(() => setGlowing(null), GLOW_MS);
    return () => window.clearTimeout(timer);
  }, [glowing]);

  // A placement reads the saved page, so it waits for the workspace's own save to land first
  const settled = useCallback(async () => {
    const until = Date.now() + SETTLE_WAIT_MS;
    while (latest.current.dirty || latest.current.busy) {
      if (Date.now() > until) return false;
      await new Promise((done) => window.setTimeout(done, 120));
    }
    return true;
  }, []);

  const refuse = (error: unknown, fallback: string) =>
    setToast({
      kind: "said",
      tone: "stop",
      text: error instanceof Error ? error.message : fallback,
    });

  function goTo(blockId: string) {
    document
      .getElementById(`block-${blockId}`)
      ?.scrollIntoView({ block: "center", behavior: "smooth" });
    setGlowing(blockId);
  }

  function markPlaced(entries: [string, string | null][]) {
    const ids = new Set(entries.map(([id]) => id));
    setPlaced((now) => ({ ...now, ...Object.fromEntries(entries) }));
    setServer((known) => without(known, (id) => ids.has(id)));
  }

  async function place(
    piece: ShelfPiece,
    placement?: Placement,
  ): Promise<boolean> {
    if (busy) return false;
    setBusy(piece.id);
    try {
      if (!(await settled())) {
        setToast({ kind: "said", tone: "stop", text: SAVING_FIRST });
        return false;
      }
      const before = new Set(latest.current.blocks.map((block) => block.id));
      const mediaId = placement?.file
        ? await addWorkImage(candidate, workId, placement.file, "gallery")
        : undefined;
      const saved = await placeShelfPiece(candidate, workId, piece.id, {
        mediaId,
        position: placement?.position,
        elementId: placement?.elementId,
      });
      workspace.applyServerBlocks(saved);
      const made = saved.find((block) => !before.has(block.id))?.id;
      const landed = made ?? placement?.blockId ?? piece.blockId ?? null;
      markPlaced([[piece.id, landed]]);
      if (landed) setGlowing(landed);
      setToast({
        kind: "placed",
        pieceIds: [piece.id],
        name: pieceName(piece),
      });
      void reload();
      if (placement?.file) router.refresh();
      return true;
    } catch (error) {
      quiet.current = false;
      refuse(error, "Illarin could not place the piece. Try again.");
      return false;
    } finally {
      setBusy(null);
      setPending(null);
    }
  }

  async function placeAll(held: ShelfImport) {
    if (busy) return;
    setBusy(held.id);
    try {
      if (!(await settled())) {
        setToast({ kind: "said", tone: "stop", text: SAVING_FIRST });
        return;
      }
      const before = new Set(latest.current.blocks.map((block) => block.id));
      const result = await placeShelfImport(candidate, workId, held.id);
      workspace.applyServerBlocks(result.blocks);
      const made = result.blocks
        .filter((block) => !before.has(block.id))
        .map((block) => block.id);
      const sections = held.pieces
        .filter((piece) => piece.kind === "section")
        .map((piece) => piece.id)
        .filter((id) => result.pieceIds.includes(id));
      markPlaced(
        result.pieceIds.map((id) => [id, made[sections.indexOf(id)] ?? null]),
      );
      if (made[0]) setGlowing(made[0]);
      const count = result.pieceIds.length;
      setToast({
        kind: "placed",
        pieceIds: result.pieceIds,
        name: `${count} ${count === 1 ? "piece" : "pieces"}`,
      });
      void reload();
    } catch (error) {
      refuse(error, "Illarin could not place the import. Try again.");
    } finally {
      setBusy(null);
    }
  }

  async function undo(pieceIds: string[], name: string) {
    if (!(await settled())) {
      setToast({ kind: "said", tone: "stop", text: SAVING_FIRST });
      return;
    }
    try {
      const saved = await undoShelfPlacements(candidate, workId, pieceIds);
      workspace.applyServerBlocks(saved);
      setPlaced((now) =>
        Object.fromEntries(
          Object.entries(now).filter(([id]) => !pieceIds.includes(id)),
        ),
      );
      await reload();
      setToast({
        kind: "said",
        text:
          pieceIds.length === 1
            ? `“${name}” is back on the shelf.`
            : `${name} are back on the shelf.`,
      });
    } catch (error) {
      refuse(error, "Illarin could not undo that. Try again.");
    }
  }

  async function letGo(piece: ShelfPiece) {
    if (busy) return;
    setBusy(piece.id);
    try {
      await letGoOfShelfPiece(candidate, workId, piece.id);
      setSeen((known) => without(known, (id) => id === piece.id));
      setServer((known) => without(known, (id) => id === piece.id));
    } catch (error) {
      refuse(error, "Illarin could not let that go. Try again.");
    } finally {
      setBusy(null);
    }
  }

  async function letGoOfImport(held: ShelfImport) {
    if (busy) return;
    setBusy(held.id);
    try {
      await letGoOfShelfImport(candidate, workId, held.id);
      setSeen((known) => known.filter((one) => one.id !== held.id));
      setServer((known) => known.filter((one) => one.id !== held.id));
    } catch (error) {
      refuse(error, "Illarin could not let that go. Try again.");
    } finally {
      setBusy(null);
    }
  }

  async function add(markdown: string): Promise<string> {
    const known = new Set(seen.map((held) => held.id));
    const next = await addMarkdownToShelf(workId, markdown);
    read(next);
    const made = next.find((held) => !known.has(held.id));
    return made ? `Added ${importCounts(made.pieces, " and ")}.` : "Added.";
  }

  const drag = useShelfDrag({
    onStart: () => workspace.setCursor(null),
    onDrop: (piece, landing, path) => {
      if (path) setFlight({ ...path, title: sectionName(piece) });
      setPending({ piece, target: landing });
      quiet.current = "position" in landing;
      void place(piece, landing);
    },
  });
  const { dragging, target } = drag;

  const waiting = new Set(
    server.flatMap((held) => held.pieces.map((piece) => piece.id)),
  );
  const imports: ShownImport[] = seen
    .map((held) => ({
      ...held,
      pieces: held.pieces.filter(
        (piece) => waiting.has(piece.id) || piece.id in placed,
      ),
      placed: Object.fromEntries(
        held.pieces
          .filter((piece) => piece.id in placed)
          .map((piece) => [piece.id, placed[piece.id]]),
      ),
    }))
    .filter((held) => held.pieces.length > 0);

  const shelf: Shelf = {
    imports,
    count: waiting.size,
    busy,
    canDrag,
    dragging,
    target,
    pending,
    glowing,
    shifting: dragging !== null || pending !== null || glowing !== null,
    add,
    place,
    placeAll: (held) => void placeAll(held),
    letGo: (piece) => void letGo(piece),
    letGoOfImport: (held) => void letGoOfImport(held),
    goTo,
    arrivesQuietly: () => {
      const was = quiet.current;
      quiet.current = false;
      return was;
    },
  };

  return (
    <ShelfContext.Provider value={shelf}>
      <DndContext {...drag.context}>
        {children}
        <DragOverlay dropAnimation={null} zIndex={60}>
          {dragging ? <LiftedSection piece={dragging} /> : null}
        </DragOverlay>
      </DndContext>
      <ShelfFlight flight={flight} onLanded={() => setFlight(null)} />
      <ShelfToast onClose={() => setToast(null)} onUndo={undo} toast={toast} />
    </ShelfContext.Provider>
  );
}

export function pieceName(piece: ShelfPiece): string {
  if (piece.kind === "section") return sectionName(piece);
  return piece.name?.trim() || "Picture";
}
