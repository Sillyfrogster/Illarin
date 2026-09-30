"use client";

import {
  SiBluesky,
  SiDiscord,
  SiGithub,
  SiItchdotio,
  SiKofi,
  SiPatreon,
  SiPixiv,
  SiReddit,
  SiTelegram,
  SiTumblr,
  SiTwitch,
  SiX,
  SiYoutube,
} from "@icons-pack/react-simple-icons";
import {
  ArrowUpRight,
  Globe,
  Mail,
  PencilLine,
  Plus,
  ShieldOff,
  X,
} from "lucide-react";
import { type ComponentType, useRef, useState } from "react";
import { CropPicture, cropsCleanly } from "@/components/media/CropPicture";
import { Alert } from "@/components/ui/alert";
import {
  AvatarFallback,
  AvatarImage,
  Avatar as UiAvatar,
} from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Field } from "@/components/ui/field";
import { Input, Textarea } from "@/components/ui/input";
import {
  arrayMove,
  Sortable,
  SortableItem,
  SortableItemHandle,
} from "@/components/ui/sortable";
import { Tooltip } from "@/components/ui/tooltip";
import type { Profile, ProfileLink } from "@/lib/api/query";
import type { SaveProfileRequest } from "@/lib/api/shapes";
import { cn } from "@/lib/cn";
import { portraitGround } from "@/lib/portrait-tone";
import {
  addLink,
  BIOGRAPHY_LIMIT,
  LINK_LIMIT,
  removeLink,
  writeLink,
} from "@/lib/profile-draft";
import { type LinkBrand, linkBrand, linkText } from "@/lib/profile-links";
import { countWords } from "@/lib/profile-portfolio";
import { FollowButton } from "./FollowButton";

const BRAND_MARKS: Record<LinkBrand, ComponentType<{ className?: string }>> = {
  bluesky: SiBluesky,
  discord: SiDiscord,
  github: SiGithub,
  itch: SiItchdotio,
  "ko-fi": SiKofi,
  patreon: SiPatreon,
  pixiv: SiPixiv,
  reddit: SiReddit,
  telegram: SiTelegram,
  tumblr: SiTumblr,
  twitch: SiTwitch,
  x: SiX,
  youtube: SiYoutube,
};

export type ProfileDraft = SaveProfileRequest;

export type Editing = {
  draft: ProfileDraft;
  change: (draft: ProfileDraft) => void;
  failedField: string;
  fieldTrouble: string;
  save: () => void;
  saving: boolean;
  cancel: () => void;
  onPickAvatar: (file: File) => void;
  onRemoveAvatar: () => void;
  picturePending: boolean;
};

/** Who the creator is, and what the viewer can do about it */
export function IdentityCard({
  address,
  editing,
  isOwner,
  onFollowChange,
  onStartEditing,
  profile,
  said,
  trouble,
}: {
  address: string;
  editing: Editing | null;
  isOwner: boolean;
  onFollowChange: (following: boolean, followers: number) => void;
  onStartEditing: () => void;
  profile: Profile;
  said: string;
  trouble: string;
}) {
  const name = profile.displayName || `@${profile.handle}`;

  return (
    <div className="relative rounded-plate bg-[color-mix(in_oklab,var(--v-plane)_94%,var(--tint))] px-6 pt-4 pb-6 shadow-cover ring-1 ring-rule/70 sm:px-7 sm:pb-7">
      <Avatar editing={editing} profile={profile} />

      <div className="mt-4">
        {editing ? (
          <label className="block">
            <span className="sr-only">Display name</span>
            <input
              aria-invalid={editing.failedField === "displayName" || undefined}
              autoComplete="nickname"
              className="w-full min-w-0 border-0 border-b border-dashed border-rule bg-transparent p-0 pb-1 font-display text-[clamp(1.85rem,2.6vw,2.4rem)] leading-[1.1] font-medium tracking-[-0.035em] text-ink outline-offset-4 placeholder:text-mute"
              maxLength={48}
              onChange={(event) =>
                editing.change({
                  ...editing.draft,
                  displayName: event.target.value,
                })
              }
              placeholder={`@${profile.handle}`}
              type="text"
              value={editing.draft.displayName}
            />
          </label>
        ) : (
          <h1 className="font-display text-[clamp(1.85rem,2.6vw,2.4rem)] leading-[1.1] font-medium tracking-[-0.035em] text-ink [overflow-wrap:anywhere]">
            {profile.restricted ? `@${profile.handle}` : name}
          </h1>
        )}
        {!profile.restricted && (editing || profile.displayName) ? (
          <p className="mt-1 font-ui text-ui text-mute [overflow-wrap:anywhere]">
            @{profile.handle}
          </p>
        ) : null}
      </div>

      {profile.restricted ? (
        <p className="mt-4 flex items-start gap-2.5 font-prose text-ui text-mute">
          <ShieldOff
            aria-hidden="true"
            className="mt-1 size-4 shrink-0 text-stop"
            strokeWidth={1.7}
          />
          Illarin has hidden what this creator added to their profile. Their
          published work is beside this.
        </p>
      ) : editing ? (
        <EditableWords editing={editing} />
      ) : (
        <>
          {profile.biography ? (
            <p className="mt-4 font-prose text-prose text-ink [overflow-wrap:anywhere]">
              {profile.biography}
            </p>
          ) : isOwner ? (
            <p className="mt-4 font-prose text-ui text-mute">
              Say what you make. Readers see it under your name.
            </p>
          ) : null}
          <LinkRows profile={profile} />
        </>
      )}

      <p className="mt-5 font-ui text-ui text-ink tabular-nums">
        {countWords(
          profile.works,
          profile.followers,
          isOwner ? "published work" : "work",
        )}
      </p>

      <div className="mt-4">
        {editing ? (
          <div className="flex flex-wrap items-center gap-2">
            <Button
              className="flex-1"
              loading={editing.saving}
              onClick={editing.save}
              variant="primary"
            >
              {editing.saving ? "Saving" : "Save"}
            </Button>
            <Button
              className="flex-1"
              disabled={editing.saving}
              onClick={editing.cancel}
              variant="secondary"
            >
              Cancel
            </Button>
          </div>
        ) : isOwner ? (
          <div className="flex flex-wrap items-center gap-2">
            {profile.restricted ? null : (
              <Button
                className="flex-1"
                onClick={onStartEditing}
                variant="primary"
              >
                <PencilLine aria-hidden="true" />
                Edit profile
              </Button>
            )}
            <CopyButton
              className="flex-1"
              label="Copy this profile's address"
              text={address}
            >
              Copy link
            </CopyButton>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-2">
            <FollowButton
              following={profile.following ?? false}
              handle={profile.handle}
              onChange={onFollowChange}
            />
            <CopyButton label="Copy this profile's address" text={address}>
              Copy link
            </CopyButton>
          </div>
        )}
        {trouble ? (
          <div className="mt-3">
            <Alert tone="stop">{trouble}</Alert>
          </div>
        ) : null}
        {said ? (
          <Alert tone="done" className="mt-3">
            {said}
          </Alert>
        ) : null}
      </div>
    </div>
  );
}

function Avatar({
  editing,
  profile,
}: {
  editing: Editing | null;
  profile: Profile;
}) {
  const picker = useRef<HTMLInputElement>(null);
  const [framing, setFraming] = useState<File | null>(null);
  const picture = profile.restricted ? undefined : profile.avatar;
  const frame =
    "size-28 rounded-full ring-4 ring-[color-mix(in_oklab,var(--v-plane)_94%,var(--tint))] shadow-cover";

  return (
    <div className="relative -mt-18 w-fit">
      <UiAvatar className={frame}>
        {picture ? (
          <AvatarImage
            alt=""
            fetchPriority="high"
            height={picture.height}
            src={picture.url}
            width={picture.width}
          />
        ) : null}
        <AvatarFallback
          aria-hidden="true"
          className={cn("text-[2.5rem]", portraitGround(profile.handle))}
          delayMs={picture ? 600 : 0}
        >
          {profile.handle.slice(0, 1).toUpperCase()}
        </AvatarFallback>
      </UiAvatar>
      {editing ? (
        <>
          <Tooltip content={picture ? "Change picture" : "Add picture"}>
            <Button
              aria-label={picture ? "Change picture" : "Add picture"}
              className="absolute! -right-1 -bottom-1 rounded-full"
              disabled={editing.picturePending}
              onClick={() => picker.current?.click()}
              size="icon"
              variant="primary"
            >
              <PencilLine aria-hidden="true" />
            </Button>
          </Tooltip>
          {picture ? (
            <Tooltip content="Remove picture">
              <Button
                aria-label="Remove picture"
                className="absolute! top-0 -right-1 rounded-full bg-field hover:text-stop"
                disabled={editing.picturePending}
                onClick={editing.onRemoveAvatar}
                size="icon-compact"
              >
                <X aria-hidden="true" />
              </Button>
            </Tooltip>
          ) : null}
          <input
            accept="image/png,image/jpeg,image/webp,image/gif"
            className="sr-only"
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file && cropsCleanly(file)) setFraming(file);
              else if (file) editing.onPickAvatar(file);
              event.target.value = "";
            }}
            ref={picker}
            type="file"
          />
          <CropPicture
            aspect={1}
            file={framing}
            onCancel={() => setFraming(null)}
            onDone={(file) => {
              setFraming(null);
              editing.onPickAvatar(file);
            }}
            title="Frame your picture"
          />
        </>
      ) : null}
    </div>
  );
}

function LinkRows({ profile }: { profile: Profile }) {
  if (!profile.contactEmail && profile.links.length === 0) return null;
  return (
    <ul className="mt-4 -mx-2 flex list-none flex-col p-0">
      {profile.contactEmail ? (
        <LinkRow
          href={`mailto:${profile.contactEmail}`}
          icon={Mail}
          label="Email"
          text={profile.contactEmail}
        />
      ) : null}
      {profile.links.map((link) => {
        const brand = linkBrand(link.address);
        return (
          <LinkRow
            external
            href={link.address}
            icon={brand ? BRAND_MARKS[brand] : Globe}
            key={link.address}
            label={link.label}
            text={linkText(link.address)}
          />
        );
      })}
    </ul>
  );
}

function LinkRow({
  external = false,
  href,
  icon: Icon,
  label,
  text,
}: {
  external?: boolean;
  href: string;
  icon: ComponentType<{ className?: string }>;
  label: string;
  text: string;
}) {
  return (
    <li>
      <a
        className="group flex min-h-control items-center gap-3 rounded-control px-2 font-ui text-ui text-ink hover:bg-deep hover:text-ink"
        href={href}
        rel={external ? "nofollow ugc noopener" : undefined}
        target={external ? "_blank" : undefined}
      >
        <Icon aria-hidden="true" className="size-4 shrink-0 text-mute" />
        <span className="shrink-0 font-medium">{label}</span>
        <span className="min-w-0 truncate text-mute">{text}</span>
        <ArrowUpRight
          aria-hidden="true"
          className="ml-auto size-3.5 shrink-0 text-mute transition-transform duration-200 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 motion-reduce:transform-none"
        />
      </a>
    </li>
  );
}

function EditableWords({ editing }: { editing: Editing }) {
  const { draft } = editing;
  const remaining = BIOGRAPHY_LIMIT - draft.biography.length;
  const failed = (field: string) =>
    editing.failedField === field ? editing.fieldTrouble : "";
  const setLinks = (links: ProfileLink[]) =>
    editing.change({ ...draft, links });

  return (
    <div className="mt-5 grid gap-5">
      <Field
        htmlFor="profile-bio"
        label="Biography"
        trailing={
          <span className="font-ui text-meta text-mute tabular-nums">
            {remaining} left
          </span>
        }
        trouble={failed("biography")}
      >
        <Textarea
          aria-invalid={Boolean(failed("biography")) || undefined}
          className="min-h-24 font-prose field-sizing-content"
          id="profile-bio"
          maxLength={BIOGRAPHY_LIMIT}
          onChange={(event) =>
            editing.change({ ...draft, biography: event.target.value })
          }
          placeholder="What you make, and for whom"
          rows={3}
          value={draft.biography}
        />
      </Field>

      <Field
        hint="Shown on your profile. Your sign-in email stays private."
        htmlFor="profile-contact"
        label="Contact email"
        trouble={failed("contactEmail")}
      >
        <Input
          aria-describedby="profile-contact-hint"
          aria-invalid={Boolean(failed("contactEmail")) || undefined}
          autoComplete="off"
          id="profile-contact"
          maxLength={254}
          onChange={(event) =>
            editing.change({ ...draft, contactEmail: event.target.value })
          }
          placeholder="nobody@example.com"
          type="email"
          value={draft.contactEmail}
        />
      </Field>

      <Field label="Links" trouble={failed("links")}>
        {draft.links.length > 0 ? (
          <Sortable
            ids={draft.links.map((_, index) => String(index))}
            labels={(id) => draft.links[Number(id)]?.label || "link"}
            onMove={(from, to) => setLinks(arrayMove(draft.links, from, to))}
          >
            <ul className="m-0 grid list-none gap-3 p-0">
              {draft.links.map((link, index) => (
                <LinkEditor
                  index={index}
                  // biome-ignore lint/suspicious/noArrayIndexKey: links carry no id, and the sortable orders them by position
                  key={index}
                  link={link}
                  links={draft.links}
                  onChange={setLinks}
                />
              ))}
            </ul>
          </Sortable>
        ) : null}
        {draft.links.length < LINK_LIMIT ? (
          <div>
            <Button
              onClick={() => setLinks(addLink(draft.links))}
              size="compact"
              variant="ghost"
            >
              <Plus aria-hidden="true" />
              Add link
            </Button>
          </div>
        ) : null}
      </Field>
    </div>
  );
}

function LinkEditor({
  index,
  link,
  links,
  onChange,
}: {
  index: number;
  link: ProfileLink;
  links: ProfileLink[];
  onChange: (links: ProfileLink[]) => void;
}) {
  const name = link.label || `link ${index + 1}`;
  return (
    <SortableItem
      className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-1.5"
      id={String(index)}
    >
      <SortableItemHandle disabled={links.length < 2} label={`Move ${name}`} />
      <div className="grid gap-1.5">
        <Input
          aria-label={`Link ${index + 1} label`}
          maxLength={32}
          onChange={(event) =>
            onChange(writeLink(links, index, "label", event.target.value))
          }
          placeholder="Label, like Ko-fi"
          type="text"
          value={link.label}
        />
        <Input
          aria-label={`Link ${index + 1} address`}
          maxLength={300}
          onChange={(event) =>
            onChange(writeLink(links, index, "address", event.target.value))
          }
          placeholder="https://"
          type="url"
          value={link.address}
        />
      </div>
      <Tooltip content={`Remove ${name}`}>
        <Button
          aria-label={`Remove ${name}`}
          className="hover:text-stop"
          onClick={() => onChange(removeLink(links, index))}
          size="icon"
          variant="ghost"
        >
          <X aria-hidden="true" />
        </Button>
      </Tooltip>
    </SortableItem>
  );
}
