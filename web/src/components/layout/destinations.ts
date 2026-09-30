import type { SignedInAccount } from "@/lib/auth";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export type Destination = { label: string; href: string };

export const BROWSE: Destination = { label: "Browse", href: "/browse" };

// Publish always reads "Publish"; the upload page asks for sign-in after the press.
export const PUBLISH: Destination = { label: "Publish", href: "/upload" };

export type AccountDestination = Destination & {
  id:
    | "profile"
    | "work"
    | "settings"
    | "posts"
    | "blog-admin"
    | "staff"
    | "verify"
    | "sign-in"
    | "sign-up";
};

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
    { id: "work", label: "Your work", href: "/work" },
    { id: "profile", label: "Your profile", href: `/@${account.handle}` },
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
    ...(account.role === "admin" || account.role === "moderator"
      ? [{ id: "staff" as const, label: "Staff", href: "/staff" }]
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
