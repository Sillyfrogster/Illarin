"use client";

import {
  AnimatePresence,
  MotionConfig,
  motion,
  useReducedMotion,
} from "framer-motion";
import {
  BookOpen,
  Command,
  FileUp,
  FlaskConical,
  LockKeyhole,
  Moon,
  Pencil,
  Sun,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { AssetPage, itemKey, proseKey } from "./asset-page";
import {
  type Asset,
  changedElementIds,
  changesBetween,
  publicationIssues,
} from "./data";
import { Dock, type SaveState } from "./dock";
import { destinationsFor, JumpPalette } from "./jump";
import { ChangeList, ReplacementReview, UpdateReview } from "./review";
import {
  initialSessions,
  type Pane,
  type Session,
  sampleReplacement,
} from "./session";
import {
  Button,
  Field,
  Notice,
  PortalTargetProvider,
  Rail,
  Select,
  SpectralButton,
} from "./ui";

const STATES: [string, string][] = [
  ["conflict", "Stale working copy"],
  ["save-failure", "Next save fails"],
  ["validation", "Validation failure"],
  ["ready", "Finish media processing"],
  ["wrong-kind", "Wrong-kind replacement"],
];

export function WorkspacePrototype() {
  const reduced = useReducedMotion();
  const [sessions, setSessions] = useState(initialSessions);
  const [kind, setKind] = useState<Asset["kind"]>("character");
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [busy, setBusy] = useState(false);
  const [readOnly, setReadOnly] = useState(false);
  const [failSave, setFailSave] = useState(false);
  const [failValidation, setFailValidation] = useState(false);
  const [issues, setIssues] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [replacementChoice, setReplacementChoice] = useState("");
  const [jumpOpen, setJumpOpen] = useState(false);
  const [sweep, setSweep] = useState(0);
  const [blading, setBlading] = useState(false);
  const [root, setRoot] = useState<HTMLElement | null>(null);
  const editTrigger = useRef<HTMLButtonElement>(null);
  const stateMenu = useRef<HTMLDetailsElement>(null);
  const actionMenu = useRef<HTMLDetailsElement>(null);

  const session = sessions[kind];
  const { draft, notes, cursor, editing, pane } = session;
  const changes = changesBetween(session.published, draft);
  const changed = changedElementIds(session.published, draft);
  const dirty =
    JSON.stringify(draft) !== JSON.stringify(session.saved) ||
    JSON.stringify(notes) !== JSON.stringify(session.savedNotes);

  const patch = useCallback(
    (update: Partial<Session>) =>
      setSessions((current) => ({
        ...current,
        [kind]: { ...current[kind], ...update },
      })),
    [kind],
  );

  function updateDraft(asset: Asset) {
    if (readOnly || busy) return;
    patch({ draft: asset, reviewed: undefined });
    setIssues([]);
  }

  const setCursor = useCallback(
    (key: string | null) => patch({ cursor: key }),
    [patch],
  );

  function startEditing() {
    patch({ editing: true });
    if (!reduced) {
      setSweep((n) => n + 1);
      setBlading(true);
    }
  }
  function stopEditing() {
    patch({ editing: false, cursor: null, pane: undefined });
    requestAnimationFrame(() =>
      editTrigger.current?.focus({ preventScroll: true }),
    );
  }
  function openPane(next: Pane) {
    if (actionMenu.current) actionMenu.current.open = false;
    patch({ pane: next, cursor: null });
    setMessage("");
  }
  function closePane() {
    patch({ pane: undefined });
  }

  function checkConflict(revision = session.baseRevision) {
    if (revision === session.serverRevision) return false;
    patch({ conflict: true, reviewed: undefined, cursor: null });
    return true;
  }

  async function save() {
    if (readOnly || busy || checkConflict()) return;
    setBusy(true);
    setMessage("");
    await new Promise((resolve) => setTimeout(resolve, 450));
    if (failSave) {
      setFailSave(false);
      setMessage(
        "The save failed. Your writing is still here — choose Save to try again.",
      );
    } else {
      patch({
        saved: draft,
        savedNotes: notes,
        baseRevision: session.serverRevision + 1,
        serverRevision: session.serverRevision + 1,
        reviewed: undefined,
      });
      setMessage("Saved privately. Readers still have the published version.");
    }
    setBusy(false);
  }

  function openReview() {
    if (readOnly || busy || checkConflict(session.reviewed?.revision)) return;
    setIssues([]);
    openPane("update");
  }

  function checkUpdate() {
    if (readOnly || busy || checkConflict(session.reviewed?.revision)) return;
    const problems = publicationIssues(draft, notes, changes);
    if (failValidation)
      problems.push(
        "The replacement media is still processing. Use Finish media processing in Try a state, then check again.",
      );
    setIssues(problems);
    if (problems.length) {
      patch({ reviewed: undefined });
      return;
    }
    patch({
      reviewed: { asset: draft, notes, revision: session.serverRevision },
    });
  }

  function publish() {
    const candidate = session.reviewed;
    if (!candidate || readOnly || busy || checkConflict(candidate.revision))
      return;
    patch({
      published: candidate.asset,
      saved: candidate.asset,
      draft: candidate.asset,
      savedNotes: { summary: "", notes: "" },
      notes: { summary: "", notes: "" },
      publishedSummary: candidate.notes.summary,
      number: session.number + 1,
      reviewed: undefined,
      baseRevision: session.serverRevision + 1,
      serverRevision: session.serverRevision + 1,
      pane: undefined,
    });
    setMessage(
      `Update ${session.number + 1} published in the demo. Nothing was sent to a server.`,
    );
  }

  function tryState(state: string) {
    if (stateMenu.current) stateMenu.current.open = false;
    setIssues([]);
    if (state === "conflict") {
      patch({
        newer: {
          ...session.saved,
          blurb:
            "A newer private edit from another browser. Compare it with your own work before choosing which copy to keep.",
        },
        serverRevision: session.serverRevision + 1,
        reviewed: undefined,
      });
      setMessage(
        "Another browser saved a newer copy. Try saving or publishing your work.",
      );
    } else if (state === "save-failure") {
      setFailSave(true);
      setMessage("The next private save will fail once. Your writing is kept.");
    } else if (state === "validation") {
      setFailValidation(true);
      patch({ reviewed: undefined });
      setMessage(
        "Media is now still processing. Update review will explain what needs attention.",
      );
    } else if (state === "ready") {
      setFailValidation(false);
      setMessage("Media processing finished. Check the update again.");
    } else if (state === "permission") {
      setReadOnly((current) => !current);
      patch({ reviewed: undefined, cursor: null });
      setMessage("");
    } else if (state === "wrong-kind") {
      setMessage(
        `Replacement refused: that file contains a ${kind === "character" ? "lorebook" : "character"}. An asset’s kind cannot change. Your working copy and public version are unchanged.`,
      );
    }
  }

  function openReplacement() {
    if (readOnly || busy) return;
    setReplacementChoice("");
    patch({
      replacement: {
        asset: sampleReplacement(draft),
        revision: session.baseRevision,
      },
      pane: "replacement",
      cursor: null,
      reviewed: undefined,
    });
    if (actionMenu.current) actionMenu.current.open = false;
    setMessage("");
  }

  const replacement = session.replacement && {
    ...session.replacement.asset,
    blocks: session.replacement.asset.blocks.map((block) => ({
      ...block,
      elements: block.elements.filter(
        (element) =>
          replacementChoice !== "remove" || element.role !== "group_greetings",
      ),
    })),
  };

  function acceptReplacement() {
    if (
      readOnly ||
      busy ||
      !replacement ||
      !session.replacement ||
      checkConflict(session.replacement.revision)
    )
      return;
    if (kind === "character" && !replacementChoice) {
      setMessage(
        "Choose whether to keep or remove group-only greetings before accepting this replacement.",
      );
      return;
    }
    patch({
      draft: replacement,
      saved: replacement,
      savedNotes: notes,
      baseRevision: session.serverRevision + 1,
      serverRevision: session.serverRevision + 1,
      replacement: undefined,
      pane: "update",
    });
    setMessage(
      "Replacement accepted into private work. Review the complete update before publication.",
    );
  }

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        if (sessions[kind].editing) setJumpOpen((open) => !open);
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [kind, sessions]);

  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(() => setMessage(""), 7000);
    return () => clearTimeout(timer);
  }, [message]);

  const saveState: SaveState = busy
    ? "saving"
    : dirty
      ? "unsaved"
      : changes.length || notes.summary || notes.notes
        ? "private"
        : "published";
  const detail = changes.length
    ? `${changes.length} ${changes.length === 1 ? "change" : "changes"} since update ${session.number}`
    : `Update ${session.number} is live`;

  const railKey = session.conflict
    ? "conflict"
    : pane === "replacement" && replacement
      ? "replacement"
      : (pane ?? null);
  const shift = railKey
    ? "ws:lg:pr-[28rem] ws:transition-[padding] ws:duration-500 ws:ease-[cubic-bezier(0.22,1,0.36,1)] ws:motion-reduce:transition-none"
    : "ws:transition-[padding] ws:duration-500 ws:ease-[cubic-bezier(0.22,1,0.36,1)] ws:motion-reduce:transition-none";

  const destinations = destinationsFor(
    draft,
    (_blockId, elementId, itemId) =>
      setCursor(
        itemId ? itemKey(elementId, itemId, "text") : proseKey(elementId),
      ),
    () => setCursor("name"),
  );

  return (
    <MotionConfig reducedMotion="user">
      <div
        ref={setRoot}
        data-workspace
        data-theme={theme}
        data-mode={editing ? "edit" : "read"}
        className="ws:min-h-dvh ws:bg-paper ws:text-ink ws:transition-colors ws:duration-700 ws:motion-reduce:transition-none"
      >
        <PortalTargetProvider value={root}>
          <div className={shift}>
            <header className="ws:relative ws:z-30 ws:mx-auto ws:flex ws:max-w-[86rem] ws:items-center ws:justify-between ws:gap-4 ws:px-5 ws:pt-6 ws:md:px-10">
              <div className="ws:flex ws:min-w-0 ws:items-baseline ws:gap-7">
                <span className="ws:font-display ws:text-3xl ws:leading-none">
                  Illarin.
                </span>
                <nav className="ws:hidden ws:gap-6 ws:text-sm ws:text-mute ws:sm:flex">
                  <span>Browse</span>
                  <span>Publish</span>
                </nav>
              </div>
              <div className="w-glass ws:flex ws:items-center ws:gap-1 ws:rounded-full ws:p-1 ws:pl-3.5">
                <span className="ws:hidden ws:text-[0.6875rem] ws:font-bold ws:tracking-[0.16em] ws:text-mute ws:uppercase ws:md:inline">
                  Prototype
                </span>
                <Select
                  aria-label="Synthetic asset"
                  value={kind}
                  disabled={busy}
                  onChange={(event) => {
                    setKind(event.target.value as Asset["kind"]);
                    setMessage("");
                    setIssues([]);
                    setReplacementChoice("");
                  }}
                  className="ws:min-h-9 ws:w-auto ws:bg-transparent ws:py-0 ws:text-xs ws:shadow-none"
                >
                  <option value="character">Character</option>
                  <option value="lorebook">Lorebook</option>
                </Select>
                <details ref={stateMenu} className="ws:relative">
                  <summary className="ws:flex ws:min-h-9 ws:cursor-pointer ws:list-none ws:items-center ws:gap-1.5 ws:rounded-full ws:px-3 ws:text-xs ws:font-semibold">
                    <FlaskConical className="ws:size-3.5" />
                    <span className="ws:hidden ws:sm:inline">Try a state</span>
                  </summary>
                  <div className="w-glass ws:absolute ws:top-full ws:right-0 ws:z-50 ws:mt-2 ws:w-64 ws:rounded-2xl ws:p-1.5">
                    {[
                      ...STATES,
                      [
                        "permission",
                        readOnly
                          ? "Restore owner access"
                          : "Read-only permission",
                      ] as [string, string],
                    ].map(([value, label]) => (
                      <Button
                        key={value}
                        size="small"
                        disabled={busy}
                        className="ws:w-full ws:justify-start ws:rounded-xl"
                        onClick={() => tryState(value)}
                      >
                        {label}
                      </Button>
                    ))}
                  </div>
                </details>
                <Button
                  size="icon"
                  className="ws:size-9"
                  aria-label={`Use the ${theme === "light" ? "dark" : "light"} theme`}
                  onClick={() => setTheme(theme === "light" ? "dark" : "light")}
                >
                  {theme === "light" ? <Moon /> : <Sun />}
                </Button>
              </div>
            </header>

            <AssetPage
              asset={draft}
              live={editing && !readOnly}
              cursor={cursor}
              setCursor={setCursor}
              theme={theme}
              changed={changed}
              update={updateDraft}
              note={setMessage}
              action={
                editing ? (
                  <p className="ws:text-sm ws:text-mute">
                    {readOnly
                      ? "Read-only. Restore owner access in Try a state to write."
                      : "Click any writing on the page to edit it where it sits."}
                  </p>
                ) : (
                  <>
                    <Button
                      ref={editTrigger}
                      variant="primary"
                      className="ws:min-h-12 ws:px-6"
                      onClick={startEditing}
                    >
                      <Pencil />
                      Edit this page
                    </Button>
                    <Button variant="outline" className="ws:min-h-12">
                      Download
                    </Button>
                    {changes.length > 0 && (
                      <span className="ws:inline-flex ws:items-center ws:gap-2.5 ws:rounded-full ws:bg-amber-field ws:px-4 ws:py-2.5 ws:text-sm">
                        <span className="ws:size-2 ws:shrink-0 ws:rounded-full ws:bg-amber" />
                        Your private working copy. Readers still have update{" "}
                        {session.number}.
                      </span>
                    )}
                  </>
                )
              }
            />
          </div>

          <AnimatePresence>
            {railKey === "conflict" && (
              <Rail
                key="conflict"
                tone="critical"
                title="A newer working copy is available"
                description="Your writing is kept. Compare the newer private copy with your work, then choose which one continues."
              >
                <div className="ws:space-y-7">
                  {session.newer && (
                    <ChangeList
                      changes={changesBetween(session.newer, draft)}
                    />
                  )}
                  <div className="ws:flex ws:flex-wrap ws:gap-3">
                    <SpectralButton
                      disabled={readOnly || busy}
                      onClick={() => {
                        patch({
                          baseRevision: session.serverRevision,
                          conflict: false,
                          newer: undefined,
                          replacement: undefined,
                          reviewed: undefined,
                        });
                        setMessage(
                          "Your version is kept. Save it privately when you are ready.",
                        );
                      }}
                    >
                      Continue with my version
                    </SpectralButton>
                    <Button
                      variant="outline"
                      disabled={readOnly || busy}
                      onClick={() => {
                        if (session.newer)
                          patch({
                            draft: session.newer,
                            saved: session.newer,
                            baseRevision: session.serverRevision,
                            conflict: false,
                            newer: undefined,
                            reviewed: undefined,
                            replacement: undefined,
                          });
                        setMessage(
                          "The newer copy is loaded. Your notes are kept.",
                        );
                      }}
                    >
                      Use the newer copy
                    </Button>
                  </div>
                </div>
              </Rail>
            )}

            {railKey === "replacement" && replacement && (
              <Rail
                key="replacement"
                tone="amber"
                title="Review the replacement file"
                description="A replacement rewrites the matching content in your private working copy. Nothing reaches readers until you publish."
                onClose={() =>
                  patch({ pane: undefined, replacement: undefined })
                }
              >
                <ReplacementReview
                  changes={changesBetween(draft, replacement)}
                  choice={replacementChoice}
                  setChoice={setReplacementChoice}
                  accept={acceptReplacement}
                  cancel={() =>
                    patch({ replacement: undefined, pane: undefined })
                  }
                  busy={busy || readOnly}
                  kind={kind}
                />
              </Rail>
            )}

            {railKey === "update" && (
              <Rail
                key="update"
                title="Review your update"
                description="Tell readers what changed. Your page stays open and editable, with every change marked on it."
                onClose={closePane}
              >
                <UpdateReview
                  asset={draft}
                  notes={notes}
                  setNotes={(next) => {
                    patch({ notes: next, reviewed: undefined });
                    setIssues([]);
                  }}
                  setVersion={(version) => updateDraft({ ...draft, version })}
                  changes={changes}
                  reviewed={Boolean(session.reviewed)}
                  issues={issues}
                  readOnly={readOnly}
                  busy={busy}
                  listed={session.listed}
                  check={checkUpdate}
                  publish={publish}
                  keepEditing={() => {
                    patch({ reviewed: undefined, pane: undefined });
                    setIssues([]);
                  }}
                />
              </Rail>
            )}

            {railKey === "access" && (
              <Rail
                key="access"
                title="Access and published state"
                description="Access changes apply immediately to the asset and its history. They do not wait for a private save or a publication."
                onClose={closePane}
              >
                <div className="ws:space-y-9">
                  <Field label="Discovery">
                    <Select
                      disabled={readOnly || busy}
                      className="ws:max-w-md"
                      value={session.listed ? "listed" : "unlisted"}
                      onChange={(event) => {
                        patch({
                          listed: event.target.value === "listed",
                          reviewed: undefined,
                        });
                        setMessage(
                          "Discovery changed immediately. Content and notes stay private.",
                        );
                      }}
                    >
                      <option value="listed">Listed in Browse</option>
                      <option value="unlisted">
                        Unlisted · direct links work
                      </option>
                    </Select>
                  </Field>
                  <section className="ws:min-w-0 ws:space-y-3">
                    <p className="ws:text-sm ws:text-mute">
                      Published update {session.number}
                    </p>
                    <h3 className="ws:font-display ws:text-3xl ws:font-medium ws:wrap-anywhere">
                      {session.published.name}
                    </h3>
                    <p className="ws:wrap-anywhere">
                      {session.published.blurb}
                    </p>
                    <p className="ws:text-sm ws:text-mute">
                      {session.publishedSummary}
                    </p>
                    <details>
                      <summary className="ws:min-h-11 ws:cursor-pointer ws:py-3 ws:text-sm ws:font-semibold">
                        Read what readers have now
                      </summary>
                      {session.published.blocks
                        .flatMap((block) => block.elements)
                        .map((element) => (
                          <div key={element.id} className="ws:mt-5">
                            <h4 className="ws:font-semibold">
                              {element.label}
                            </h4>
                            <p className="w-writing ws:max-w-[64ch] ws:whitespace-pre-wrap ws:wrap-anywhere">
                              {element.text ||
                                element.items
                                  .map((item) => `${item.name}\n${item.text}`)
                                  .join("\n\n")}
                            </p>
                          </div>
                        ))}
                    </details>
                  </section>
                </div>
              </Rail>
            )}
          </AnimatePresence>

          {/* The beam that crosses the page when editing starts */}
          <div className="ws:pointer-events-none ws:fixed ws:inset-0 ws:z-20 ws:overflow-hidden">
            <AnimatePresence>
              {blading && (
                <motion.div
                  key={sweep}
                  className="w-blade"
                  initial={{ left: "-40vw" }}
                  animate={{ left: "110vw" }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.95, ease: [0.4, 0, 0.2, 1] }}
                  onAnimationComplete={() => setBlading(false)}
                />
              )}
            </AnimatePresence>
          </div>

          <AnimatePresence>
            {editing && (
              <Dock
                state={saveState}
                detail={detail}
                busy={busy}
                readOnly={readOnly}
                shifted={Boolean(railKey)}
                onSave={() => void save()}
                onReview={openReview}
                reviewLabel={
                  session.reviewed ? "Ready to publish" : "Review update"
                }
                items={[
                  {
                    icon: BookOpen,
                    label: "Reading view",
                    onClick: stopEditing,
                  },
                  {
                    icon: Command,
                    label: "Go to content",
                    onClick: () => setJumpOpen(true),
                  },
                  {
                    icon: FileUp,
                    label: "Review a replacement file",
                    active: pane === "replacement",
                    onClick: openReplacement,
                  },
                  {
                    icon: LockKeyhole,
                    label: "Access and published state",
                    active: pane === "access",
                    onClick: () => openPane("access"),
                  },
                ]}
              />
            )}
          </AnimatePresence>

          <AnimatePresence>
            {message && (
              <motion.output
                initial={{ opacity: 0, y: 14 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 8 }}
                className="w-glass ws:fixed ws:inset-x-4 ws:bottom-28 ws:z-50 ws:mx-auto ws:block ws:max-w-lg ws:rounded-2xl ws:px-5 ws:py-3.5 ws:text-sm ws:leading-6 ws:md:bottom-32"
              >
                {message}
              </motion.output>
            )}
          </AnimatePresence>

          <JumpPalette
            open={jumpOpen}
            onOpenChange={setJumpOpen}
            destinations={destinations}
          />

          {readOnly && (
            <div className="ws:fixed ws:inset-x-4 ws:top-4 ws:z-50 ws:mx-auto ws:max-w-md">
              <Notice tone="amber">
                Read-only permission. Restore owner access in Try a state to
                continue writing.
              </Notice>
            </div>
          )}
        </PortalTargetProvider>
      </div>
    </MotionConfig>
  );
}
