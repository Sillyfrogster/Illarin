"use client";

import {
  describePermission,
  PERMISSION_ORDER,
  type Permission,
  withPermission,
} from "@/lib/permissions";

/** PermissionChoices is one checkbox per permission, each with what turning it on or off does. */
export function PermissionChoices({
  disabled,
  granted,
  onChange,
  requested,
}: {
  disabled?: boolean;
  granted: Permission[];
  onChange: (granted: Permission[]) => void;
  requested?: Permission[];
}) {
  return (
    <ul className="m-0 grid list-none gap-4 p-0">
      {PERMISSION_ORDER.map((permission) => {
        const copy = describePermission(permission);
        return (
          <li key={permission}>
            <label className="flex cursor-pointer items-start gap-3 has-disabled:cursor-default has-disabled:opacity-60">
              <input
                checked={granted.includes(permission)}
                className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
                disabled={disabled}
                onChange={(event) =>
                  onChange(
                    withPermission(granted, permission, event.target.checked),
                  )
                }
                type="checkbox"
              />
              <span className="min-w-0">
                <strong className="block font-ui text-ui font-medium text-ink">
                  {copy.title}
                  {requested?.includes(permission) ? (
                    <span className="ml-2 font-normal text-meta text-mute">
                      The app asks for this
                    </span>
                  ) : null}
                </strong>
                <span className="font-prose text-ui text-mute">
                  {copy.detail}
                </span>
              </span>
            </label>
          </li>
        );
      })}
    </ul>
  );
}
