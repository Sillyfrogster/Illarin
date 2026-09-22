"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ShieldAlert } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import type {
  ReadinessItem,
  WorkDetail,
  WorkElement,
  WorkImage,
} from "@/lib/api/query";
import { fetchWaitingReplacement, type UploadOperation } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import type { PageTarget } from "@/lib/readiness";
import { ElementFields, elementHint } from "../ElementEditors";
import { MakePublicConfirmation } from "../MakePublicConfirmation";
import { PreservedPanel } from "../PreservedPanel";
import { PreservedPromptsPanel } from "../PreservedPromptsPanel";
import {
  elementHasPrivatePrompts,
  NO_ALLOWED_APP,
  PrivatePromptsControl,
} from "../PrivatePromptsControl";
import { RecordedPromptsPanel } from "../RecordedPromptsPanel";
import { TakedownControl } from "../TakedownControl";
import { VisibilityControl } from "../VisibilityControl";
import { AddBlock } from "./AddBlock";
import { FoundImagesPanel } from "./FoundImagesPanel";
import { useFoundImages } from "./found-images";
import { type Destination, destinationsIn, JumpPalette } from "./JumpPalette";
import { PublishDialog } from "./PublishDialog";
import { RemoveBlock } from "./RemoveBlock";
import { ReplacementStep } from "./ReplacementStep";
import { firstCursor } from "./save";
import { useWorkspace } from "./state";
import { WorkspaceDock } from "./WorkspaceDock";

export type WorkspaceSurfacesProps = {
  creator: string;
  visibility: WorkDetail["visibility"];
  hasOriginal: boolean;
  images: WorkImage[];
  typeName: string;
  readiness?: ReadinessItem[];
  preservedPrompts?: number;
  hasPrivatePrompts: boolean;
  takenDown: boolean;
};

export function WorkspaceSurfaces(props: WorkspaceSurfacesProps) {
  const workspace = useWorkspace();
  const { account } = useAuth();
  const reduced = useReducedMotion();
  const [jumping, setJumping] = useState(false);
  const foundImages = useFoundImages(workspace.workId, workspace.isOwner);
  const canTakeDown = Boolean(
    account?.role === "admin" && !workspace.isDraft && !props.takenDown,
  );
  const _previewing =
    workspace.isOwner && !workspace.editing && workspace.sweep > 0;

  useEffect(() => {
    if (!workspace.editing) return;
    function onKey(event: KeyboardEvent) {
      if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== "k")
        return;
      event.preventDefault();
      setJumping((open) => !open);
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [workspace.editing]);

  function go(destination: Destination) {
    setJumping(false);
    const block = workspace.blocks.find(
      (item) => item.id === destination.blockId,
    );
    document
      .getElementById(`block-${destination.blockId}`)
      ?.scrollIntoView({ block: "start" });
    if (!block || !destination.elementId) return;
    const element = block.elements.find(
      (item) => item.id === destination.elementId,
    );
    if (!element) return;
    const cursor = firstCursor(element);
    if (cursor) {
      workspace.setCursor(cursor);
      return;
    }
    workspace.openPane({
      blockId: block.id,
      elementId: element.id,
      kind: "element",
    });
  }

  function goToPage(target: PageTarget) {
    workspace.closePane();
    if (target.where === "block") {
      go({ blockId: target.blockId, id: target.blockId, label: "", where: "" });
      return;
    }
    document
      .getElementById(
        target.where === "name" ? "work-name" : "adult-content-answer",
      )
      ?.scrollIntoView({ block: "center" });
    if (target.where === "name") workspace.setCursor("identity:name");
  }

  const pane = workspace.pane;
  const edited =
    pane?.kind === "element"
      ? workspace.blocks.find((block) => block.id === pane.blockId)
      : undefined;
  const element = edited?.elements.find(
    (item) => pane?.kind === "element" && item.id === pane.elementId,
  );
  const removed =
    pane?.kind === "remove"
      ? workspace.blocks.find((block) => block.id === pane.blockId)
      : undefined;

  return (
    <>
      {!workspace.isOwner && canTakeDown && workspace.pane === null ? (
        <button
          className="fixed right-4 bottom-4 z-30 inline-flex min-h-11 items-center gap-2 rounded-control bg-ink px-4 text-meta font-medium text-field shadow-popover outline-offset-3"
          onClick={() => workspace.openPane({ kind: "access" })}
          type="button"
        >
          <ShieldAlert aria-hidden="true" size={16} />
          Staff tools
        </button>
      ) : null}

      {workspace.editing ? <div aria-hidden="true" className="h-28" /> : null}

      {workspace.editing ? (
        <WorkspaceDock
          detail={detail(workspace.isDraft, workspace.saveState)}
          onJump={() => setJumping(true)}
          waiting={foundImages.pictures.length}
        />
      ) : null}

      <AnimatePresence>
        {pane?.kind === "conflict" ? (
          <WorkspaceRail
            description="Your writing is still on the page. Copy anything worth keeping, then reload to work from the newer drafted changes."
            key="conflict"
            title="Newer drafted changes exist"
            tone="stop"
          >
            <div className="flex flex-col gap-5">
              <p className="text-ui text-mute">
                This page was saved in another session. Copy any unsaved text,
                then reload to edit the latest version.
              </p>
              <button
                className="min-h-11 rounded-control bg-action px-5 text-ui font-medium text-on-accent outline-offset-3"
                onClick={() => window.location.reload()}
                type="button"
              >
                Reload the page
              </button>
            </div>
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "element" && edited && element ? (
          <WorkspaceRail
            description={[...element.facts, elementHint(element.type)].join(
              " · ",
            )}
            key="element"
            onClose={workspace.closePane}
            title={element.label || edited.title}
          >
            <div className="flex flex-col gap-5">
              {workspace.message === NO_ALLOWED_APP ? (
                <p
                  className="rounded-control bg-stop-wash p-3 text-meta text-ink"
                  role="alert"
                >
                  {workspace.message}
                </p>
              ) : null}
              {elementHasPrivatePrompts(element) ? (
                <PrivatePromptsControl
                  pending={workspace.busy}
                  policy={{
                    allowedApps: workspace.allowedApps,
                    eligibleApps: workspace.eligibleApps,
                    onChange: workspace.setAllowedApps,
                  }}
                  unanswered={workspace.message === NO_ALLOWED_APP}
                />
              ) : null}
              <Fields
                workId={workspace.workId}
                blockId={edited.id}
                element={element}
                images={props.images}
              />
            </div>
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "add-block" ? (
          <WorkspaceRail
            description="Blocks are grouped by where their content ends up. Nothing here is a decision you have to make now."
            key="add-block"
            onClose={workspace.closePane}
            title="Add a block"
          >
            <AddBlock />
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "remove" && removed ? (
          <WorkspaceRail
            description="Removing a block deletes what it holds. Nothing else on the page changes."
            key="remove"
            onClose={workspace.closePane}
            title={`Remove “${removed.title}”?`}
            tone="stop"
          >
            <RemoveBlock block={removed} />
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "found-images" ? (
          <WorkspaceRail
            description="These came with your upload. Place each one in a block, or remove it."
            key="found-images"
            onClose={workspace.closePane}
            title="Found images"
          >
            <FoundImagesPanel
              onRelease={foundImages.release}
              onReload={foundImages.reload}
              pictures={foundImages.pictures}
            />
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "publication" ? (
          <PublishDialog
            key="publication"
            typeName={props.typeName}
            onGo={(target) =>
              target.where === "replacement"
                ? workspace.openPane({ kind: "replacement" })
                : goToPage(target)
            }
            readiness={props.readiness ?? []}
            unlisted={props.visibility === "unlisted"}
          />
        ) : null}

        {pane?.kind === "replacement" ? (
          <WorkspaceRail
            key="replacement"
            title="Upload a new version"
            onClose={workspace.closePane}
          >
            <Replacement />
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "access" ? (
          <WorkspaceRail
            description="Access changes apply the moment you make them. They do not wait for a save or a publication."
            key="access"
            onClose={workspace.closePane}
            title="Access and published state"
          >
            <div className="flex flex-col gap-7">
              {workspace.isOwner && !workspace.isDraft ? (
                <VisibilityControl
                  workId={workspace.workId}
                  frozen={props.takenDown}
                  initialVisibility={props.visibility}
                  typeName={props.typeName}
                />
              ) : null}
              {workspace.isOwner && props.hasOriginal ? (
                <PreservedPanel workId={workspace.workId} />
              ) : null}
              {workspace.isOwner &&
              !workspace.isDraft &&
              props.hasPrivatePrompts ? (
                <RecordedPromptsPanel workId={workspace.workId} />
              ) : null}
              {workspace.isOwner && props.preservedPrompts ? (
                <PreservedPromptsPanel
                  workId={workspace.workId}
                  count={props.preservedPrompts}
                />
              ) : null}
              {canTakeDown ? (
                <TakedownControl
                  workId={workspace.workId}
                  creator={props.creator}
                  typeName={props.typeName}
                />
              ) : null}
            </div>
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>

      {workspace.makingPublic ? (
        <MakePublicConfirmation
          keepsAPrivatePrompt={workspace.makingPublic.keepsAPrivatePrompt}
          onMakePublic={workspace.confirmMakePublic}
          onKeepPrivate={workspace.cancelMakePublic}
          pending={workspace.busy}
          prompts={workspace.makingPublic.prompts}
        />
      ) : null}

      {jumping ? (
        <JumpPalette
          destinations={destinationsIn(workspace.blocks)}
          onClose={() => setJumping(false)}
          onGo={go}
        />
      ) : null}

      <AnimatePresence>
        {workspace.message && workspace.message !== NO_ALLOWED_APP ? (
          <motion.output
            animate={{ opacity: 1, y: 0 }}
            className="fixed inset-x-4 bottom-28 z-50 mx-auto block max-w-lg rounded-plate bg-plane px-5 py-3.5 text-meta text-ink shadow-popover md:bottom-32"
            exit={{ opacity: 0, y: 8 }}
            initial={reduced ? { opacity: 0 } : { opacity: 0, y: 14 }}
          >
            {workspace.message}
          </motion.output>
        ) : null}
      </AnimatePresence>

      <ActivationSweep />
    </>
  );
}

function Fields({
  workId,
  blockId,
  element,
  images,
}: {
  workId: string;
  blockId: string;
  element: WorkElement;
  images: WorkImage[];
}) {
  const workspace = useWorkspace();
  return (
    <ElementFields
      workId={workId}
      chosen={workspace.chosenItems[element.id] ?? null}
      element={element}
      images={images}
      onChange={(next) => workspace.writeElement(blockId, next)}
      onChoose={(key) => workspace.chooseItem(element.id, key)}
      onImageAdded={() => workspace.say("Picture added.")}
      pending={false}
    />
  );
}

function ActivationSweep() {
  const workspace = useWorkspace();
  const reduced = useReducedMotion();
  if (reduced || !workspace.editing || workspace.sweep === 0) return null;
  return (
    <div className="pointer-events-none fixed inset-0 z-20 overflow-hidden">
      <motion.div
        animate={{ left: "110vw" }}
        className="absolute inset-y-[-10%] w-[34vw] min-w-60 bg-[linear-gradient(90deg,transparent,var(--v-action),transparent)] opacity-25 blur-[14px]"
        initial={{ left: "-40vw" }}
        key={workspace.sweep}
        transition={{ duration: 0.95, ease: [0.4, 0, 0.2, 1] }}
      />
    </div>
  );
}

function detail(isDraft: boolean, state: string): string {
  if (state === "failed")
    return "Your edits are still on this page. Try saving again.";
  if (isDraft) return "Only you can open this page.";
  if (state === "private" || state === "unsaved" || state === "saving") {
    return "Readers do not have your changes yet.";
  }
  return "All changes are published.";
}

function Replacement() {
  const workspace = useWorkspace();
  const router = useRouter();
  const [waiting, setWaiting] = useState<UploadOperation | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    void fetchWaitingReplacement(workspace.workId)
      .then((upload) => {
        if (active) {
          setWaiting(upload);
          setLoaded(true);
        }
      })
      .catch(() => {
        if (active)
          setError(
            "Could not load your upload. Close this panel and try again.",
          );
      });
    return () => {
      active = false;
    };
  }, [workspace.workId]);
  function settled() {
    workspace.closePane();
    router.refresh();
  }
  return error ? (
    <p role="alert" className="text-ui text-stop">
      {error}
    </p>
  ) : loaded ? (
    <ReplacementStep
      onApplied={settled}
      onDiscarded={settled}
      onBack={workspace.closePane}
      waiting={waiting}
      onWaiting={setWaiting}
    />
  ) : (
    <output>Loading…</output>
  );
}
