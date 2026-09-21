import type { SignedInAccount } from "@/lib/auth";
import { BLOG_HOME } from "@/lib/blog-paths";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export type Destination = { label: string; href: string };

export function primaryDestinations(): Destination[] {
  return [
    { label: "Browse", href: "/browse" },
    { label: "Blog", href: BLOG_HOME },
  ];
}
export type AccountDestination = Destination & {
  id:
    | "profile"
    | "settings"
    | "posts"
    | "blog-admin"
    | "staff"
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
