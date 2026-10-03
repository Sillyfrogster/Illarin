import type { SignedInAccount } from "@/lib/auth";

const UPLOAD_RETURN = encodeURIComponent("/upload");

export type Destination = { label: string; href: string };

export const BROWSE: Destination = { label: "Browse", href: "/browse" };

// Publish always reads "Publish"; the upload page asks for sign-in after the press.
export const PUBLISH: Destination = { label: "Publish", href: "/upload" };

export type AccountDestination = Destination & {
  id:
    | "work"
    | "settings"
    | "posts"
    | "blog-admin"
    | "staff"
    | "verify"
    | "sign-in"
    | "sign-up";
};

/** accountDestinations groups the account menu's pages: your things, your account, then staff tools. */
export function accountDestinations(
  account: SignedInAccount | null | undefined,
  writer: boolean,
): AccountDestination[][] {
  if (!account)
    return [
      [
        { id: "sign-in", label: "Sign in", href: "/sign-in" },
        { id: "sign-up", label: "Create account", href: "/sign-up" },
      ],
    ];

  const staff = account.role === "admin" || account.role === "moderator";
  const groups: AccountDestination[][] = [
    [
      { id: "work", label: "Your work", href: "/work" },
      ...(writer
        ? [{ id: "posts" as const, label: "Your posts", href: "/posts" }]
        : []),
    ],
    [
      { id: "settings", label: "Settings", href: "/settings" },
      ...(account.emailVerified
        ? []
        : [
            {
              id: "verify" as const,
              label: "Verify your email",
              href: `/verify-email?returnTo=${UPLOAD_RETURN}`,
            },
          ]),
    ],
    [
      ...(account.role === "admin"
        ? [
            {
              id: "blog-admin" as const,
              label: "Blog admin",
              href: "/admin/blog",
            },
          ]
        : []),
      ...(staff
        ? [{ id: "staff" as const, label: "Staff", href: "/staff" }]
        : []),
    ],
  ];
  return groups.filter((group) => group.length > 0);
}

export function isCurrentPage(pathname: string, href: string) {
  const [path] = href.split("?");
  return pathname === path || pathname.startsWith(`${path}/`);
}
