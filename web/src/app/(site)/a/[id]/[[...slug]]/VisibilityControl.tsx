"use client";

import { Eye, EyeOff, LockKeyhole } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { WorkDetail } from "@/lib/api/query";
import { saveWorkVisibility } from "@/lib/api/query";

export function VisibilityControl({
  workId,
  initialVisibility,
  frozen,
  typeName,
}: {
  workId: string;
  initialVisibility: WorkDetail["visibility"];
  frozen: boolean;
  typeName: string;
}) {
  const router = useRouter();
  const [visibility, setVisibility] = useState(initialVisibility);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  const listed = visibility === "listed";
  const next = listed ? "unlisted" : "listed";

  async function changeVisibility() {
    setPending(true);
    setMessage("");
    setVisibility(next);
    try {
      await saveWorkVisibility(workId, next);
      router.refresh();
    } catch {
      setVisibility(visibility);
      setMessage("The visibility could not be changed. Try again.");
    } finally {
      setPending(false);
    }
  }

  return (
    <section aria-labelledby="visibility-heading" className="flex gap-3">
      <span aria-hidden="true" className="mt-0.5 shrink-0 text-mute">
        {frozen ? (
          <LockKeyhole size={18} />
        ) : listed ? (
          <Eye size={18} />
        ) : (
          <EyeOff size={18} />
        )}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div>
          <h3 className="text-ui font-medium text-ink" id="visibility-heading">
            Visibility
          </h3>
          <p className="mt-1 text-meta text-mute">
            {frozen
              ? `Locked while this ${typeName} is withheld. Only an admin can remove the withhold.`
              : listed
                ? "Listed in Browse and on your public profile."
                : "Not listed in Browse. Anyone with the link can still view and download it."}
          </p>
        </div>
        {message ? (
          <p className="text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
        <Button
          className="self-start"
          disabled={frozen}
          loading={pending}
          onClick={changeVisibility}
        >
          {frozen
            ? "Listing locked"
            : listed
              ? "Make unlisted"
              : "List in Browse"}
        </Button>
      </div>
    </section>
  );
}
