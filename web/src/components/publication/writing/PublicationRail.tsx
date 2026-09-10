"use client";

import type { LucideIcon } from "lucide-react";
import { CalendarClock, Eye, EyeOff, Send, Trash2, Undo2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Trouble } from "@/components/ui/field";
import { RailBack } from "@/components/workspace/WorkspaceRail";
import { readPostDeliveries } from "@/lib/api/posts";
import type { Post, PostDelivery } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { readableMoment } from "@/lib/dates";
import { remainingDeletionWindow } from "@/lib/deletion-window";
import {
  type PublicationAction,
  publicationActions,
  writerStanding,
} from "@/lib/post-writing";
import {
  deliveryStanding,
  deliveryState,
  eventWord,
} from "@/lib/publication-delivery";
import { howSoon } from "@/lib/schedule-time";
import {
  DeleteStep,
  PublishStep,
  RecoverStep,
  RepublishStep,
  RescheduleStep,
  type StepProps,
  UnscheduleStep,
  WithdrawStep,
} from "./PublicationSteps";

/** What the writer is doing to the post's standing right now. */
type Step = "home" | PublicationAction | "reschedule" | "unschedule";

const OFFERS: Record<
  PublicationAction,
  { icon: LucideIcon; label: string; line: string; tone?: "stop" }
> = {
  publish: {
    icon: Send,
    label: "Publish now",
    line: "Readers see this version as soon as you press the button.",
  },
  schedule: {
    icon: CalendarClock,
    label: "Schedule it",
    line: "Illarin publishes this exact version at a time you set.",
  },
  withdraw: {
    icon: EyeOff,
    label: "Take it down",
    line: "Its editions, pictures and dates stay. You can put it back at the same address.",
    tone: "stop",
  },
  republish: {
    icon: Eye,
    label: "Put it back",
    line: "It returns to the same address under the date it first published.",
  },
  delete: {
    icon: Trash2,
    label: "Delete it",
    line: "Illarin holds it for thirty days. After that the writing, the editions and the pictures are gone.",
    tone: "stop",
  },
  recover: {
    icon: Undo2,
    label: "Bring it back",
    line: "It returns exactly as it was, in the standing it was deleted from.",
  },
};

/**
 * Everything a writer does to reach readers, as one step of the rail at a time.
 * The post itself stays on the page beside it, so what is about to be published
 * is the thing being looked at.
 */
export function PublicationRail({
  admin,
  onFailure,
  onSaveFirst,
  onSettled,
  post,
  stamp,
}: {
  admin: boolean;
  onFailure: (message: string) => void;
  /** Publishing captures what is written, so anything unsaved is saved first. */
  onSaveFirst: () => Promise<number>;
  onSettled: (post: Post) => void;
  post: Post;
  /** Changes when the post settles, so what was sent is read again. */
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
      <Home
        admin={admin}
        onStep={setStep}
        post={post}
        refusal={refusal}
        stamp={stamp}
      />
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <RailBack onClick={leave}>Publication</RailBack>
      {refusal ? <Trouble>{refusal}</Trouble> : null}
      {step === "publish" ? <PublishStep {...shared} door="now" /> : null}
      {step === "schedule" ? <PublishStep {...shared} door="later" /> : null}
      {step === "withdraw" ? <WithdrawStep {...shared} /> : null}
      {step === "republish" ? <RepublishStep {...shared} /> : null}
      {step === "delete" ? <DeleteStep {...shared} /> : null}
      {step === "recover" ? <RecoverStep {...shared} /> : null}
      {step === "reschedule" ? <RescheduleStep {...shared} /> : null}
      {step === "unschedule" ? <UnscheduleStep {...shared} /> : null}
    </div>
  );
}

function Home({
  admin,
  onStep,
  post,
  refusal,
  stamp,
}: {
  admin: boolean;
  onStep: (step: Step) => void;
  post: Post;
  refusal: string;
  stamp: number;
}) {
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
          illarin.xyz/blog/{post.slug}
        </p>
        {post.deletion ? (
          <p
            className="font-prose text-meta text-stop"
            suppressHydrationWarning
          >
            {remainingDeletionWindow(post.deletion.until)}. Gone for good{" "}
            {readableMoment(post.deletion.until)}.
          </p>
        ) : null}
      </section>

      {waiting ? (
        <section className="flex flex-col gap-3">
          <h3 className="font-display text-ui font-medium text-ink">
            Waiting to go live
          </h3>
          <p className="font-prose text-meta text-mute">
            Edition {schedule.revisionNumber}{" "}
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
                Replace edition
              </Button>
              <Button
                className="text-stop hover:bg-stop-wash hover:text-stop"
                onClick={() => onStep("unschedule")}
                size="compact"
                variant="ghost"
              >
                Cancel
              </Button>
            </div>
          ) : null}
        </section>
      ) : null}

      <section className="flex flex-col gap-2">
        {publicationActions(post, admin).map((action) => (
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
  action: PublicationAction;
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

/** What readers can do with the post now, said once at the head of the rail. */
function readersHave(post: Post): string {
  const standing = writerStanding(post);
  if (standing === "deleted") return "Deleted. Nobody can read this.";
  if (standing === "withdrawn") return "Out of public view.";
  if (standing === "published") {
    return post.updatedPublicAt
      ? `Readers have the edition published ${readableMoment(post.updatedPublicAt)}.`
      : "Readers have this post.";
  }
  return "A private draft. Nobody else can open it.";
}

function Sent({ postId }: { postId: string }) {
  const [sent, setSent] = useState<PostDelivery[]>([]);

  useEffect(() => {
    let live = true;
    void readPostDeliveries(postId).then((answer) => {
      if (live && answer.value) setSent(answer.value.deliveries);
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
              {one.destination}
            </span>
            <span className="text-mute">{eventWord(one.eventType)}</span>
            <span className={cn(troubled(one) ? "text-stop" : "text-mute")}>
              {deliveryStanding(one)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

// troubled answers whether a delivery ended somewhere the writer has to know about.
function troubled(one: PostDelivery): boolean {
  const state = deliveryState(one);
  return state === "gaveUp" || state === "stopped";
}
