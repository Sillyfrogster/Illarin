"use client";

import type { ReactNode } from "react";
import { useOrigins } from "@/lib/origins";
import { isSafeAddress, leavesIllarin } from "@/lib/post-link";

export function PostLink({
  href,
  children,
}: {
  href: string;
  children: ReactNode;
}) {
  const { site } = useOrigins();
  if (!isSafeAddress(href)) return <>{children}</>;
  const away = leavesIllarin(href, site);
  return (
    <a
      className="text-ink underline decoration-accent/55 underline-offset-[3px] transition-colors hover:decoration-accent"
      href={href}
      rel="noreferrer nofollow"
      target={away ? "_blank" : undefined}
    >
      {children}
      {away ? <span className="sr-only"> (opens in a new tab)</span> : null}
    </a>
  );
}
