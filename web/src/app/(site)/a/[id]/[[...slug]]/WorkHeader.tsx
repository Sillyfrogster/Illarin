"use client";

import { ArrowLeft, EyeOff, Globe, PencilLine } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { Alert } from "@/components/ui/alert";
import { BadgeList } from "@/components/ui/badge";
import { Field } from "@/components/ui/field";
import { Textarea } from "@/components/ui/input";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import { RadioGroup } from "@/components/ui/radio-group";
import { TagField } from "@/components/ui/tag-field";
import { WorkOwnerMenu } from "@/components/work/WorkOwnerMenu";
import type { WorkConnectedApp, WorkDetail } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { formattingWasRemoved } from "@/lib/rich-text";
import { tagSearchHref } from "@/lib/tag-search";
import { workDisplayName } from "@/lib/work-name";
import { canSendWork } from "@/lib/work-send";
import { FollowControl } from "./follow/FollowControl";
import { WorkFollowProvider } from "./follow/state";
import { GetWork } from "./GetWork";
import { VersionHistory } from "./history/VersionHistory";
import { TakedownNotice } from "./TakedownNotice";
import { WorkCounts } from "./WorkCounts";
import { coverMedia, WorkMedia } from "./WorkMedia";
import {
  BLURB_LIMIT,
  blurbCharacterCount,
  blurbLimitMessage,
} from "./workspace/details";
import { EditableText } from "./workspace/EditableText";
import { FileMark } from "./workspace/fields";
import { useWorkspace } from "./workspace/state";

const TAG_PREVIEW_LIMIT = 8;

type RatingKey = "unanswered" | "no" | "yes";

const RATINGS: { value: RatingKey; label: string; isNsfw: boolean | null }[] = [
  { value: "unanswered", label: "Not answered", isNsfw: null },
  { value: "no", label: "No adult content", isNsfw: false },
  { value: "yes", label: "Adult content", isNsfw: true },
];

function ratingLabel(isNsfw: boolean | null): string {
  if (isNsfw === null) return "Rating not set";
  return isNsfw ? "Adult content" : "No adult content";
}

/** OwnerPageState tells the owner whether this page is a private draft, drafted changes, or what readers see. */
function OwnerPageState({
  isDraft,
  unpublished,
}: {
  isDraft: boolean;
  unpublished: boolean;
}) {
  const [Icon, words] = isDraft
    ? [EyeOff, "Private draft · only you can see this"]
    : unpublished
      ? [
          PencilLine,
          "Drafted changes · readers still see the published version",
        ]
      : [Globe, "Published · this is what readers see"];
  return (
    <Alert
      className="mt-4 flex items-center gap-2 font-medium"
      tone={isDraft || unpublished ? "done" : "quiet"}
    >
      <Icon aria-hidden="true" />
      {words}
    </Alert>
  );
}

/** Long names step down the type scale so they wrap in a few lines, not a column of single words */
function nameSize(name: string): string {
  const length = Array.from(name).length;
  if (length > 60) return "max-w-[26ch] text-title leading-tight";
  if (length > 24) return "max-w-[20ch] text-display leading-[1.05]";
  return "max-w-[15ch] text-hero";
}

export function WorkHeader({
  connectedApps,
  work,
  typeLabel,
  sharedDate,
  shellClassName,
}: {
  connectedApps: WorkConnectedApp[];
  work: WorkDetail;
  typeLabel: string;
  sharedDate: string;
  shellClassName: string;
}) {
  const workspace = useWorkspace();
  const [tagTrouble, setTagTrouble] = useState("");
  const isDraft = work.lifecycle === "draft";
  const writing = workspace.editing;
  const ratings = isDraft
    ? RATINGS
    : RATINGS.filter((rating) => rating.isNsfw !== null);
  const blurbCount = blurbCharacterCount(workspace.details.blurb);
  const blurbTrouble = blurbLimitMessage(workspace.details.blurb);
  const inFile = (field: string) =>
    !isDraft && writing && Boolean(work.fileFields?.includes(field));
  const covers = coverMedia(work.media);
  const showsMedia = covers.length > 0 || (work.isOwner && writing);
  const sendable = canSendWork(work);
  const download = (
    <GetWork
      connectedApps={connectedApps}
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
            className="inline-flex min-h-control items-center gap-2 text-meta text-mute hover:text-ink"
            href="/browse"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            Browse
          </Link>
          {work.isOwner ? (
            <WorkOwnerMenu work={work} onEdit={workspace.startEditing} />
          ) : null}
        </div>

        {work.isOwner ? (
          <OwnerPageState
            isDraft={isDraft}
            unpublished={workspace.unpublishedChanges}
          />
        ) : null}

        {work.isOwner && writing ? (
          <p className="mt-3 text-meta text-mute">
            Click any text to edit it. Each block has a menu to move, resize or
            hide it.
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
                "font-display font-medium tracking-[-0.035em] text-balance break-words",
                nameSize(workspace.details.name),
                work.name ? "text-ink" : "text-mute italic",
              )}
              done={() => workspace.setCursor(null)}
              id="work-name"
              label="Name"
              live={writing}
              onChange={(name) =>
                workspace.writeDetails({ ...workspace.details, name })
              }
              placeholder={`Name this ${typeLabel.toLowerCase()}`}
              singleLine
              value={
                writing
                  ? workspace.details.name
                  : workDisplayName(workspace.details.name)
              }
            />
            {inFile("name") ? (
              <p className="mt-3">
                <FileMark />
              </p>
            ) : null}
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
            </p>
            {writing ? (
              <fieldset
                className="mt-4 min-w-0 border-0 p-0"
                id="adult-content-answer"
              >
                <legend className="text-label text-mute">Adult content</legend>
                <RadioGroup
                  className="mt-2"
                  onValueChange={(key) =>
                    workspace.writeDetails({
                      ...workspace.details,
                      isNsfw:
                        RATINGS.find((rating) => rating.value === key)
                          ?.isNsfw ?? null,
                    })
                  }
                  options={ratings}
                  value={
                    RATINGS.find(
                      (rating) => rating.isNsfw === workspace.details.isNsfw,
                    )?.value ?? null
                  }
                />
                {workspace.details.isNsfw === null ? (
                  <p className="mt-2 text-label text-mute">
                    Say whether this is adult content before you publish.
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
            {isDraft ? null : (
              <WorkCounts
                workId={work.id}
                isOwner={work.isOwner}
                views={work.viewCount}
                downloads={work.downloadCount}
                sends={work.sendCount}
                followers={work.followerCount}
              />
            )}
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
                media={work.media}
                name={work.name}
                preference={work.nsfwPreference}
                coverInFile={inFile("cover")}
                writing={work.isOwner && writing}
              />
            </div>
          ) : null}

          <div className="min-w-0 md:col-start-1 lg:col-start-auto">
            {writing ? (
              <Field
                hint="A short description for readers and search."
                htmlFor="work-blurb"
                label={
                  inFile("blurb") ? (
                    <span className="inline-flex flex-wrap items-center gap-2">
                      Blurb
                      <FileMark />
                    </span>
                  ) : (
                    "Blurb"
                  )
                }
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
                <Textarea
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
            ) : null}

            {writing ? (
              <div className="mt-5 max-w-[42ch]">
                <Field
                  htmlFor="work-tag"
                  label="Tags"
                  hint={`Readers find this ${typeLabel.toLowerCase()} by its tags. Up to 32.`}
                  trouble={tagTrouble || undefined}
                >
                  <TagField
                    onAdd={(tag) => {
                      const trouble =
                        Array.from(tag).length > 64
                          ? "Use 1 to 64 characters for each tag."
                          : workspace.details.tags.includes(tag)
                            ? "That tag is already here."
                            : workspace.details.tags.length >= 32
                              ? "Use up to 32 tags."
                              : "";
                      setTagTrouble(trouble);
                      if (trouble) return false;
                      workspace.writeDetails({
                        ...workspace.details,
                        tags: [...workspace.details.tags, tag],
                      });
                      return true;
                    }}
                    onRemove={(index) => {
                      setTagTrouble("");
                      workspace.writeDetails({
                        ...workspace.details,
                        tags: workspace.details.tags.filter(
                          (_, at) => at !== index,
                        ),
                      });
                    }}
                    tags={workspace.details.tags}
                  />
                </Field>
              </div>
            ) : workspace.details.tags.length > 0 ? (
              <BadgeList
                className="mt-5 max-w-[42ch]"
                items={workspace.details.tags.map((tag) => ({
                  href: isDraft
                    ? undefined
                    : tagSearchHref(tag.trim().toLowerCase()),
                  id: tag,
                  label: tag,
                }))}
                limit={TAG_PREVIEW_LIMIT}
              />
            ) : null}

            {isDraft ? null : (
              <div className="mt-7">
                <GetWork
                  aside={<FollowControl />}
                  connectedApps={connectedApps}
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
