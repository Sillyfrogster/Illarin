"use client";

import { ShieldCheck } from "lucide-react";
import type { WorkElement } from "@/lib/api/query";
import type { AppName } from "@/lib/api/shapes";
import { cn } from "@/lib/cn";

export type SealedPolicyState = {
  allowedApps: AppName[];
  eligibleApps: AppName[];
  onChange: (apps: AppName[]) => void;
};

export const NO_ALLOWED_APP =
  "Choose at least one allowed app before saving a sealed prompt.";

export function hasSealedPrompts(elements: WorkElement[]): boolean {
  return elements.some(elementSealsAPrompt);
}

export function elementSealsAPrompt(element: WorkElement): boolean {
  return (
    element.type === "prompt_list" &&
    "fragments" in element.content &&
    element.content.fragments.some((fragment) => fragment.protected)
  );
}

export function SealedPolicy({
  policy,
  pending,
  unanswered,
  innerRef,
}: {
  policy: SealedPolicyState;
  pending: boolean;
  unanswered: boolean;
  innerRef?: React.Ref<HTMLFieldSetElement>;
}) {
  return (
    <fieldset
      ref={innerRef}
      className={cn(
        "rounded-plate border-0 bg-deep p-4",
        unanswered && "inset-ring-2 inset-ring-stop",
      )}
      aria-describedby="sealed-policy-note"
    >
      <legend className="flex items-center gap-2 text-ui font-medium text-ink">
        <ShieldCheck size={15} aria-hidden="true" />
        Allowed apps
      </legend>
      <p className="mt-2 text-meta text-mute" id="sealed-policy-note">
        A sealed prompt leaves Illarin only through a connected app you allow
        here, and it arrives as plain text. Sealing is not encryption.
      </p>
      {policy.eligibleApps.length > 0 ? (
        <div className="mt-3 flex flex-wrap gap-2">
          {policy.eligibleApps.map((app) => {
            const chosen = policy.allowedApps.some((one) => one.id === app.id);
            return (
              <label
                key={app.id}
                className={cn(
                  "flex min-h-11 cursor-pointer items-center gap-2 rounded-control px-3 text-ui text-ink",
                  chosen ? "bg-accent-wash" : "bg-plane",
                )}
              >
                <input
                  type="checkbox"
                  checked={chosen}
                  onChange={(event) =>
                    policy.onChange(
                      event.target.checked
                        ? [
                            ...policy.allowedApps.filter(
                              (one) => one.id !== app.id,
                            ),
                            app,
                          ]
                        : policy.allowedApps.filter((one) => one.id !== app.id),
                    )
                  }
                  disabled={pending}
                  className="size-4 accent-[var(--v-action)]"
                />
                {app.label}
              </label>
            );
          })}
        </div>
      ) : (
        <p className="mt-3 text-meta text-mute">
          No connected app can receive this preset in its current form.
        </p>
      )}
      {unanswered ? (
        <p className="mt-3 text-meta text-stop" role="alert">
          {NO_ALLOWED_APP}
        </p>
      ) : null}
    </fieldset>
  );
}
