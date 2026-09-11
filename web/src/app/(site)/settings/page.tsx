import { AccountSettings } from "@/components/auth/AccountSettings";
import { Shell } from "@/components/layout/Shell";
import { LinkedInstances } from "@/components/linking/LinkedInstances";
import { PublicProfileCard } from "@/components/profile/PublicProfileCard";
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
    <Shell className="pt-10 pb-chapter lg:pt-14">
      <header className="max-w-[52ch]">
        <h1 className="font-display text-[clamp(1.85rem,3.4vw,3rem)] leading-[1.05] font-medium tracking-[-0.045em] text-balance">
          Account settings
        </h1>
        <p className="mt-4 font-prose text-lede text-mute">
          Manage your sign-in methods and linked applications.
        </p>
      </header>

      <div className="mt-10 grid gap-section lg:grid-cols-[minmax(0,26rem)_minmax(0,1fr)] lg:gap-16">
        <section
          aria-labelledby="ways-in"
          className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)] lg:self-start"
        >
          <PublicProfileCard />
          <h2
            className="mt-8 font-display text-section font-medium tracking-tight text-ink"
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

        <div className="min-w-0">
          <LinkedInstances />
        </div>
      </div>
    </Shell>
  );
}
