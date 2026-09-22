import type { Permission } from "@/lib/api/shapes";
export type { Permission };

export type PermissionCopy = {
  title: string;
  detail: string;
};

const PERMISSIONS: Record<Permission, PermissionCopy> = {
  "work:receive": {
    title: "Receive works you send it",
    detail:
      "You pick each work. It can't browse or take anything. Turned off, nothing you send reaches it.",
  },
  "library:sync": {
    title: "Share its library",
    detail:
      "Illarin shows which works it has and when a newer version is out. Turned off, Illarin forgets the list.",
  },
};

export const PERMISSION_ORDER = Object.keys(PERMISSIONS) as Permission[];

export function describePermission(permission: Permission): PermissionCopy {
  return (
    PERMISSIONS[permission] ?? {
      title: permission,
      detail: "This version of Illarin does not recognise this permission.",
    }
  );
}

/** withPermission turns one permission on or off, keeping the set in Illarin's order. */
export function withPermission(
  granted: Permission[],
  permission: Permission,
  on: boolean,
): Permission[] {
  return PERMISSION_ORDER.filter((one) =>
    one === permission ? on : granted.includes(one),
  );
}
