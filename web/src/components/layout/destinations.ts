import type { SignedInAccount } from "@/lib/auth";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export const NAV = [
  { label: "Browse", href: "/browse" },
  { label: "Blog", href: "/blog" },
] as const;

export type Destination = { label: string; href: string };

/** Publishing needs an account and a verified address, and says which one is missing */
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
      label: "Verify to publish",
      href: `/verify-email?returnTo=${UPLOAD_RETURN}`,
    };
  return { label: "Publish", href: "/upload" };
}

/** Where an account can go from the shell, in one list both the menu and the sheet read */
export function accountDestinations(
  account: SignedInAccount | null | undefined,
  publicationAuthority: boolean,
): Destination[] {
  if (!account)
    return [
      { label: "Sign in", href: "/sign-in" },
      { label: "Create account", href: "/sign-up" },
    ];

  return [
    { label: "View profile", href: `/@${account.handle}` },
    { label: "Account settings", href: "/settings" },
    ...(publicationAuthority
      ? [
          { label: "The publication", href: "/publication" },
          { label: "Profile badges", href: "/recognition" },
        ]
      : []),
    ...(account.emailVerified
      ? []
      : [
          {
            label: "Verify email",
            href: `/verify-email?returnTo=${UPLOAD_RETURN}`,
          },
        ]),
  ];
}

/** A destination is current when the reader is on it or inside it */
export function isCurrentPage(pathname: string, href: string) {
  const [path] = href.split("?");
  return pathname === path || pathname.startsWith(`${path}/`);
}
