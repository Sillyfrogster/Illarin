import { ArrowUpRight, KeyRound, Plug, Send } from "lucide-react";
import Link from "next/link";
import { AccountSettings } from "@/components/auth/AccountSettings";
import { Shell } from "@/components/layout/Shell";
import { LinkedInstances } from "@/components/linking/LinkedInstances";
import { PublicProfileCard } from "@/components/profile/PublicProfileCard";
import { Button } from "@/components/ui/button";
import { pageMetadata } from "@/lib/site-metadata";

const DISCORD_NOTICES: Record<string, string> = {
  attached: "Discord is connected. You can use it to sign in.",
  claimed:
    "That Discord account cannot be connected. Try another Discord account.",
  "email-conflict":
    "Discord's email address belongs to another Illarin account. Use a different Discord account.",
  failed: "Discord could not be connected. Try again.",
};

export const metadata = pageMetadata(
  "Account settings",
  "Manage your sign-in methods and linked applications.",
);

export default async function SettingsPage({
  searchParams,
}: {
  searchParams: Promise<{ discord?: string | string[] }>;
}) {
  const query = await searchParams;
  const discord = Array.isArray(query.discord)
    ? query.discord[0]
    : query.discord;

  return (
    <Shell className="max-w-[78rem] pt-12 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <h1 className="font-display text-[clamp(2rem,3vw,2.75rem)] leading-[1.1] font-medium tracking-[-0.035em] text-balance">
          Account settings
        </h1>
        <p className="mt-3 font-prose text-ui text-mute">
          Manage your sign-in methods and linked applications.
        </p>
      </header>

      <div className="mt-9 grid gap-10 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-14">
        <aside className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:self-start">
          <PublicProfileCard />
          <nav
            aria-label="Account settings"
            className="mt-5 grid grid-cols-1 gap-1 sm:grid-cols-3 lg:grid-cols-1"
          >
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#ways-in"
            >
              <KeyRound aria-hidden="true" className="size-4 text-accent" />
              Sign-in methods
            </a>
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#linked-applications"
            >
              <Plug aria-hidden="true" className="size-4 text-accent" />
              Linked applications
            </a>
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#update-destinations"
            >
              <Send aria-hidden="true" className="size-4 text-accent" />
              Update destinations
            </a>
          </nav>
        </aside>

        <div className="min-w-0">
          <section aria-labelledby="ways-in">
            <h2
              className="scroll-mt-[calc(var(--header-height)+3rem)] font-display text-section font-medium tracking-tight text-ink"
              id="ways-in"
            >
              Sign-in methods
            </h2>
            <div className="mt-5">
              <AccountSettings
                discordNotice={discord ? DISCORD_NOTICES[discord] : undefined}
              />
            </div>
          </section>
          <div className="mt-12 border-t border-rule pt-9 [&_#linked-applications]:scroll-mt-[calc(var(--header-height)+3rem)]">
            <LinkedInstances />
          </div>
          <section
            aria-labelledby="update-destinations"
            className="mt-12 border-t border-rule pt-8"
          >
            <h2
              className="scroll-mt-[calc(var(--header-height)+3rem)] font-display text-section font-medium tracking-tight text-ink"
              id="update-destinations"
            >
              Asset update destinations
            </h2>
            <p className="mt-2 max-w-[52ch] font-prose text-ui text-mute">
              Connect a Discord channel or webhook, then choose defaults for
              each asset.
            </p>
            <Button asChild className="mt-5" variant="secondary">
              <Link href="/settings/update-destinations">
                Manage destinations
                <ArrowUpRight aria-hidden="true" />
              </Link>
            </Button>
          </section>
        </div>
      </div>
    </Shell>
  );
}
