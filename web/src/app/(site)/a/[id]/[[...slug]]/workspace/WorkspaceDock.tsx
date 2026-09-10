"use client";

import { BookOpen, Command, LockKeyhole } from "lucide-react";
import { Dock, DockAction, DockTool } from "@/components/workspace/Dock";
import type { SaveState } from "./state";
import { useWorkspace } from "./state";

const STATUS: Record<SaveState, string> = {
  failed: "Not saved",
  private: "Saved privately",
  published: "Readers have this",
  saving: "Saving",
  unsaved: "Unsaved",
};

export function WorkspaceDock({
  detail,
  onJump,
  publicationLabel,
}: {
  detail: string;
  onJump: () => void;
  publicationLabel: string;
}) {
  const workspace = useWorkspace();

  return (
    <Dock
      actions={
        <>
          <DockAction
            disabled={workspace.busy || !workspace.dirty}
            onClick={workspace.save}
          >
            {workspace.busy ? "Saving…" : "Save"}
          </DockAction>
          <DockAction
            disabled={workspace.busy}
            onClick={() => workspace.openPane({ kind: "publication" })}
            strong
          >
            {publicationLabel}
          </DockAction>
        </>
      }
      detail={detail}
      railOpen={workspace.pane !== null}
      state={workspace.saveState}
      tools={
        <>
          <DockTool
            icon={BookOpen}
            label="Reading view"
            onClick={workspace.stopEditing}
          />
          <DockTool icon={Command} label="Go to content" onClick={onJump} />
          <DockTool
            active={workspace.pane?.kind === "access"}
            icon={LockKeyhole}
            label="Access and published state"
            onClick={() => workspace.openPane({ kind: "access" })}
          />
        </>
      }
      words={STATUS[workspace.saveState]}
    />
  );
}
