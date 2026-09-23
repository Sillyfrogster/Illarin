import { ChartColumn, type LucideIcon } from "lucide-react";
import type { SignedInAccount } from "@/lib/auth";

export const CONSOLE_ROOT = "/staff";

export type StaffRole = "moderator" | "admin";

export const ROLE_NAMES: Record<StaffRole, string> = {
  admin: "Admin",
  moderator: "Moderator",
};

export type StaffSection = {
  href: string;
  label: string;
  icon: LucideIcon;
  group: string;
  /** The lowest role that sees the section. */
  role: StaffRole;
  /** What the section's badge counts, once something counts it. */
  count?: () => number;
};

/** Every page of the console. A new page is one entry here and one route under /staff. */
export const STAFF_SECTIONS: StaffSection[] = [
  {
    href: CONSOLE_ROOT,
    label: "Report",
    icon: ChartColumn,
    group: "Overview",
    role: "moderator",
  },
];

export function isStaff(
  account: SignedInAccount | null | undefined,
): account is SignedInAccount & { role: StaffRole } {
  return account?.role === "admin" || account?.role === "moderator";
}

export function sectionsFor(role: StaffRole): StaffSection[] {
  return STAFF_SECTIONS.filter(
    (section) => role === "admin" || section.role === "moderator",
  );
}

/** The sections in the order they are shown, gathered under their group heading. */
export function groupsFor(role: StaffRole): [string, StaffSection[]][] {
  const groups = new Map<string, StaffSection[]>();
  for (const section of sectionsFor(role)) {
    groups.set(section.group, [...(groups.get(section.group) ?? []), section]);
  }
  return [...groups];
}

/** The section a path is inside, where the console's own root is exact and a section owns its child pages. */
export function sectionAt(role: StaffRole, pathname: string) {
  return sectionsFor(role)
    .toSorted((a, b) => b.href.length - a.href.length)
    .find(
      (section) =>
        pathname === section.href ||
        (section.href !== CONSOLE_ROOT &&
          pathname.startsWith(`${section.href}/`)),
    );
}
