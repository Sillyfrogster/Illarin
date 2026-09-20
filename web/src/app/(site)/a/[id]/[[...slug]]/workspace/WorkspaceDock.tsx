"use client";

import { Command, Images, LockKeyhole } from "lucide-react";
import { Dock, DockAction, DockTool } from "@/components/workspace/Dock";
import type { SaveState } from "./state";
import { useWorkspace } from "./state";

const STATUS: Record<SaveState, string> = {
  failed: "Not saved",
  private: "Saved privately",
  published: "Published",
  saving: "Saving",
  unsaved: "Unsaved",
};

export function WorkspaceDock({
  detail,
  onJump,
  publishLabel,
  waiting = 0,
}: {
  detail: string;
  onJump: () => void;
  publishLabel: string;
  waiting?: number;
}) {
  const workspace = useWorkspace();

  return (
    <Dock
      actions={
        <>
          <DockAction onClick={workspace.stopEditing}>Stop editing</DockAction>
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
            {publishLabel}
          </DockAction>
        </>
      }
      detail={detail}
      railOpen={workspace.pane !== null}
      state={workspace.saveState}
      tools={
        <>
          <DockTool icon={Command} label="Go to content" onClick={onJump} />
          {waiting > 0 ? (
            <DockTool
              active={workspace.pane?.kind === "vault"}
              count={waiting}
              icon={Images}
              label={`${waiting} ${waiting === 1 ? "picture" : "pictures"} waiting from the README`}
              onClick={() => workspace.openPane({ kind: "vault" })}
            />
          ) : null}
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
