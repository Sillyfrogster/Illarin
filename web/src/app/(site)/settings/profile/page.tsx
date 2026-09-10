import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { PublicProfileEditor } from "@/components/profile/PublicProfileEditor";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Public profile",
  "The identity visitors meet at your handle.",
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
          Your account
        </Link>

        <header className="mt-3 max-w-[58ch]">
          <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
            Public profile
          </h1>
          <p className="mt-4 font-prose text-lede text-mute">
            Everything here is visible to anyone who opens your profile. Your
            handle stays your address; leave a field empty to show nothing.
          </p>
        </header>

        <div className="mt-10">
          <PublicProfileEditor />
        </div>
      </div>
    </Shell>
  );
}
