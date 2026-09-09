"use client";

import { ShieldCheck } from "lucide-react";
import type { AssetElement } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { protectedAppLabel } from "@/lib/protected-apps";

export type AllowedApp = "lumiverse";

/** The allowed-app choice every surface that can seal a prompt has to offer. */
export type SealedPolicyState = {
  allowedApps: AllowedApp[];
  eligibleApps: AllowedApp[];
  onChange: (apps: AllowedApp[]) => void;
};

export const NO_ALLOWED_APP =
  "Choose at least one allowed app before saving a sealed prompt.";

export function hasSealedPrompts(elements: AssetElement[]): boolean {
  return elements.some(elementSealsAPrompt);
}

export function elementSealsAPrompt(element: AssetElement): boolean {
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
        A sealed prompt leaves Illarin only through a linked application you
        allow here, and it arrives as plain text. Sealing is not encryption.
      </p>
      {policy.eligibleApps.length > 0 ? (
        <div className="mt-3 flex flex-wrap gap-2">
          {policy.eligibleApps.map((app) => {
            const chosen = policy.allowedApps.includes(app);
            return (
              <label
                key={app}
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
                            ...policy.allowedApps.filter((one) => one !== app),
                            app,
                          ]
                        : policy.allowedApps.filter((one) => one !== app),
                    )
                  }
                  disabled={pending}
                  className="size-4 accent-[var(--v-action)]"
                />
                {protectedAppLabel(app)}
              </label>
            );
          })}
        </div>
      ) : (
        <p className="mt-3 text-meta text-mute">
          No linked app can receive this preset in its current form.
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
