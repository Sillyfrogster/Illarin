import { Compass, KeyRound, Plug, Send } from "lucide-react";
import { AccountSettings } from "@/components/auth/AccountSettings";
import { ConnectedApps } from "@/components/connect/ConnectedApps";
import { Shell } from "@/components/layout/Shell";
import { BrowsePreferences } from "@/components/preferences/BrowsePreferences";
import { PublicProfileCard } from "@/components/profile/PublicProfileCard";
import { DiscordChannel } from "@/components/updates/DiscordChannel";
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
  "Manage what Browse shows you, your sign-in methods and connected apps.",
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
          Manage what Browse shows you, your sign-in methods and connected apps.
        </p>
      </header>

      <div className="mt-9 grid gap-10 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-14">
        <aside className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:self-start">
          <PublicProfileCard />
          <nav
            aria-label="Account settings"
            className="mt-5 grid grid-cols-1 gap-1 sm:grid-cols-2 lg:grid-cols-1"
          >
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#browse"
            >
              <Compass aria-hidden="true" className="size-4 text-accent" />
              Browse
            </a>
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#ways-in"
            >
              <KeyRound aria-hidden="true" className="size-4 text-accent" />
              Sign-in methods
            </a>
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#connected-apps"
            >
              <Plug aria-hidden="true" className="size-4 text-accent" />
              Connected apps
            </a>
            <a
              className="flex min-h-11 items-center gap-3 rounded-control px-3 text-ui text-ink hover:bg-deep"
              href="#discord-channel"
            >
              <Send aria-hidden="true" className="size-4 text-accent" />
              Discord channel
            </a>
          </nav>
        </aside>

        <div className="min-w-0">
          <section aria-labelledby="browse">
            <h2
              className="scroll-mt-[calc(var(--header-height)+3rem)] font-display text-section font-medium tracking-tight text-ink"
              id="browse"
            >
              Browse
            </h2>
            <div className="mt-5">
              <BrowsePreferences />
            </div>
          </section>
          <section
            aria-labelledby="ways-in"
            className="mt-12 border-t border-rule pt-9"
          >
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
          <div className="mt-12 border-t border-rule pt-9 [&_#connected-apps]:scroll-mt-[calc(var(--header-height)+3rem)]">
            <ConnectedApps />
          </div>
          <section
            aria-labelledby="discord-channel"
            className="mt-12 border-t border-rule pt-8"
          >
            <h2
              className="scroll-mt-[calc(var(--header-height)+3rem)] font-display text-section font-medium tracking-tight text-ink"
              id="discord-channel"
            >
              Discord channel
            </h2>
            <p className="mt-2 mb-5 max-w-[52ch] font-prose text-ui text-mute">
              When you publish a new version of a public work, Illarin can post
              its summary and a link here.
            </p>
            <DiscordChannel scope="account" />
          </section>
        </div>
      </div>
    </Shell>
  );
}
