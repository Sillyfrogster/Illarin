"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { OwnerMenu } from "@/components/ui/owner-menu";
import {
  type BrowseWork,
  deleteWork,
  type WorkDetail,
  workKeys,
} from "@/lib/api/query";
import { workDisplayName } from "@/lib/work-name";
import { TYPE_LABELS } from "@/lib/work-types";
import { workHref } from "@/lib/work-url";
import { VisibilityItems } from "./VisibilityItems";

export function WorkOwnerMenu({
  work,
  onEdit,
}: {
  work: BrowseWork | WorkDetail;
  onEdit?: () => void;
}) {
  const router = useRouter();
  const query = useQueryClient();
  const href = workHref(work.id, work.name);
  const draft =
    "lifecycle" in work
      ? work.lifecycle === "draft"
      : work.ownerState === "draft";
  const visibility =
    "visibility" in work
      ? work.visibility
      : work.ownerState === "unlisted"
        ? "unlisted"
        : "listed";
  const noun = TYPE_LABELS[work.type].toLowerCase();
  return (
    <OwnerMenu
      name={workDisplayName(work.name)}
      noun={noun}
      href={draft ? null : href}
      frozen={Boolean(work.takedown)}
      onEdit={onEdit ?? (() => router.push(`${href}?edit=true`))}
      onDelete={async () => {
        await deleteWork(work.id);
        await query.invalidateQueries({ queryKey: workKeys.all });
        router.push("/work?deleted=true");
        router.refresh();
      }}
      visibilityItems={
        <VisibilityItems
          workId={work.id}
          isDraft={draft}
          initialVisibility={visibility}
          frozen={Boolean(work.takedown)}
          typeName={noun}
        />
      }
    />
  );
}
