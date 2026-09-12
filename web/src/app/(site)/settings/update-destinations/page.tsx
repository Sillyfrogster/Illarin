import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { UpdateDestinationSettings } from "@/components/updates/UpdateDestinationSettings";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Update destinations",
  "Where Illarin announces the updates you publish.",
);

export default function UpdateDestinationsPage() {
  return (
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <Link
        className="inline-flex min-h-11 items-center text-ui text-accent underline-offset-4 hover:underline"
        href="/settings"
      >
        Account settings
      </Link>
      <header className="mt-4 max-w-[56ch]">
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          Update destinations
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">
          Connect a Discord channel or your own endpoint, then choose it when
          publishing an update. Connecting a destination does not send an
          announcement.
        </p>
      </header>
      <UpdateDestinationSettings />
    </Shell>
  );
}
