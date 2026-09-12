import { ArrowUpRight, Mail } from "lucide-react";
import { CreatorPortrait } from "@/components/media/CreatorPortrait";
import type { Profile, ProfileLink } from "@/lib/api/query";

export function ProfilePreview({
  biography,
  contactEmail,
  displayName,
  handle,
  links,
  picture,
}: {
  biography: string;
  contactEmail: string;
  displayName: string;
  handle: string;
  links: ProfileLink[];
  picture: Profile["avatar"];
}) {
  const shown = links.filter((link) => link.label || link.address);

  return (
    <aside
      aria-label="Profile preview"
      className="min-w-0 lg:sticky lg:top-[calc(var(--header-height)+2.5rem)]"
    >
      <p className="font-ui text-meta text-mute">Public profile preview</p>
      <div className="mt-3 rounded-plate bg-deep p-6">
        <CreatorPortrait handle={handle} picture={picture} size="md" />
        <p className="mt-4 font-display text-section font-medium tracking-tight text-ink [overflow-wrap:anywhere]">
          {displayName || `@${handle}`}
        </p>
        {displayName ? (
          <p className="font-ui text-ui text-mute [overflow-wrap:anywhere]">
            @{handle}
          </p>
        ) : null}
        {biography ? (
          <p className="mt-3 font-prose text-ui text-ink [overflow-wrap:anywhere]">
            {biography}
          </p>
        ) : null}
        {contactEmail || shown.length > 0 ? (
          <ul className="m-0 mt-4 grid list-none gap-1.5 p-0">
            {contactEmail ? (
              <li className="flex items-center gap-2 font-ui text-meta text-mute [overflow-wrap:anywhere]">
                <Mail
                  aria-hidden="true"
                  className="size-3.5 shrink-0"
                  strokeWidth={1.7}
                />
                {contactEmail}
              </li>
            ) : null}
            {shown.map((link, index) => (
              <li
                className="flex items-center gap-1 font-ui text-meta text-mute [overflow-wrap:anywhere]"
                // biome-ignore lint/suspicious/noArrayIndexKey: a link's position is its identity here
                key={index}
              >
                {link.label || link.address}
                <ArrowUpRight
                  aria-hidden="true"
                  className="size-3.5 shrink-0"
                  strokeWidth={1.7}
                />
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <p className="mt-3 font-ui text-meta text-mute [overflow-wrap:anywhere]">
        Your published work follows straight after this, at /@{handle}.
      </p>
    </aside>
  );
}
