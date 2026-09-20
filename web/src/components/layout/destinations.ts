import type { SignedInAccount } from "@/lib/auth";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export type Destination = { label: string; href: string };

/** The primary places to go. The blog is its own origin, so its entry is a full address. */
export function primaryDestinations(blog: string): Destination[] {
  return [
    { label: "Browse", href: "/browse" },
    { label: "Blog", href: blog },
  ];
}
export type AccountDestination = Destination & {
  id:
    | "profile"
    | "settings"
    | "posts"
    | "blog-admin"
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
  writer: boolean,
): AccountDestination[] {
  if (!account)
    return [
      { id: "sign-in", label: "Sign in", href: "/sign-in" },
      { id: "sign-up", label: "Create account", href: "/sign-up" },
    ];

  return [
    { id: "profile", label: "Your work", href: `/@${account.handle}` },
    { id: "settings", label: "Account settings", href: "/settings" },
    ...(writer
      ? [{ id: "posts" as const, label: "Your posts", href: "/posts" }]
      : []),
    ...(account.role === "admin"
      ? [
          {
            id: "blog-admin" as const,
            label: "Blog administration",
            href: "/admin/blog",
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
