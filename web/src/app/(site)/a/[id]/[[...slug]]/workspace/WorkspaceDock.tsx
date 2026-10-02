"use client";

import { motion } from "framer-motion";
import {
  Check,
  ChevronDown,
  Ellipsis,
  EyeOff,
  Globe,
  LibraryBig,
  Link2,
  LockKeyhole,
  PencilLine,
  RefreshCw,
  Upload,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { VisibilityItems } from "@/components/work/VisibilityItems";
import { DockTool, SaveStatus } from "@/components/workspace/Dock";
import type { WorkDetail } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import { timing } from "@/lib/timing";
import { AltitudeSwitch } from "./Altitudes";
import { type Pane, useWorkspace } from "./state";

export const EDIT_CONTROL = "work-edit-control";

const SHOWN = {
  draft: { icon: EyeOff, label: "Draft" },
  listed: { icon: Globe, label: "Public" },
  unlisted: { icon: Link2, label: "Unlisted" },
};

type EditProps = {
  detail: string;
  takenDown: boolean;
  typeName: string;
  visibility: WorkDetail["visibility"];
  waiting?: number;
};

type Tool = {
  key: string;
  pane: Pane["kind"];
  icon: typeof Upload;
  label: string;
  words?: string;
  count?: number;
};

function useTools(waiting: number): Tool[] {
  const workspace = useWorkspace();
  const holdsPrompts = workspace.blocks.some((block) =>
    block.elements.some((element) => element.type === "prompt_list"),
  );
  return [
    ...(workspace.isDraft
      ? []
      : [
          {
            icon: Upload,
            key: "replacement",
            label: "Upload a new version",
            pane: "replacement" as const,
            words: "New version",
          },
        ]),
    {
      count: waiting,
      icon: LibraryBig,
      key: "shelf",
      label:
        waiting > 0
          ? `Shelf, ${waiting} ${waiting === 1 ? "piece" : "pieces"} waiting`
          : "Shelf, empty",
      pane: "shelf" as const,
      words: "Shelf",
    },
    ...(holdsPrompts
      ? [
          {
            icon: LockKeyhole,
            key: "private-prompts",
            label: "Private prompts",
            pane: "private-prompts" as const,
            words: "Private prompts",
          },
        ]
      : []),
  ];
}

function Tools({ tools }: { tools: Tool[] }) {
  const workspace = useWorkspace();
  return tools.map((tool) => (
    <DockTool
      active={workspace.pane?.kind === tool.pane}
      count={tool.count}
      icon={tool.icon}
      key={tool.key}
      label={tool.label}
      onClick={() => workspace.openPane({ kind: tool.pane } as Pane)}
      words={tool.words}
    />
  ));
}

/** MoreTools folds the tools into one menu where the bar has no room for them. */
function MoreTools({ tools, done }: { tools: Tool[]; done?: boolean }) {
  const workspace = useWorkspace();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label="More tools" size="icon" variant="ghost">
          <Ellipsis aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top">
        {tools.map((tool) => (
          <DropdownMenuItem
            key={tool.key}
            onSelect={() => workspace.openPane({ kind: tool.pane } as Pane)}
          >
            <tool.icon />
            {tool.words ?? tool.label}
            {tool.count ? (
              <span className="ml-auto text-label text-accent tabular-nums">
                {tool.count}
              </span>
            ) : null}
          </DropdownMenuItem>
        ))}
        {done ? (
          <DropdownMenuItem onSelect={workspace.stopEditing}>
            <Check />
            Done editing
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** PublishButton reads what Publish would do now: publish a draft, publish changes, or nothing because readers already see this. */
function PublishButton() {
  const workspace = useWorkspace();
  const upToDate = !workspace.isDraft && !workspace.unpublishedChanges;
  const waiting =
    workspace.busy || workspace.dirty || workspace.arrangement.busy;
  if (upToDate && !workspace.dirty)
    return (
      <Button className="text-mute" disabled variant="secondary">
        <Check aria-hidden="true" />
        Published
      </Button>
    );
  return (
    <Button
      disabled={waiting}
      onClick={() => workspace.openPane({ kind: "publication" })}
      variant="primary"
    >
      {workspace.isDraft ? "Publish" : "Publish changes"}
    </Button>
  );
}

function TryAgain() {
  const workspace = useWorkspace();
  if (workspace.saveState !== "failed") return null;
  return (
    <Button onClick={workspace.save} size="compact" variant="ghost">
      <RefreshCw aria-hidden="true" />
      Try again
    </Button>
  );
}

/** WorkspaceDock is the arrange-map editor's strip at the bottom: whether the edits are saved, writing or arranging, the tools, and Publish. */
export function WorkspaceDock(props: EditProps) {
  const workspace = useWorkspace();
  const tools = useTools(props.waiting ?? 0);
  return (
    <div
      className={cn(
        "pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-3 transition-[padding] duration-240 ease-wipe md:pb-6",
        workspace.pane !== null && "max-lg:hidden lg:pr-[28rem]",
      )}
    >
      <motion.div
        animate={{ opacity: 1, y: 0 }}
        className="pointer-events-auto flex w-full max-w-[58rem] items-center gap-1.5 rounded-card bg-plane p-1.5 shadow-popover ring-1 ring-ink/8 md:gap-2 md:p-2"
        initial={false}
        layoutId={EDIT_CONTROL}
        transition={timing.settle}
      >
        <div className="min-w-0 flex-1 px-2 md:px-3">
          <SaveStatus
            compact
            detail={props.detail}
            state={workspace.saveState}
          />
        </div>
        <TryAgain />
        <AltitudeSwitch compact />
        <div className="hidden items-center gap-0.5 md:flex">
          <VisibilityMenu {...props} />
          <Tools tools={tools} />
        </div>
        <div className="md:hidden">
          <MoreTools done tools={tools} />
        </div>
        <Button
          className="max-md:hidden"
          onClick={workspace.stopEditing}
          variant="secondary"
        >
          Done
        </Button>
        <PublishButton />
      </motion.div>
    </div>
  );
}

function VisibilityMenu({ takenDown, typeName, visibility }: EditProps) {
  const workspace = useWorkspace();
  const shown = SHOWN[workspace.isDraft ? "draft" : visibility];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={`Who can see it: ${shown.label}`}
          className={cn(
            "inline-flex h-control shrink-0 items-center gap-2 rounded-control px-2.5 text-meta font-medium text-mute transition-colors duration-80 hover:bg-fill hover:text-ink",
            focusRing,
          )}
          type="button"
        >
          <shown.icon aria-hidden="true" className="size-4.5" />
          <span className="max-xl:sr-only">{shown.label}</span>
          <ChevronDown aria-hidden="true" className="size-3.5" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center">
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
  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-30 flex justify-center px-3 pb-4 md:pb-7">
      <motion.button
        animate={{ opacity: 1, y: 0 }}
        className={cn(
          "pointer-events-auto inline-flex h-12 items-center gap-2.5 rounded-card bg-action pr-6 pl-5 text-ui font-medium text-on-accent shadow-popover transition-colors duration-80 hover:bg-action-hover",
          focusRing,
          "focus-visible:ring-offset-1 focus-visible:ring-offset-field",
        )}
        initial={{ opacity: 0, y: 24 }}
        layoutId={EDIT_CONTROL}
        onClick={workspace.startEditing}
        transition={timing.settle}
        type="button"
      >
        <PencilLine aria-hidden="true" className="size-4.5" />
        Edit your {typeName}
      </motion.button>
    </div>
  );
}
