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
      "You choose what goes across. It cannot browse or take anything on its own.",
  },
  "library:sync": {
    title: "Report what it has installed",
    detail:
      "So Illarin can show what you already have, and when a newer version exists.",
  },
};

export function describePermission(permission: Permission): PermissionCopy {
  return (
    PERMISSIONS[permission] ?? {
      title: permission,
      detail: "This version of Illarin does not recognise this permission.",
    }
  );
}
