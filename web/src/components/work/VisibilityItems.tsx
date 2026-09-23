"use client";

import { useQueryClient } from "@tanstack/react-query";
import { EyeOff, Globe, Link2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from "@/components/ui/dropdown-menu";
import type { WorkDetail } from "@/lib/api/query";
import { saveWorkVisibility, workKeys } from "@/lib/api/query";

type Choice = WorkDetail["visibility"] | "draft";

const CHOICES = [
  {
    value: "listed",
    label: "Public",
    line: "In Browse and on your profile",
    icon: Globe,
  },
  {
    value: "unlisted",
    label: "Unlisted",
    line: "Anyone with the link",
    icon: Link2,
  },
  { value: "draft", label: "Draft", line: "Only you", icon: EyeOff },
] as const;

/** VisibilityItems are the owner menu choices of public, unlisted or draft. */
export function VisibilityItems({
  workId,
  isDraft,
  initialVisibility,
  frozen,
  typeName,
}: {
  workId: string;
  isDraft: boolean;
  initialVisibility: WorkDetail["visibility"];
  frozen: boolean;
  typeName: string;
}) {
  const router = useRouter();
  const query = useQueryClient();
  const [visibility, setVisibility] = useState(initialVisibility);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const current: Choice = isDraft ? "draft" : visibility;

  async function choose(next: string) {
    if (next === current || next === "draft") return;
    const before = visibility;
    const chosen = next as WorkDetail["visibility"];
    setPending(true);
    setMessage("");
    setVisibility(chosen);
    try {
      await saveWorkVisibility(workId, chosen);
      await query.invalidateQueries({ queryKey: workKeys.all });
      router.refresh();
    } catch {
      setVisibility(before);
      setMessage("Illarin could not change the visibility. Try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <>
      <DropdownMenuLabel className="pb-1 text-label text-mute">
        Visibility
      </DropdownMenuLabel>
      <DropdownMenuRadioGroup value={current} onValueChange={choose}>
        {CHOICES.map(({ value, label, line, icon: Icon }) => (
          <DropdownMenuRadioItem
            disabled={frozen || pending || (value === "draft") !== isDraft}
            key={value}
            onSelect={(event) => event.preventDefault()}
            value={value}
          >
            <Icon aria-hidden="true" />
            <span className="flex flex-col py-1.5">
              {label}
              <span className="text-label text-mute">{line}</span>
            </span>
          </DropdownMenuRadioItem>
        ))}
      </DropdownMenuRadioGroup>
      <p
        className="max-w-64 px-3 pt-1 pb-2 text-label text-mute"
        role={message ? "alert" : undefined}
      >
        {message ||
          (frozen
            ? `Locked while this ${typeName} is taken down.`
            : isDraft
              ? `Publish this ${typeName} to make it public or unlisted.`
              : `A published ${typeName} can't go back to draft.`)}
      </p>
    </>
  );
}
