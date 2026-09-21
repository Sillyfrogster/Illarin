import Link from "next/link";
import type { ReactNode } from "react";
import { Shell } from "@/components/layout/Shell";

/** AuthPage holds an account form in one centred card, with the way to the other form in its footer. */
export function AuthPage({
  children,
  introduction,
  returnTo,
  switchTo,
  title,
}: {
  children: ReactNode;
  introduction?: string;
  returnTo?: string;
  switchTo?: { href: string; label: string };
  title: ReactNode;
}) {
  return (
    <Shell className="flex justify-center pt-8 pb-16 sm:pt-12 lg:pt-16">
      <div className="w-full max-w-[30rem] overflow-hidden rounded-[1.75rem] bg-deep p-1 shadow-[0_18px_40px_-24px_rgb(0_0_0/0.35)]">
        <div className="rounded-[1.5rem] bg-plane px-6 pt-8 pb-9 shadow-[0_1px_2px_rgb(0_0_0/0.06)] sm:px-9 sm:pt-10 sm:pb-10">
          <h1 className="font-display text-[clamp(1.75rem,2.6vw,2.125rem)] leading-[1.15] font-medium tracking-[-0.03em] text-balance">
            {title}
          </h1>
          {introduction ? (
            <p className="mt-2.5 max-w-[46ch] font-prose text-ui text-mute">
              {introduction}
            </p>
          ) : null}
          <div className="mt-7">{children}</div>
        </div>
        {switchTo ? (
          <div className="flex justify-center px-6 py-1.5">
            <Link
              className="inline-flex min-h-11 items-center rounded-control px-3 font-ui text-ui font-medium text-accent underline-offset-4 hover:underline"
              href={`${switchTo.href}${returnTo ? `?returnTo=${encodeURIComponent(returnTo)}` : ""}`}
            >
              {switchTo.label}
            </Link>
          </div>
        ) : null}
      </div>
    </Shell>
  );
}
