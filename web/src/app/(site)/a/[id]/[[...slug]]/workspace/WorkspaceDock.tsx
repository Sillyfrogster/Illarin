"use client";

import { motion, useReducedMotion } from "framer-motion";
import {
  ChevronDown,
  EyeOff,
  Globe,
  Images,
  Link2,
  LockKeyhole,
  PencilLine,
  Upload,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { VisibilityItems } from "@/components/work/VisibilityItems";
import {
  Dock,
  DockAction,
  DockTool,
  TOOL,
  WORDED_TOOL,
} from "@/components/workspace/Dock";
import type { WorkDetail } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import type { SaveState } from "./state";
import { useWorkspace } from "./state";

const STATUS: Record<SaveState, string> = {
  failed: "Not saved",
  private: "Drafted changes",
  published: "Published",
  saving: "Saving",
  unsaved: "Saving soon",
};

export const EDIT_CONTROL = "work-edit-control";

const SHOWN = {
  draft: { icon: EyeOff, label: "Draft" },
  listed: { icon: Globe, label: "Public" },
  unlisted: { icon: Link2, label: "Unlisted" },
};

export function WorkspaceDock({
  detail,
  takenDown,
  typeName,
  visibility,
  waiting = 0,
}: {
  detail: string;
  takenDown: boolean;
  typeName: string;
  visibility: WorkDetail["visibility"];
  waiting?: number;
}) {
  const workspace = useWorkspace();
  const holdsPrompts = workspace.blocks.some((block) =>
    block.elements.some((element) => element.type === "prompt_list"),
  );

  return (
    <Dock
      actions={
        <>
          <DockAction onClick={workspace.stopEditing}>Done</DockAction>
          {workspace.saveState === "failed" ? (
            <DockAction onClick={workspace.save}>Retry</DockAction>
          ) : null}
          <DockAction
            disabled={
              workspace.busy ||
              workspace.dirty ||
              workspace.arrangement.busy ||
              (!workspace.isDraft && !workspace.unpublishedChanges)
            }
            onClick={() => workspace.openPane({ kind: "publication" })}
            strong
          >
            Publish
          </DockAction>
        </>
      }
      detail={detail}
      layoutId={EDIT_CONTROL}
      railOpen={workspace.pane !== null}
      state={workspace.saveState}
      tools={
        <>
          <VisibilityMenu
            takenDown={takenDown}
            typeName={typeName}
            visibility={visibility}
          />
          {!workspace.isDraft ? (
            <DockTool
              active={workspace.pane?.kind === "replacement"}
              icon={Upload}
              label="Upload a new version"
              onClick={() => workspace.openPane({ kind: "replacement" })}
              worded
            />
          ) : null}
          {waiting > 0 ? (
            <DockTool
              active={workspace.pane?.kind === "found-images"}
              count={waiting}
              icon={Images}
              label={`${waiting} found ${waiting === 1 ? "image" : "images"} waiting`}
              onClick={() => workspace.openPane({ kind: "found-images" })}
            />
          ) : null}
          {holdsPrompts ? (
            <DockTool
              active={workspace.pane?.kind === "private-prompts"}
              icon={LockKeyhole}
              label="Private prompts"
              onClick={() => workspace.openPane({ kind: "private-prompts" })}
            />
          ) : null}
        </>
      }
      words={STATUS[workspace.saveState]}
    />
  );
}

function VisibilityMenu({
  takenDown,
  typeName,
  visibility,
}: {
  takenDown: boolean;
  typeName: string;
  visibility: WorkDetail["visibility"];
}) {
  const workspace = useWorkspace();
  const shown = SHOWN[workspace.isDraft ? "draft" : visibility];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={`Visibility: ${shown.label}`}
          className={cn(TOOL, WORDED_TOOL)}
          type="button"
        >
          <shown.icon aria-hidden="true" size={18} />
          {shown.label}
          <ChevronDown aria-hidden="true" size={14} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center" side="top">
        <VisibilityItems
          frozen={takenDown}
          initialVisibility={visibility}
          isDraft={workspace.isDraft}
          key={visibility}
          typeName={typeName}
          workId={workspace.workId}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** EditToggle sits where the dock opens and turns into it. */
export function EditToggle({ typeName }: { typeName: string }) {
  const workspace = useWorkspace();
  const reduced = useReducedMotion();
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-4 md:pb-7">
      <motion.button
        className="pointer-events-auto inline-flex min-h-13 items-center gap-2.5 rounded-plate bg-ink py-2 pr-6 pl-5 text-ui font-medium text-field shadow-popover outline-offset-3 hover:bg-ink/90"
        initial={reduced ? false : { opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        layoutId={reduced ? undefined : EDIT_CONTROL}
        onClick={workspace.startEditing}
        transition={{ duration: reduced ? 0 : 0.45, ease: [0.22, 1, 0.36, 1] }}
        type="button"
      >
        <PencilLine aria-hidden="true" size={17} />
        Edit your {typeName}
      </motion.button>
    </div>
  );
}
