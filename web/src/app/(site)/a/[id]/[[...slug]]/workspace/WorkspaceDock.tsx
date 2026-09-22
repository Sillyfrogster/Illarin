"use client";

import { Command, FileUp, Images, LockKeyhole } from "lucide-react";
import { Dock, DockAction, DockTool } from "@/components/workspace/Dock";
import type { SaveState } from "./state";
import { useWorkspace } from "./state";

const STATUS: Record<SaveState, string> = {
  failed: "Not saved",
  private: "Drafted changes",
  published: "Published",
  saving: "Saving",
  unsaved: "Saving soon",
};

export function WorkspaceDock({
  detail,
  onJump,
  waiting = 0,
}: {
  detail: string;
  onJump: () => void;
  waiting?: number;
}) {
  const workspace = useWorkspace();

  return (
    <Dock
      actions={
        <>
          <DockAction onClick={workspace.stopEditing}>Stop editing</DockAction>
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
      railOpen={workspace.pane !== null}
      state={workspace.saveState}
      tools={
        <>
          <DockTool icon={Command} label="Go to content" onClick={onJump} />
          {!workspace.isDraft ? (
            <DockTool
              icon={FileUp}
              label="Upload a new version"
              onClick={() => workspace.openPane({ kind: "replacement" })}
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
