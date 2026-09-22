"use client";

import type { LucideIcon } from "lucide-react";
import { CalendarClock, Eye, EyeOff, Send, Undo2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import { readPostDeliveries } from "@/lib/api/posts";
import type { BlogAnnouncementAttempt, Post } from "@/lib/api/query";
import { attemptStanding, attemptState } from "@/lib/attempt-standing";
import { announcementWord } from "@/lib/blog-announcement-attempt";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import { useBlogAddress } from "@/lib/origins";
import {
  type PublishAction,
  publishActions,
  writerStanding,
} from "@/lib/post-writing";
import { howSoon } from "@/lib/schedule-time";
import {
  PublishStep,
  RecoverStep,
  RepublishStep,
  RescheduleStep,
  type StepProps,
  UnpublishStep,
  UnscheduleStep,
} from "./PublishSteps";

type Step = "home" | PublishAction | "reschedule" | "unschedule";

const OFFERS: Record<
  PublishAction,
  { icon: LucideIcon; label: string; line: string; tone?: "stop" }
> = {
  publish: {
    icon: Send,
    label: "Publish now",
    line: "Readers see this version as soon as you press the button.",
  },
  schedule: {
    icon: CalendarClock,
    label: "Schedule post",
    line: "Illarin publishes this exact version at a time you set.",
  },
  unpublish: {
    icon: EyeOff,
    label: "Unpublish post",
    line: "Hide the post from readers. Its content and history remain available for republication.",
    tone: "stop",
  },
  republish: {
    icon: Eye,
    label: "Republish post",
    line: "Republish at the same address with the original publication date.",
  },
  recover: {
    icon: Undo2,
    label: "Restore post",
    line: "Restore the post to its state before deletion.",
  },
};

export function PublishRail({
  onFailure,
  onSaveFirst,
  onSettled,
  post,
  stamp,
}: {
  onFailure: (message: string) => void;
  onSaveFirst: () => Promise<number>;
  onSettled: (post: Post) => void;
  post: Post;
  stamp: number;
}) {
  const [step, setStep] = useState<Step>("home");
  const [refusal, setRefusal] = useState("");

  function leave() {
    setRefusal("");
    setStep("home");
  }

  function settled(next: Post) {
    setRefusal("");
    setStep("home");
    onSettled(next);
  }

  const shared: StepProps = {
    onFailure: (message: string) => {
      setRefusal(message);
      onFailure(message);
    },
    onSaveFirst,
    onSettled: settled,
    post,
  };

  if (step === "home") {
    return (
      <Home onStep={setStep} post={post} refusal={refusal} stamp={stamp} />
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <RailBack onClick={leave}>Publishing</RailBack>
      {refusal ? <Trouble>{refusal}</Trouble> : null}
      {step === "publish" ? <PublishStep {...shared} door="now" /> : null}
      {step === "schedule" ? <PublishStep {...shared} door="later" /> : null}
      {step === "unpublish" ? <UnpublishStep {...shared} /> : null}
      {step === "republish" ? <RepublishStep {...shared} /> : null}
      {step === "recover" ? <RecoverStep {...shared} /> : null}
      {step === "reschedule" ? <RescheduleStep {...shared} /> : null}
      {step === "unschedule" ? <UnscheduleStep {...shared} /> : null}
    </div>
  );
}

function Home({
  onStep,
  post,
  refusal,
  stamp,
}: {
  onStep: (step: Step) => void;
  post: Post;
  refusal: string;
  stamp: number;
}) {
  const blogAddress = useBlogAddress();
  const schedule = post.schedule;
  const waiting =
    schedule &&
    (schedule.state === "pending" || schedule.state === "publishing");
  const soon =
    waiting && schedule.state === "pending" ? howSoon(schedule.at) : "";

  return (
    <div className="flex flex-col gap-7">
      {refusal ? <Trouble>{refusal}</Trouble> : null}

      <section className="flex flex-col gap-2 rounded-plate bg-deep p-4">
        <p className="font-ui text-ui font-medium text-ink">
          {readersHave(post)}
        </p>
        <p className="font-prose text-meta text-mute wrap-anywhere">
          {blogAddress}/{post.slug}
        </p>
        {post.deletion ? (
          <p
            className="font-prose text-meta text-stop"
            suppressHydrationWarning
          >
            {remainingDeletionWindow(post.deletion.until)}. Permanently deleted{" "}
            {readableMoment(post.deletion.until)}.
          </p>
        ) : null}
      </section>

      {waiting ? (
        <section className="flex flex-col gap-3">
          <h3 className="font-display text-ui font-medium text-ink">
            Scheduled
          </h3>
          <p className="font-prose text-meta text-mute">
            Revision {schedule.revisionNumber}{" "}
            {schedule.state === "publishing" ? (
              "is going live now."
            ) : (
              <>
                goes live{" "}
                <time dateTime={schedule.at}>
                  {readableMoment(schedule.at)}
                </time>
                {soon ? `, ${soon}` : ""}.
              </>
            )}
          </p>
          {schedule.state === "pending" ? (
            <div className="flex flex-wrap items-center gap-2">
              <Button onClick={() => onStep("reschedule")} size="compact">
                Change scheduled revision
              </Button>
              <Button
                className="text-stop hover:bg-stop-wash hover:text-stop"
                onClick={() => onStep("unschedule")}
                size="compact"
                variant="ghost"
              >
                Cancel schedule
              </Button>
            </div>
          ) : null}
        </section>
      ) : null}

      <section className="flex flex-col gap-2">
        {publishActions(post).map((action) => (
          <Offer action={action} key={action} onChoose={() => onStep(action)} />
        ))}
      </section>

      <Sent key={stamp} postId={post.id} />
    </div>
  );
}

function Offer({
  action,
  onChoose,
}: {
  action: PublishAction;
  onChoose: () => void;
}) {
  const { icon: Icon, label, line, tone } = OFFERS[action];
  return (
    <button
      className={cn(
        "flex w-full items-start gap-3 rounded-plate p-4 text-left outline-offset-3 transition-colors duration-200 hover:bg-deep motion-reduce:transition-none",
        tone === "stop" ? "text-stop" : "text-accent",
      )}
      onClick={onChoose}
      type="button"
    >
      <Icon aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <span className="min-w-0">
        <span className="block font-ui text-ui font-medium">{label}</span>
        <span className="mt-1 block font-prose text-meta text-mute">
          {line}
        </span>
      </span>
    </button>
  );
}

function readersHave(post: Post): string {
  const standing = writerStanding(post);
  if (standing === "deleted") return "Deleted. Nobody can read this.";
  if (standing === "unpublished") return "Unpublished from the blog.";
  if (standing === "published") {
    return post.updatedPublicAt
      ? `Current revision published ${readableMoment(post.updatedPublicAt)}.`
      : "Readers have this post.";
  }
  return "Private draft. Not visible to readers.";
}

function Sent({ postId }: { postId: string }) {
  const [sent, setSent] = useState<BlogAnnouncementAttempt[]>([]);

  useEffect(() => {
    let live = true;
    void readPostDeliveries(postId).then((answer) => {
      if (live && answer.value) setSent(answer.value.attempts);
    });
    return () => {
      live = false;
    };
  }, [postId]);

  if (sent.length === 0) return null;

  return (
    <section aria-labelledby="sent-to" className="flex flex-col gap-2">
      <h3 className="font-display text-ui font-medium text-ink" id="sent-to">
        Sent
      </h3>
      <ul className="flex list-none flex-col gap-1.5">
        {sent.map((one) => (
          <li
            className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 rounded-control bg-deep px-3 py-2 font-prose text-meta"
            key={one.id}
          >
            <span className="min-w-0 text-ink wrap-anywhere">
              {one.integration}
            </span>
            <span className="text-mute">
              {announcementWord(one.announcementType)}
            </span>
            <span className={cn(troubled(one) ? "text-stop" : "text-mute")}>
              {attemptStanding(one)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function troubled(one: BlogAnnouncementAttempt): boolean {
  const state = attemptState(one);
  return state === "gaveUp" || state === "stopped";
}
