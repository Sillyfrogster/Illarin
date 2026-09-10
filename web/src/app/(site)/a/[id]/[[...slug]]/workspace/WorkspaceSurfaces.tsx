"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ShieldAlert } from "lucide-react";
import { useEffect, useState } from "react";
import { WorkspaceRail } from "@/components/workspace/WorkspaceRail";
import type {
  AssetDetail,
  AssetElement,
  AssetImage,
  ReadinessItem,
} from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import type { PageTarget } from "@/lib/readiness";
import { DeleteControl } from "../DeleteControl";
import { DiscoveryControl } from "../DiscoveryControl";
import { ElementFields, elementHint } from "../ElementEditors";
import { PreservedPanel } from "../PreservedPanel";
import { RecordedPromptsPanel } from "../RecordedPromptsPanel";
import { SealedPanel } from "../SealedPanel";
import {
  elementSealsAPrompt,
  NO_ALLOWED_APP,
  SealedPolicy,
} from "../SealedPolicy";
import { UnsealConfirmation } from "../UnsealConfirmation";
import { WithholdControl } from "../WithholdControl";
import { BlockCatalog } from "./BlockCatalog";
import { type Destination, destinationsIn, JumpPalette } from "./JumpPalette";
import { PublicationRail } from "./PublicationRail";
import { RemoveBlock } from "./RemoveBlock";
import { firstCursor } from "./save";
import { useWorkspace } from "./state";
import { WorkspaceDock } from "./WorkspaceDock";

export type WorkspaceSurfacesProps = {
  creator: string;
  discovery: AssetDetail["discovery"];
  hasOriginal: boolean;
  images: AssetImage[];
  kind: string;
  readiness?: ReadinessItem[];
  sealedBlocks?: number;
  sealsPrompts: boolean;
  unpublishedChanges: boolean;
  withheld: boolean;
};

/** The dock, the rail and the palette: everything the workspace puts beside the page. */
export function WorkspaceSurfaces(props: WorkspaceSurfacesProps) {
  const workspace = useWorkspace();
  const { account } = useAuth();
  const reduced = useReducedMotion();
  const [jumping, setJumping] = useState(false);
  const canWithhold = Boolean(
    account?.role === "admin" && !workspace.isDraft && !props.withheld,
  );
  /** Leaving the workspace for the reader's view keeps a way back where the creator stands. */
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
        target.where === "name" ? "asset-name" : "adult-content-answer",
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
      {!workspace.isOwner && canWithhold && workspace.pane === null ? (
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
          detail={detail(props, workspace.isDraft, workspace.saveState)}
          onJump={() => setJumping(true)}
          publicationLabel={workspace.isDraft ? "Publish" : "Review update"}
        />
      ) : null}

      <AnimatePresence>
        {pane?.kind === "conflict" ? (
          <WorkspaceRail
            description="Your writing is still on the page. Copy anything worth keeping, then reload to work from the newer copy."
            key="conflict"
            title="A newer working copy exists"
            tone="stop"
          >
            <div className="flex flex-col gap-5">
              <p className="text-ui text-mute">
                This asset was saved somewhere else, so nothing here can be
                saved until you catch up.
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
              {elementSealsAPrompt(element) ? (
                <SealedPolicy
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
                assetId={workspace.assetId}
                blockId={edited.id}
                element={element}
                images={props.images}
              />
            </div>
          </WorkspaceRail>
        ) : null}

        {pane?.kind === "catalog" ? (
          <WorkspaceRail
            description="Blocks are grouped by where their content ends up. Nothing here is a decision you have to make now."
            key="catalog"
            onClose={workspace.closePane}
            title="Add a block"
          >
            <BlockCatalog />
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

        {pane?.kind === "publication" ? (
          <WorkspaceRail
            description="Nothing here reaches readers until you publish. Content and notes stay private in the meantime."
            key="publication"
            onClose={workspace.closePane}
            title="Publication"
          >
            <PublicationRail
              kind={props.kind}
              onGo={goToPage}
              readiness={props.readiness}
              unpublishedChanges={props.unpublishedChanges}
            />
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
                <DiscoveryControl
                  assetId={workspace.assetId}
                  frozen={props.withheld}
                  initialDiscovery={props.discovery}
                />
              ) : null}
              {workspace.isOwner && props.hasOriginal ? (
                <PreservedPanel assetId={workspace.assetId} />
              ) : null}
              {workspace.isOwner && !workspace.isDraft && props.sealsPrompts ? (
                <RecordedPromptsPanel assetId={workspace.assetId} />
              ) : null}
              {workspace.isOwner && props.sealedBlocks ? (
                <SealedPanel
                  assetId={workspace.assetId}
                  count={props.sealedBlocks}
                />
              ) : null}
              {workspace.isOwner ? (
                <DeleteControl
                  assetId={workspace.assetId}
                  creator={props.creator}
                  frozen={props.withheld}
                  isDraft={workspace.isDraft}
                  kind={props.kind}
                />
              ) : null}
              {canWithhold ? (
                <WithholdControl
                  assetId={workspace.assetId}
                  creator={props.creator}
                />
              ) : null}
            </div>
          </WorkspaceRail>
        ) : null}
      </AnimatePresence>

      {workspace.unsealing ? (
        <UnsealConfirmation
          keepsASeal={workspace.unsealing.keepsASeal}
          onExpose={workspace.confirmUnseal}
          onKeepSealed={workspace.cancelUnseal}
          pending={workspace.busy}
          prompts={workspace.unsealing.prompts}
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

/** The rail remembers which item of a collection is open, so closing it does not lose the creator's place. */
function Fields({
  assetId,
  blockId,
  element,
  images,
}: {
  assetId: string;
  blockId: string;
  element: AssetElement;
  images: AssetImage[];
}) {
  const workspace = useWorkspace();
  return (
    <ElementFields
      assetId={assetId}
      chosen={workspace.chosenItems[element.id] ?? null}
      element={element}
      images={images}
      onChange={(next) => workspace.writeElement(blockId, next)}
      onChoose={(key) => workspace.chooseItem(element.id, key)}
      onImageAdded={() => workspace.say("Picture added.")}
      pending={workspace.busy}
    />
  );
}

/** The light that crosses the page when writing begins. */
function ActivationSweep() {
  const workspace = useWorkspace();
  const reduced = useReducedMotion();
  if (reduced || !workspace.editing || workspace.sweep === 0) return null;
  return (
    <div className="pointer-events-none fixed inset-0 z-20 overflow-hidden">
      <motion.div
        animate={{ left: "110vw" }}
        className="absolute inset-y-[-10%] w-[34vw] min-w-60 bg-[linear-gradient(90deg,transparent,var(--color-accent),transparent)] opacity-25 blur-[14px]"
        initial={{ left: "-40vw" }}
        key={workspace.sweep}
        transition={{ duration: 0.95, ease: [0.4, 0, 0.2, 1] }}
      />
    </div>
  );
}

function detail(
  props: WorkspaceSurfacesProps,
  isDraft: boolean,
  state: string,
): string {
  if (state === "failed") return "Nothing was lost. Try saving again.";
  if (isDraft) return "Only you can open this page.";
  if (props.unpublishedChanges || state === "unsaved") {
    return "Readers do not have your changes yet.";
  }
  return "Readers have everything on this page.";
}
