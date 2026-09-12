import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { PublicProfileEditor } from "@/components/profile/PublicProfileEditor";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Edit profile",
  "Change your public name, picture, biography and contact details.",
);

export default function PublicProfileSettings() {
  return (
    <Shell className="pt-8 pb-chapter lg:pt-12">
      <div className="mx-auto max-w-[64rem]">
        <Link
          className="inline-flex min-h-11 items-center gap-1.5 font-ui text-ui text-mute hover:text-ink"
          href="/settings"
        >
          <ChevronLeft
            aria-hidden="true"
            className="size-4"
            strokeWidth={1.8}
          />
          Account settings
        </Link>

        <header className="mt-3 max-w-[58ch]">
          <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
            Edit profile
          </h1>
          <p className="mt-4 font-prose text-lede text-mute">
            These details are public. Leave optional fields empty to hide them.
          </p>
        </header>

        <div className="mt-10">
          <PublicProfileEditor />
        </div>
      </div>
    </Shell>
  );
}
