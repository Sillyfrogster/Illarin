import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { IntegrationSettings } from "@/components/updates/IntegrationSettings";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Integrations",
  "Where Illarin announces the versions you publish.",
);

export default function IntegrationsPage() {
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
          Integrations
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">
          An integration is a place Illarin announces to: a Discord channel or
          your own webhook. Choose one when you publish a version. Adding one
          announces nothing on its own.
        </p>
      </header>
      <IntegrationSettings />
    </Shell>
  );
}
