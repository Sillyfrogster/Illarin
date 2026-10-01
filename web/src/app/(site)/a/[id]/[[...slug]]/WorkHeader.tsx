"use client";

import { ArrowLeft, EyeOff, Globe, PencilLine } from "lucide-react";
import Link from "next/link";
import { type CSSProperties, useState } from "react";
import { Badge, BadgeList } from "@/components/ui/badge";
import { Field } from "@/components/ui/field";
import { Textarea } from "@/components/ui/input";
import { FormattingNotice, RichText } from "@/components/ui/RichText";
import { RadioGroup } from "@/components/ui/radio-group";
import { TagField } from "@/components/ui/tag-field";
import { WorkOwnerMenu } from "@/components/work/WorkOwnerMenu";
import type { Profile, WorkConnectedApp, WorkDetail } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { formattingWasRemoved } from "@/lib/rich-text";
import { tagSearchHref } from "@/lib/tag-search";
import { workDisplayName } from "@/lib/work-name";
import { canSendWork } from "@/lib/work-send";
import { CreatorLine } from "./card/CreatorLine";
import { CardStage } from "./card/stage";
import { VersionTimeline } from "./card/VersionTimeline";
import { cardNameSize, WorkCard } from "./card/WorkCard";
import { FollowControl } from "./follow/FollowControl";
import { WorkFollowProvider } from "./follow/state";
import { GetWork } from "./GetWork";
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
/** Covers wider than this get a wider card column so they are not a thin strip. */
const WIDE_COVER = 1.25;

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
      : [Globe, "Published · readers see this page"];
  if (!isDraft && !unpublished)
    return (
      <p className="inline-flex items-center gap-1.5 text-meta text-mute">
        <Icon aria-hidden="true" className="size-4" />
        {words}
      </p>
    );
  return (
    <Badge className="min-h-control px-3 text-meta" role="status" tone="accent">
      <Icon aria-hidden="true" className="size-4" />
      {words}
    </Badge>
  );
}

export function WorkHeader({
  connectedApps,
  creator,
  work,
  typeLabel,
  sharedDate,
  shellClassName,
}: {
  connectedApps: WorkConnectedApp[];
  creator: Profile | null;
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
  const cover = coverMedia(work.media).find((image) => image.isCover);
  const aspect = cover ? cover.width / cover.height : 3 / 4;
  const sendable = canSendWork(work);
  const live = !writing;
  const hasBack =
    live && !isDraft && (work.downloads.length > 0 || Boolean(work.original));
  const download = (
    <GetWork
      connectedApps={connectedApps}
      primary={false}
      sendable={sendable}
      typeLabel={typeLabel.toLowerCase()}
      work={work}
    />
  );

  const about = (
    <>
      {writing ? (
        <fieldset className="min-w-0 border-0 p-0" id="adult-content-answer">
          <legend className="text-label text-mute">Adult content</legend>
          <RadioGroup
            className="mt-2"
            onValueChange={(key) =>
              workspace.writeDetails({
                ...workspace.details,
                isNsfw:
                  RATINGS.find((rating) => rating.value === key)?.isNsfw ??
                  null,
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
        <div className="font-prose text-lede text-ink">
          <RichText text={workspace.details.blurb} />
          {formattingWasRemoved([workspace.details.blurb]) ? (
            <FormattingNotice />
          ) : null}
        </div>
      ) : null}

      {writing ? (
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
                tags: workspace.details.tags.filter((_, at) => at !== index),
              });
            }}
            tags={workspace.details.tags}
          />
        </Field>
      ) : workspace.details.tags.length > 0 ? (
        <BadgeList
          items={workspace.details.tags.map((tag) => ({
            href: isDraft ? undefined : tagSearchHref(tag.trim().toLowerCase()),
            id: tag,
            label: tag,
          }))}
          limit={TAG_PREVIEW_LIMIT}
        />
      ) : null}

      <div className="flex flex-col gap-1.5">
        <p className="text-meta text-mute">
          {isDraft ? `Created ${sharedDate}` : `Published ${sharedDate}`}
          {work.identifier ? (
            <span className="ml-2 font-mono">{work.identifier}</span>
          ) : null}
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
    </>
  );

  return (
    <WorkFollowProvider
      workId={work.id}
      initial={work.follow}
      typeName={typeLabel.toLowerCase()}
    >
      <CardStage recorded={Boolean(work.latestVersion)} workId={work.id}>
        <div className={shellClassName}>
          <div className="mt-6 flex flex-wrap items-center gap-x-3 gap-y-2">
            <Link
              className="mr-auto inline-flex min-h-control items-center gap-2 text-meta text-mute hover:text-ink"
              href="/browse"
            >
              <ArrowLeft aria-hidden="true" className="size-4" />
              Browse
            </Link>
            {work.isOwner ? (
              <>
                <div className="order-last w-full sm:order-none sm:w-auto">
                  <OwnerPageState
                    isDraft={isDraft}
                    unpublished={workspace.unpublishedChanges}
                  />
                </div>
                <WorkOwnerMenu work={work} onEdit={workspace.startEditing} />
              </>
            ) : null}
          </div>

          {work.isOwner && writing ? (
            <p className="mt-3 text-meta text-mute">
              Click any text to edit it. Each block has a menu to move, resize
              or hide it.
            </p>
          ) : null}

          <div
            className="grid gap-x-12 gap-y-8 py-8 md:grid-cols-[var(--card-md)_minmax(0,1fr)] md:grid-rows-[auto_auto_auto_auto_1fr] lg:grid-cols-[minmax(0,1fr)_var(--card-w)_minmax(0,1fr)] lg:grid-rows-1 lg:py-12"
            style={
              {
                "--card-w": aspect > WIDE_COVER ? "520px" : "400px",
                "--card-md": aspect > WIDE_COVER ? "360px" : "300px",
                "--card-phone": `min(100%, calc((65svh - 7.5rem) * ${aspect.toFixed(3)} + 1.5rem))`,
              } as CSSProperties
            }
          >
            <div className="contents lg:col-start-1 lg:row-start-1 lg:flex lg:min-w-0 lg:flex-col lg:gap-8">
              <div className="order-2 min-w-0 md:col-start-2">
                <CreatorLine handle={work.creator} profile={creator} />
              </div>
              <div className="order-4 flex min-w-0 flex-col gap-5 md:col-start-2">
                {about}
              </div>
            </div>

            <div className="order-1 mx-auto w-(--card-phone) min-w-0 md:col-start-1 md:row-span-5 md:row-start-1 md:mx-0 md:w-full lg:col-start-2 lg:row-span-1">
              <WorkCard
                art={
                  <WorkMedia
                    id={work.id}
                    isNsfw={work.isNsfw}
                    key={cover?.id ?? "coverless"}
                    type={work.type}
                    media={work.media}
                    name={work.name}
                    preference={work.nsfwPreference}
                    coverInFile={inFile("cover")}
                    writing={work.isOwner && writing}
                  />
                }
                details={
                  <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-meta text-mute">
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
                }
                hasBack={hasBack}
                live={live}
                name={
                  <>
                    <EditableText
                      active={workspace.cursor === "identity:name"}
                      activate={() => workspace.setCursor("identity:name")}
                      as="h1"
                      className={cn(
                        "font-display font-medium tracking-[-0.02em] text-balance break-words",
                        cardNameSize(workspace.details.name),
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
                    {inFile("name") ? <FileMark /> : null}
                  </>
                }
              />
            </div>

            <div className="contents lg:col-start-3 lg:row-start-1 lg:flex lg:min-w-0 lg:flex-col lg:gap-10">
              <div className="order-3 min-w-0 md:col-start-2">
                {isDraft ? null : (
                  <GetWork
                    aside={<FollowControl />}
                    connectedApps={connectedApps}
                    sendable={sendable}
                    typeLabel={typeLabel.toLowerCase()}
                    work={work}
                  />
                )}
                {work.hasPrivatePrompts ? (
                  <p className="mt-4 text-meta text-mute">
                    This {typeLabel.toLowerCase()} has private prompts, so only
                    an app its creator allows can use it:{" "}
                    {work.allowedApps.map((app) => app.label).join(", ")}.
                  </p>
                ) : null}
                {work.takedown ? (
                  <TakedownNotice takedown={work.takedown} />
                ) : null}
              </div>
              <div className="order-5 min-w-0 md:col-start-2">
                <VersionTimeline
                  download={download}
                  typeLabel={typeLabel}
                  typeName={typeLabel.toLowerCase()}
                  work={work}
                />
              </div>
            </div>
          </div>
        </div>
      </CardStage>
    </WorkFollowProvider>
  );
}
