import { AccountSettings } from "@/components/auth/AccountSettings";
import { Shell } from "@/components/layout/Shell";
import { LinkedInstances } from "@/components/linking/LinkedInstances";
import { PublicProfileCard } from "@/components/profile/PublicProfileCard";
import { pageMetadata } from "@/lib/site-metadata";

const DISCORD_NOTICES: Record<string, string> = {
  attached: "Discord is now another way into this account.",
  claimed:
    "That Discord identity cannot be attached here. No account details were revealed and nothing was merged.",
  "email-conflict":
    "Discord reported an address already verified on another account. Nothing was changed.",
  failed: "Discord could not be attached. Please try again.",
};

export const metadata = pageMetadata(
  "Account settings",
  "The ways back into your account and the applications that can reach it.",
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
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          Your account
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">
          The independent ways back in, and every application allowed to reach
          your work.
        </p>
      </header>

      <div className="mt-10 grid gap-section lg:grid-cols-[minmax(0,26rem)_minmax(0,1fr)] lg:gap-16">
        <section
          aria-labelledby="ways-in"
          className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:self-start"
        >
          <h2
            className="font-display text-section font-medium tracking-tight text-ink"
            id="ways-in"
          >
            Ways in
          </h2>
          <div className="mt-5">
            <PublicProfileCard />
          </div>
          <div className="mt-5">
            <AccountSettings
              discordNotice={discord ? DISCORD_NOTICES[discord] : undefined}
            />
          </div>
        </section>

        <div className="min-w-0">
          <LinkedInstances />
        </div>
      </div>
    </Shell>
  );
}
