import type { ReplacementPreview } from "@/lib/api/query";
import { previewConflicts } from "@/lib/asset-publication";
import { replacementSubjectLabel } from "@/lib/replacement-subject";

export function ReplacementWarnings({
  preview,
}: {
  preview: Pick<ReplacementPreview, "conflicts" | "missingWording" | "seals">;
}) {
  const conflicts = previewConflicts(preview);
  const missingWording = preview.missingWording ?? [];
  return (
    <>
      {preview.seals > 0 ? (
        <p className="rounded-control bg-accent-wash p-3 text-meta text-ink">
          This file keeps the wording of {preview.seals} prompt
          {preview.seals === 1 ? "" : "s"} back. Applying it means readers can
          only install this asset through a linked app.
        </p>
      ) : null}
      {missingWording.length > 0 ? (
        <p className="rounded-control bg-stop-wash p-3 text-meta text-ink">
          This file does not include the wording for:{" "}
          {missingWording.join(", ")}. Illarin will keep{" "}
          {missingWording.length === 1 ? "that prompt" : "those prompts"} sealed
          and empty.
        </p>
      ) : null}
      {conflicts.length > 0 ? (
        <p className="rounded-control bg-stop-wash p-3 text-meta text-ink">
          This file overwrites edits you have not published yet:{" "}
          {conflicts.map(replacementSubjectLabel).join(", ")}.
        </p>
      ) : null}
    </>
  );
}
