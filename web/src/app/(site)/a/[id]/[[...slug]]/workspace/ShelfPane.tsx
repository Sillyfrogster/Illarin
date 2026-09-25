"use client";

import {
  AnimatePresence,
  LayoutGroup,
  motion,
  useReducedMotion,
} from "framer-motion";
import { FilePlus2, FileText, Layers, Package, Rows3 } from "lucide-react";
import { type DragEvent, useEffect, useId, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { RollingNumber } from "@/components/ui/rolling-number";
import { cn } from "@/lib/cn";
import { PictureTile, SectionRow } from "./ShelfPiece";
import { type ShownImport, useShelf } from "./shelf";
import { importCounts, importLabel } from "./shelf-places";

const SPRING = { type: "spring", stiffness: 380, damping: 34 } as const;
const MAX_MARKDOWN_BYTES = 1 << 20;
const MARKDOWN_FILE = /\.(md|markdown|txt)$/i;
const STACKED = 3;
const CONFIRM_MS = 4000;
const TOO_BIG = "That's over 1 MB of Markdown. Split it and add each part.";

/** ShelfPane holds what waits on the shelf, newest import first, and the way to add more Markdown. */
export function ShelfPane() {
  const shelf = useShelf();
  const [open, setOpen] = useState(false);
  const newestFirst = [...shelf.imports].sort((one, other) =>
    other.createdAt.localeCompare(one.createdAt),
  );
  const [newest, ...older] = newestFirst;
  const empty = shelf.imports.length === 0;

  return (
    <LayoutGroup>
      <div className="flex flex-col gap-10">
        {empty ? (
          <div className="flex flex-col gap-1">
            <p className="font-display text-section font-medium text-ink">
              Nothing on the shelf
            </p>
            <p className="text-meta text-mute">
              Add Markdown and its sections wait here.
            </p>
          </div>
        ) : null}
        <AddMarkdown onOpenChange={setOpen} open={open || empty} />
        {newest ? <ImportCard held={newest} key={newest.id} /> : null}
        {older.length > 0 ? <OlderImports imports={older} /> : null}
      </div>
    </LayoutGroup>
  );
}

function AddMarkdown({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const field = useId();
  const picker = useRef<HTMLInputElement>(null);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [said, setSaid] = useState("");
  const [over, setOver] = useState(false);

  useEffect(() => {
    if (!said) return;
    const timer = window.setTimeout(() => setSaid(""), 6000);
    return () => window.clearTimeout(timer);
  }, [said]);

  async function submit(markdown: string) {
    if (busy) return;
    if (new Blob([markdown]).size > MAX_MARKDOWN_BYTES) {
      setError(TOO_BIG);
      return;
    }
    setBusy(true);
    setError("");
    try {
      setSaid(await shelf.add(markdown));
      setText("");
      onOpenChange(false);
    } catch (failure) {
      setError(
        failure instanceof Error
          ? failure.message
          : "Illarin could not add the Markdown. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function read(file: File | undefined) {
    if (!file) return;
    if (!MARKDOWN_FILE.test(file.name)) {
      setError("That file isn't Markdown. Drop a .md or .txt file.");
      onOpenChange(true);
      return;
    }
    if (file.size > MAX_MARKDOWN_BYTES) {
      setError(TOO_BIG);
      onOpenChange(true);
      return;
    }
    await submit(await file.text());
  }

  const holdsFiles = (event: DragEvent) =>
    event.dataTransfer.types.includes("Files");
  const drops = {
    onDragLeave: (event: DragEvent) => {
      if (!event.currentTarget.contains(event.relatedTarget as Node)) {
        setOver(false);
      }
    },
    onDragOver: (event: DragEvent) => {
      if (!holdsFiles(event)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "copy";
      setOver(true);
    },
    onDrop: (event: DragEvent) => {
      if (!holdsFiles(event)) return;
      event.preventDefault();
      setOver(false);
      void read(event.dataTransfer.files[0]);
    },
  };
  const morph = still ? { duration: 0 } : SPRING;

  return (
    <div className="flex flex-col gap-3">
      <AnimatePresence initial={false} mode="popLayout">
        {open ? (
          <motion.div
            className={cn(
              "relative flex flex-col gap-3 rounded-plate bg-deep p-3",
              over && "inset-ring-2 inset-ring-accent",
            )}
            key="open"
            layoutId="shelf-add"
            style={{ borderRadius: 14 }}
            transition={morph}
            {...drops}
          >
            <motion.div
              animate={{ opacity: 1 }}
              className="flex flex-col gap-3"
              initial={{ opacity: 0 }}
              transition={{ delay: still ? 0 : 0.08, duration: 0.2 }}
            >
              <label className="sr-only" htmlFor={field}>
                Markdown
              </label>
              <textarea
                className="min-h-48 w-full resize-y rounded-control bg-plane p-4 font-mono text-meta leading-relaxed text-ink outline-offset-2 placeholder:text-mute"
                disabled={busy}
                id={field}
                onChange={(event) => {
                  setText(event.target.value);
                  setError("");
                }}
                placeholder="Paste Markdown or drop a .md file"
                spellCheck={false}
                value={text}
              />
              <p className="px-1 text-label text-mute">
                Every heading starts its own piece. For a rentry, copy
                everything on its raw page.
              </p>
              {error ? (
                <p className="px-1 text-meta text-stop" role="alert">
                  {error}
                </p>
              ) : null}
              <div className="flex flex-wrap items-center gap-1">
                <Button
                  disabled={busy}
                  onClick={() => picker.current?.click()}
                  size="compact"
                  variant="ghost"
                >
                  Choose a file
                </Button>
                <span className="flex-1" />
                {shelf.count > 0 ? (
                  <Button
                    disabled={busy}
                    onClick={() => {
                      onOpenChange(false);
                      setError("");
                    }}
                    size="compact"
                    variant="ghost"
                  >
                    Cancel
                  </Button>
                ) : null}
                <Button
                  disabled={text.trim() === ""}
                  loading={busy}
                  onClick={() => void submit(text)}
                  variant="primary"
                >
                  {busy ? "Adding…" : "Add to shelf"}
                </Button>
              </div>
            </motion.div>
            <AnimatePresence>
              {over ? (
                <motion.div
                  animate={{ opacity: 1 }}
                  className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-plate bg-accent-wash/95 font-display text-ui font-medium text-accent"
                  exit={{ opacity: 0 }}
                  initial={{ opacity: 0 }}
                >
                  Drop to add
                </motion.div>
              ) : null}
            </AnimatePresence>
          </motion.div>
        ) : (
          <motion.button
            className={cn(
              "flex min-h-13 w-full items-center gap-3 rounded-plate border-2 border-rule border-dashed px-4 text-left font-ui text-ui font-medium text-ink outline-offset-3 hover:border-accent hover:bg-accent-wash/40",
              over && "border-accent bg-accent-wash/60",
            )}
            key="closed"
            layoutId="shelf-add"
            onClick={() => onOpenChange(true)}
            style={{ borderRadius: 14 }}
            transition={morph}
            type="button"
            {...drops}
          >
            <FilePlus2 aria-hidden="true" className="size-4.5 text-accent" />
            {over ? "Drop to add" : "Add from Markdown"}
          </motion.button>
        )}
      </AnimatePresence>
      <input
        accept=".md,.markdown,.txt,text/markdown,text/plain"
        className="sr-only"
        onChange={(event) => {
          void read(event.target.files?.[0]);
          event.target.value = "";
        }}
        ref={picker}
        tabIndex={-1}
        type="file"
      />
      <AnimatePresence>
        {said ? (
          <motion.output
            animate={{ opacity: 1, y: 0 }}
            className="px-1 text-meta text-mute"
            exit={{ opacity: 0 }}
            initial={{ opacity: 0, y: still ? 0 : -4 }}
          >
            {said}
          </motion.output>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function importName(held: ShownImport): string {
  return held.title.trim() || importLabel(held);
}

function ImportHead({ held }: { held: ShownImport }) {
  const Icon = held.source === "readme" ? Package : FileText;
  const waiting = held.pieces.filter((piece) => !(piece.id in held.placed));
  const where = importLabel(held).split(" · ")[0];
  return (
    <div className="flex min-w-0 items-start gap-3">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-control bg-accent-wash text-accent">
        <Icon aria-hidden="true" className="size-5" />
      </span>
      <div className="min-w-0">
        <p className="font-display text-section leading-tight font-medium text-ink wrap-anywhere">
          {importName(held)}
        </p>
        <p className="mt-0.5 text-label text-mute">
          {held.title.trim() ? `${where} · ` : ""}
          {waiting.length > 0
            ? `${importCounts(waiting)} waiting`
            : "Everything is on the page"}
        </p>
      </div>
    </div>
  );
}

/** Progress draws one mark per piece shown this session, filling in as each one reaches the page. */
function Progress({ held }: { held: ShownImport }) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const done = Object.keys(held.placed).length;
  if (done === 0) return null;
  return (
    <motion.div
      animate={{ opacity: 1, height: "auto" }}
      className="flex flex-col gap-2"
      initial={still ? false : { opacity: 0, height: 0 }}
    >
      <div aria-hidden="true" className="flex h-1.5 gap-[3px]">
        {held.pieces.map((piece) => (
          <span
            className="relative min-w-[3px] flex-1 overflow-hidden rounded-full bg-rule/60"
            key={piece.id}
          >
            <motion.span
              animate={{ scaleX: piece.id in held.placed ? 1 : 0 }}
              className="absolute inset-0 origin-left rounded-full bg-action"
              initial={false}
              transition={{ type: "spring", stiffness: 200, damping: 26 }}
            />
            {shelf.busy === piece.id || shelf.busy === held.id ? (
              <span className="absolute inset-0 animate-pulse bg-accent/60 motion-reduce:animate-none" />
            ) : null}
          </span>
        ))}
      </div>
      <p className="text-label text-mute">
        <span className="font-medium text-ink">
          <RollingNumber value={done} />
        </span>{" "}
        of {held.pieces.length} placed
      </p>
    </motion.div>
  );
}

function ImportCard({ held }: { held: ShownImport }) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const sections = held.pieces.filter((piece) => piece.kind === "section");
  const pictures = held.pieces.filter((piece) => piece.kind === "picture");
  const placeable = held.pieces.some(
    (piece) =>
      !(piece.id in held.placed) &&
      (piece.kind === "section" || piece.media !== undefined),
  );

  return (
    <motion.section
      animate={{ opacity: 1 }}
      aria-label={importName(held)}
      className="flex flex-col gap-5"
      exit={{ opacity: 0 }}
      initial={{ opacity: 0 }}
      layout={still ? false : "position"}
    >
      <motion.header
        className="flex flex-col gap-4"
        layoutId={still ? undefined : `import-${held.id}`}
        transition={SPRING}
      >
        <ImportHead held={held} />
        <Progress held={held} />
        <div className="flex flex-wrap items-center gap-1">
          {placeable ? (
            <Button
              disabled={shelf.busy !== null}
              loading={shelf.busy === held.id}
              onClick={() => shelf.placeAll(held)}
              variant="primary"
            >
              <Rows3 aria-hidden="true" />
              Place all as blocks
            </Button>
          ) : null}
          <LetGoOfAll held={held} />
        </div>
      </motion.header>

      {sections.length > 0 ? (
        <ol className="m-0 flex list-none flex-col p-0">
          <AnimatePresence initial={false}>
            {sections.map((piece, index) => (
              <SectionRow
                key={piece.id}
                last={index === sections.length - 1}
                piece={piece}
                placedIn={held.placed[piece.id]}
              />
            ))}
          </AnimatePresence>
        </ol>
      ) : null}

      {pictures.length > 0 ? (
        <div className="flex flex-col gap-3">
          <p className="text-meta font-medium text-ink">Pictures</p>
          <ul className="m-0 -mx-1 flex list-none gap-3 overflow-x-auto p-1 pb-2">
            <AnimatePresence initial={false}>
              {pictures.map((piece) => (
                <PictureTile
                  key={piece.id}
                  piece={piece}
                  placedIn={held.placed[piece.id]}
                />
              ))}
            </AnimatePresence>
          </ul>
        </div>
      ) : null}
    </motion.section>
  );
}

function LetGoOfAll({ held }: { held: ShownImport }) {
  const shelf = useShelf();
  const still = useReducedMotion();
  const [asking, setAsking] = useState(false);
  const count = held.pieces.filter(
    (piece) => !(piece.id in held.placed),
  ).length;

  useEffect(() => {
    if (!asking) return;
    const timer = window.setTimeout(() => setAsking(false), CONFIRM_MS);
    return () => window.clearTimeout(timer);
  }, [asking]);

  if (count === 0) return null;
  const swap = {
    animate: { opacity: 1, x: 0, filter: "blur(0px)" },
    exit: { opacity: 0, x: still ? 0 : 8, filter: "blur(2px)" },
    initial: { opacity: 0, x: still ? 0 : 8, filter: "blur(2px)" },
    transition: still ? { duration: 0 } : { duration: 0.18 },
  };

  return (
    <AnimatePresence initial={false} mode="popLayout">
      {asking ? (
        <motion.div className="flex items-center gap-1" key="ask" {...swap}>
          <span className="pr-1 pl-2 text-meta text-ink">
            Let go of {count}?
          </span>
          <Button
            disabled={shelf.busy !== null}
            loading={shelf.busy === held.id}
            onClick={() => shelf.letGoOfImport(held)}
            size="compact"
            variant="stop"
          >
            Let go
          </Button>
          <Button
            onClick={() => setAsking(false)}
            size="compact"
            variant="ghost"
          >
            Keep
          </Button>
        </motion.div>
      ) : (
        <motion.div key="offer" {...swap}>
          <Button
            className="text-mute hover:bg-stop-wash hover:text-stop"
            disabled={shelf.busy !== null}
            onClick={() => setAsking(true)}
            size="compact"
            variant="ghost"
          >
            Let go of all
          </Button>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/** OlderImports rests every import but the newest as a deck, and fans it out into full imports on a tap. */
function OlderImports({ imports }: { imports: ShownImport[] }) {
  const still = useReducedMotion();
  const [fanned, setFanned] = useState(false);
  const shown = imports.slice(0, STACKED);
  const noun = imports.length === 1 ? "import" : "imports";

  if (fanned) {
    return (
      <div className="flex flex-col gap-10">
        {imports.map((held) => (
          <ImportCard held={held} key={held.id} />
        ))}
        <Button
          className="self-start"
          onClick={() => setFanned(false)}
          size="compact"
          variant="ghost"
        >
          Hide older imports
        </Button>
      </div>
    );
  }
  return (
    <button
      aria-expanded={false}
      className="group relative grid pb-4 text-left outline-offset-3"
      onClick={() => setFanned(true)}
      type="button"
    >
      {shown.map((held, index) => (
        <motion.div
          animate={{
            y: index * 8,
            scale: 1 - index * 0.04,
            opacity: 1 - index * 0.3,
          }}
          className="flex items-center gap-3 rounded-plate bg-deep px-4 py-3.5 [grid-area:1/1] group-hover:bg-rule/45"
          key={held.id}
          layoutId={still ? undefined : `import-${held.id}`}
          style={{ zIndex: STACKED - index, originY: 1 }}
          transition={SPRING}
        >
          {index === 0 ? (
            <>
              <Layers
                aria-hidden="true"
                className="size-4.5 shrink-0 text-accent"
              />
              <span className="min-w-0 flex-1 truncate font-ui text-ui font-medium text-ink">
                {importName(held)}
              </span>
              <span className="shrink-0 text-meta font-medium text-accent">
                Show {imports.length} older {noun}
              </span>
            </>
          ) : (
            <span aria-hidden="true" className="invisible text-ui">
              .
            </span>
          )}
        </motion.div>
      ))}
    </button>
  );
}
