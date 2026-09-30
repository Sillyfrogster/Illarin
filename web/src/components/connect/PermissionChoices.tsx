"use client";

import { CheckboxGroup } from "@/components/ui/checkbox";
import {
  describePermission,
  PERMISSION_ORDER,
  type Permission,
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
    <CheckboxGroup
      disabled={disabled}
      onValueChange={onChange}
      options={PERMISSION_ORDER.map((permission) => {
        const copy = describePermission(permission);
        return {
          value: permission,
          label: requested?.includes(permission) ? (
            <>
              {copy.title}
              <span className="ml-2 text-meta text-mute">
                The app asks for this
              </span>
            </>
          ) : (
            copy.title
          ),
          hint: copy.detail,
        };
      })}
      value={granted}
    />
  );
}
