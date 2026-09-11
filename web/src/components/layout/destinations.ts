import type { SignedInAccount } from "@/lib/auth";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export const NAV = [
  { label: "Browse", href: "/browse" },
  { label: "Blog", href: "/blog" },
] as const;

export type Destination = { label: string; href: string };
export type AccountDestination = Destination & {
  id:
    | "profile"
    | "settings"
    | "publication"
    | "recognition"
    | "verify"
    | "sign-in"
    | "sign-up";
};

export function publishAction(
  account: SignedInAccount | null | undefined,
): Destination {
  if (account === undefined) return { label: "Publish", href: "/upload" };
  if (account === null)
    return {
      label: "Sign in to publish",
      href: `/sign-in?returnTo=${UPLOAD_RETURN}`,
    };
  if (!account.emailVerified)
    return {
      label: "Verify email to publish",
      href: `/verify-email?returnTo=${UPLOAD_RETURN}`,
    };
  return { label: "Publish", href: "/upload" };
}

export function accountDestinations(
  account: SignedInAccount | null | undefined,
  publicationAuthority: boolean,
): AccountDestination[] {
  if (!account)
    return [
      { id: "sign-in", label: "Sign in", href: "/sign-in" },
      { id: "sign-up", label: "Create account", href: "/sign-up" },
    ];

  return [
    { id: "profile", label: "View profile", href: `/@${account.handle}` },
    { id: "settings", label: "Account settings", href: "/settings" },
    ...(publicationAuthority
      ? [
          {
            id: "publication" as const,
            label: "Blog administration",
            href: "/publication",
          },
          {
            id: "recognition" as const,
            label: "Profile recognition",
            href: "/recognition",
          },
        ]
      : []),
    ...(account.emailVerified
      ? []
      : [
          {
            id: "verify" as const,
            label: "Verify email",
            href: `/verify-email?returnTo=${UPLOAD_RETURN}`,
          },
        ]),
  ];
}

export function isCurrentPage(pathname: string, href: string) {
  const [path] = href.split("?");
  return pathname === path || pathname.startsWith(`${path}/`);
}
