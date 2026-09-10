import { ArrowUpRight, Mail, ShieldOff, SquarePen, Trash2 } from "lucide-react";
import Link from "next/link";
import { Shell } from "@/components/layout/Shell";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { LineLink } from "@/components/ui/line-link";
import type { Profile } from "@/lib/api/query";
import { siteUrl } from "@/lib/site-metadata";
import { ProfileRecognition } from "./ProfileRecognition";
import { RestrictionControl } from "./RestrictionControl";

function Portrait({ profile }: { profile: Profile }) {
  return (
    <CreatorPortrait
      handle={profile.handle}
      picture={profile.restricted ? null : profile.avatar}
      priority
      size="lg"
    />
  );
}

/** Who this creator is, everything Illarin has given them, and the ways to reach them. */
export function ProfileBanner({
  deletedCount,
  isOwner,
  profile,
}: {
  deletedCount: number | null;
  isOwner: boolean;
  profile: Profile;
}) {
  const name = profile.displayName || `@${profile.handle}`;
  const address = `${siteUrl}/@${profile.handle}`;

  return (
    <Shell as="header" className="pt-10 pb-8 lg:pt-14 lg:pb-10">
      <div className="grid grid-cols-[auto_minmax(0,1fr)] items-start gap-x-5 gap-y-6 sm:gap-x-7 lg:grid-cols-[auto_minmax(0,1fr)_auto] lg:gap-x-9">
        <div className="lg:row-span-2">
          <Portrait profile={profile} />
        </div>

        <div className="min-w-0 self-center lg:self-start">
          <h1 className="font-display text-[clamp(1.7rem,3.6vw,3rem)] leading-[1.05] font-medium tracking-[-0.04em] [overflow-wrap:anywhere]">
            {profile.restricted ? `@${profile.handle}` : name}
          </h1>
          {profile.restricted ? null : (
            <p className="mt-2 flex flex-wrap items-center gap-x-2.5 gap-y-1 font-ui text-ui text-mute">
              <span className="[overflow-wrap:anywhere]">
                @{profile.handle}
              </span>
              {profile.positions.map((position) => (
                <span className="flex items-center gap-2.5" key={position.id}>
                  <span aria-hidden="true">·</span>
                  <span className="text-ink">{position.name}</span>
                </span>
              ))}
            </p>
          )}
        </div>

        <div className="col-span-2 min-w-0 empty:hidden lg:col-span-1 lg:col-start-2 lg:row-start-2">
          {profile.restricted ? (
            <p className="flex max-w-[52ch] items-start gap-2.5 font-prose text-ui text-mute">
              <ShieldOff
                aria-hidden="true"
                className="mt-1 size-4 shrink-0 text-stop"
                strokeWidth={1.7}
              />
              Illarin has hidden what this creator added to their profile. Their
              published work is below.
            </p>
          ) : (
            <>
              {profile.biography ? (
                <p className="max-w-[62ch] font-prose text-lede text-ink [overflow-wrap:anywhere]">
                  {profile.biography}
                </p>
              ) : null}

              {profile.contactEmail || profile.links.length > 0 ? (
                <ul className="mt-4 flex list-none flex-wrap items-center gap-x-6 p-0">
                  {profile.contactEmail ? (
                    <li>
                      <a
                        className="group flex min-h-11 items-center gap-2 font-ui text-ui font-medium text-mute hover:text-ink"
                        href={`mailto:${profile.contactEmail}`}
                      >
                        <Mail
                          aria-hidden="true"
                          className="size-4"
                          strokeWidth={1.7}
                        />
                        <span className="[overflow-wrap:anywhere]">
                          {profile.contactEmail}
                        </span>
                      </a>
                    </li>
                  ) : null}
                  {profile.links.map((link) => (
                    <li key={link.address}>
                      <LineLink
                        href={link.address}
                        rel="nofollow ugc noopener"
                        target="_blank"
                      >
                        {link.label}
                        <ArrowUpRight
                          aria-hidden="true"
                          className="ml-1.5 inline size-3.5 align-[-2px]"
                        />
                      </LineLink>
                    </li>
                  ))}
                </ul>
              ) : null}
            </>
          )}
        </div>

        <div className="col-span-2 flex flex-wrap items-center gap-2 lg:col-span-1 lg:col-start-3 lg:row-start-1 lg:justify-end">
          <CopyButton label="Copy this profile's address" text={address} />
          {deletedCount !== null ? (
            <Button asChild variant="ghost">
              <a href="#deleted">
                <Trash2 aria-hidden="true" />
                Deleted
                <span className="tabular-nums">{deletedCount}</span>
              </a>
            </Button>
          ) : null}
          {isOwner && !profile.restricted ? (
            <Button asChild variant="outline">
              <Link href="/settings/profile">
                <SquarePen aria-hidden="true" />
                Edit profile
              </Link>
            </Button>
          ) : null}
        </div>
      </div>

      {!profile.restricted &&
      (profile.titles.length > 0 || profile.badges.length > 0) ? (
        <div className="mt-8 border-t border-rule pt-6">
          <ProfileRecognition badges={profile.badges} titles={profile.titles} />
        </div>
      ) : null}

      <RestrictionControl
        handle={profile.handle}
        restricted={profile.restricted}
      />
    </Shell>
  );
}
