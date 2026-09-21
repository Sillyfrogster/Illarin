"use client";

import { ArrowLeft, PencilLine } from "lucide-react";
import Link from "next/link";
import { ChipSet } from "@/components/ui/Chip";
import { Field, TextArea } from "@/components/ui/field";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import type { WorkDetail } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { formattingWasRemoved } from "@/lib/rich-text";
import { workDisplayName } from "@/lib/work-name";
import { canSendWork } from "@/lib/work-send";
import { FollowControl } from "./follow/FollowControl";
import { WorkFollowProvider } from "./follow/state";
import { GetWork } from "./GetWork";
import { VersionHistory } from "./history/VersionHistory";
import { TakedownNotice } from "./TakedownNotice";
import { coverMedia, WorkMedia } from "./WorkMedia";
import {
  BLURB_LIMIT,
  blurbCharacterCount,
  blurbLimitMessage,
} from "./workspace/details";
import { EditableText } from "./workspace/EditableText";
import { useWorkspace } from "./workspace/state";

const TAG_PREVIEW_LIMIT = 8;

const RATINGS: { label: string; value: boolean | null }[] = [
  { label: "Not answered", value: null },
  { label: "No adult content", value: false },
  { label: "Adult content", value: true },
];

function ratingLabel(isNsfw: boolean | null): string {
  if (isNsfw === null) return "Rating not set";
  return isNsfw ? "Adult content" : "No adult content";
}

function browseTagHref(value: string): string {
  const quoted = value.includes(" ") ? `"${value}"` : value;
  return `/browse?q=${encodeURIComponent(`tag:${quoted}`)}`;
}

export function WorkHeader({
  work,
  typeLabel,
  sharedDate,
  shellClassName,
}: {
  work: WorkDetail;
  typeLabel: string;
  sharedDate: string;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const isDraft = work.lifecycle === "draft";
  const writing = workspace.editing;
  const ratings = isDraft
    ? RATINGS
    : RATINGS.filter((rating) => rating.value !== null);
  const blurbCount = blurbCharacterCount(workspace.details.blurb);
  const blurbTrouble = blurbLimitMessage(workspace.details.blurb);
  const covers = coverMedia(work.media);
  const showsMedia = covers.length > 0 || (work.isOwner && writing);
  const sendable = canSendWork(work);
  const download = (
    <GetWork
      sendable={sendable}
      typeLabel={typeLabel.toLowerCase()}
      work={work}
    />
  );

  return (
    <WorkFollowProvider
      workId={work.id}
      initial={work.follow}
      typeName={typeLabel.toLowerCase()}
    >
      <div className={shellClassName}>
        <div className="mt-6 flex flex-wrap items-center justify-between gap-4">
          <Link
            className="inline-flex min-h-11 items-center gap-2 text-meta text-mute hover:text-ink"
            href="/browse"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            Browse
          </Link>
          {work.isOwner && !writing ? (
            <button
              className="inline-flex min-h-11 items-center gap-2 rounded-control bg-action px-5 text-ui font-medium text-on-accent outline-offset-3 hover:opacity-90"
              onClick={workspace.startEditing}
              type="button"
            >
              <PencilLine aria-hidden="true" size={16} />
              Edit your {typeLabel.toLowerCase()}
            </button>
          ) : null}
        </div>

        {work.isOwner && writing ? (
          <p className="mt-4 text-meta text-mute">
            Select text to edit it. Use block controls to arrange content.
          </p>
        ) : null}

        <div
          className={cn(
            "grid gap-8 py-8 lg:gap-12 lg:py-14",
            showsMedia
              ? "items-center md:grid-cols-2 lg:grid-cols-[1fr_minmax(260px,0.9fr)_1fr]"
              : "items-start md:grid-cols-[1.1fr_1fr]",
          )}
        >
          <div className="min-w-0">
            <EditableText
              active={workspace.cursor === "identity:name"}
              activate={() => workspace.setCursor("identity:name")}
              as="h1"
              className={cn(
                "max-w-[15ch] font-display text-hero font-medium tracking-[-0.035em] break-words",
                work.name ? "text-ink" : "text-mute italic",
              )}
              done={() => workspace.setCursor(null)}
              id="work-name"
              label="Name"
              live={writing}
              onChange={(name) =>
                workspace.writeDetails({ ...workspace.details, name })
              }
              placeholder="Name this page"
              singleLine
              value={
                writing
                  ? workspace.details.name
                  : workDisplayName(workspace.details.name)
              }
            />
            <p className="mt-5 flex flex-wrap items-center gap-x-2 gap-y-1 text-ui text-mute">
              <span
                aria-hidden="true"
                className="size-2 shrink-0 rounded-full bg-accent"
              />
              {typeLabel}
              {writing ? null : (
                <>
                  <span aria-hidden="true">·</span>
                  {ratingLabel(workspace.details.isNsfw)}
                </>
              )}
              {work.hasPrivatePrompts ? (
                <>
                  <span aria-hidden="true">·</span>
                  Private prompts
                </>
              ) : null}
              {isDraft ? (
                <span className="rounded-control bg-accent-wash px-2 py-0.5 text-label font-medium text-ink">
                  Private draft
                </span>
              ) : null}
            </p>
            {writing ? (
              <fieldset
                className="mt-4 min-w-0 border-0 p-0"
                id="adult-content-answer"
              >
                <legend className="text-label text-mute">Adult content</legend>
                <div className="mt-2 flex flex-wrap gap-2">
                  {ratings.map((rating) => (
                    <button
                      aria-pressed={rating.value === workspace.details.isNsfw}
                      className="min-h-11 rounded-control bg-deep px-4 text-meta text-ink outline-offset-3 aria-pressed:bg-action aria-pressed:text-on-accent"
                      key={rating.label}
                      onClick={() =>
                        workspace.writeDetails({
                          ...workspace.details,
                          isNsfw: rating.value,
                        })
                      }
                      type="button"
                    >
                      {rating.label}
                    </button>
                  ))}
                </div>
                {workspace.details.isNsfw === null ? (
                  <p className="mt-2 text-label text-mute">
                    Answer the adult content question before publishing.
                  </p>
                ) : null}
              </fieldset>
            ) : null}
            <p className="mt-7 text-ui text-mute">
              <Link
                className="font-medium text-ink underline decoration-accent/55 underline-offset-4 hover:decoration-accent"
                href={`/@${work.creator}`}
              >
                {work.creator}
              </Link>
              {work.identifier ? (
                <span className="ml-2 font-mono text-meta">
                  {work.identifier}
                </span>
              ) : null}
              <span className="ml-2">
                {isDraft ? `Created ${sharedDate}` : `Published ${sharedDate}`}
              </span>
            </p>
          </div>

          {showsMedia ? (
            <div className="min-w-0 md:row-span-2 lg:row-span-1">
              <WorkMedia
                id={work.id}
                isNsfw={work.isNsfw}
                key={
                  work.media.find((image) => image.isCover)?.id ?? "coverless"
                }
                type={work.type}
                typeLabel={typeLabel.toLowerCase()}
                media={work.media}
                name={work.name}
                preference={work.nsfwPreference}
                writing={work.isOwner && writing}
              />
            </div>
          ) : null}

          <div className="min-w-0 md:col-start-1 lg:col-start-auto">
            {writing ? (
              <Field
                hint="A short description for readers and search."
                htmlFor="work-blurb"
                label="Blurb"
                trailing={
                  <span
                    className={
                      blurbTrouble
                        ? "text-meta tabular-nums text-stop"
                        : "text-meta tabular-nums text-mute"
                    }
                  >
                    {blurbCount} / {BLURB_LIMIT} characters
                  </span>
                }
                trouble={blurbTrouble || undefined}
              >
                <TextArea
                  aria-describedby={
                    blurbTrouble
                      ? "work-blurb-trouble work-blurb-hint"
                      : "work-blurb-hint"
                  }
                  aria-invalid={Boolean(blurbTrouble) || undefined}
                  id="work-blurb"
                  onChange={(event) =>
                    workspace.writeDetails({
                      ...workspace.details,
                      blurb: event.target.value,
                    })
                  }
                  placeholder={`Describe this ${typeLabel.toLowerCase()} in a few words`}
                  rows={5}
                  value={workspace.details.blurb}
                />
              </Field>
            ) : workspace.details.blurb ? (
              <div className="max-w-[42ch] font-prose text-lede text-ink">
                <RichText text={workspace.details.blurb} />
                {formattingWasRemoved([workspace.details.blurb]) ? (
                  <FormattingNotice />
                ) : null}
              </div>
            ) : (
              <p className="max-w-[42ch] font-prose text-lede text-mute">
                The creator has not written a blurb for this{" "}
                {typeLabel.toLowerCase()} yet.
              </p>
            )}

            {work.tags.length > 0 ? (
              <ChipSet
                className="mt-5 max-w-[42ch]"
                items={work.tags.map((tag) => ({
                  href: browseTagHref(tag.value),
                  id: tag.value,
                  label: tag.label,
                }))}
                limit={TAG_PREVIEW_LIMIT}
              />
            ) : null}

            {isDraft ? null : (
              <div className="mt-7">
                <GetWork
                  aside={<FollowControl />}
                  sendable={sendable}
                  typeLabel={typeLabel.toLowerCase()}
                  work={work}
                />
              </div>
            )}

            {work.hasPrivatePrompts ? (
              <p className="mt-4 max-w-[42ch] text-meta text-mute">
                This {typeLabel.toLowerCase()} has private prompts, so only an
                app its creator allows can use it:{" "}
                {work.allowedApps.map((app) => app.label).join(", ")}.
              </p>
            ) : null}

            {work.takedown ? <TakedownNotice takedown={work.takedown} /> : null}

            <VersionHistory
              work={work}
              download={download}
              typeName={typeLabel.toLowerCase()}
              typeLabel={typeLabel}
            />
          </div>
        </div>
      </div>
    </WorkFollowProvider>
  );
}
