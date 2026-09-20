"use client";

import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CircleSlash,
  Clock,
} from "lucide-react";
import { useState } from "react";
import {
  Nothing,
  PanelHead,
  Row,
  RowAction,
  RowMark,
  Rows,
} from "@/components/register/RowParts";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { Select } from "@/components/ui/select";
import {
  readAnnouncementTries,
  replayAnnouncementAttempt,
} from "@/lib/api/blog";
import type {
  BlogAnnouncementAttempt,
  BlogAnnouncementAttemptState,
  BlogAnnouncementTry,
} from "@/lib/api/query";
import {
  type AttemptState,
  attemptStanding,
  attemptState,
} from "@/lib/attempt-standing";
import { canReplay, nothingDelivered } from "@/lib/blog-admin";
import { announcementWord } from "@/lib/blog-announcement-attempt";
import { cn } from "@/lib/cn";
import { shortMoment } from "@/lib/dates";
import { DiscordRepairControls } from "./DiscordRepairControls";

export const VIEWS: {
  key: string;
  state?: BlogAnnouncementAttemptState;
  word: string;
}[] = [
  { key: "all", word: "All announcements" },
  { key: "failed", state: "failed", word: "Failed" },
  { key: "pending", state: "pending", word: "Pending" },
  { key: "delivered", state: "delivered", word: "Delivered" },
  { key: "unconfirmed", state: "unconfirmed", word: "Unconfirmed" },
];

const MARKS: Record<
  AttemptState,
  { icon: typeof Clock; tone: "accent" | "quiet" | "stop" }
> = {
  arrived: { icon: CircleCheck, tone: "accent" },
  gaveUp: { icon: CircleAlert, tone: "stop" },
  stopped: { icon: CircleSlash, tone: "quiet" },
  unconfirmed: { icon: CircleHelp, tone: "quiet" },
  waiting: { icon: Clock, tone: "quiet" },
};

export function AttemptRows({
  attempts,
  onChanged,
  onFailure,
  onView,
  view,
}: {
  attempts: BlogAnnouncementAttempt[];
  onChanged: (attempt: BlogAnnouncementAttempt) => void;
  onFailure: (message: string) => void;
  onView: (view: string, state?: BlogAnnouncementAttemptState) => void;
  view: string;
}) {
  const [tried, setTried] = useState<Record<string, BlogAnnouncementTry[]>>({});
  const [working, setWorking] = useState("");

  async function look(one: BlogAnnouncementAttempt) {
    if (tried[one.id]) return;
    const answer = await readAnnouncementTries(one.id);
    const made = answer.value?.tries;
    if (made) setTried((held) => ({ ...held, [one.id]: made }));
  }

  async function again(one: BlogAnnouncementAttempt) {
    setWorking(one.id);
    const answer = await replayAnnouncementAttempt(one.id);
    setWorking("");
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onFailure("");
    setTried((held) => ({ ...held, [one.id]: [] }));
    onChanged(answer.value);
  }

  return (
    <>
      <PanelHead
        action={
          <span className="flex min-w-0 items-center gap-2">
            <label
              className="shrink-0 font-ui text-meta text-mute"
              htmlFor="attempt-view"
            >
              Showing
            </label>
            <Select
              id="attempt-view"
              onChange={(event) => {
                const next = VIEWS.find(
                  (one) => one.key === event.target.value,
                );
                if (next) onView(next.key, next.state);
              }}
              value={view}
            >
              {VIEWS.map((one) => (
                <option key={one.key} value={one.key}>
                  {one.word}
                </option>
              ))}
            </Select>
          </span>
        }
        id="register-heading"
        title="Announcements"
      />

      {attempts.length === 0 ? (
        <Nothing>{nothingDelivered(view)}</Nothing>
      ) : (
        <Rows>
          {attempts.map((one) => {
            const state = attemptState(one);
            const mark = MARKS[state];
            const attempts = tried[one.id];
            return (
              <Row
                aside={
                  canReplay(one) && one.type !== "discord" ? (
                    <RowAction
                      busy={working === one.id}
                      onClick={() => void again(one)}
                    >
                      {working === one.id ? "Sending…" : "Send again"}
                    </RowAction>
                  ) : null
                }
                facts={
                  <>
                    <span>{one.integration}</span>
                    <span>{announcementWord(one.announcementType)}</span>
                  </>
                }
                key={one.id}
                lead={
                  <RowMark tone={mark.tone}>
                    <mark.icon
                      aria-hidden="true"
                      className="size-5"
                      strokeWidth={1.8}
                    />
                  </RowMark>
                }
                standing={attemptStanding(one)}
                title={one.postTitle}
              >
                {one.type === "discord" && one.settledAt && !one.removed ? (
                  <DiscordRepairControls attempt={one} />
                ) : null}
                {one.tries > 0 ? (
                  <div
                    className="relative mt-3"
                    onClickCapture={() => void look(one)}
                  >
                    <MorphingDisclosure
                      summary={`${one.tries === 1 ? "1 try" : `${one.tries} tries`}${one.run > 1 ? ` over ${one.run} runs` : ""}`}
                    >
                      {attempts ? (
                        <ol className="mt-3 flex list-none flex-col">
                          {attempts.map((made) => (
                            <li
                              className="flex flex-wrap items-baseline gap-x-3 gap-y-0.5 border-rule/45 py-2 font-prose text-meta not-first:border-t"
                              key={`${made.run}-${made.number}`}
                            >
                              <span className="shrink-0 font-mono text-label text-mute tabular-nums">
                                {made.run > 1 ? `${made.run}.` : ""}
                                {made.number}
                              </span>
                              <span
                                className={cn(
                                  "min-w-0 flex-1 wrap-anywhere",
                                  made.outcome === "delivered"
                                    ? "text-ink"
                                    : "text-stop",
                                )}
                              >
                                {made.detail || "The announcement arrived."}
                              </span>
                              <span className="shrink-0 text-mute tabular-nums">
                                {shortMoment(made.attemptedAt)} · {made.tookMs}
                                ms
                              </span>
                            </li>
                          ))}
                        </ol>
                      ) : (
                        <p className="mt-3 font-prose text-meta text-mute">
                          Loading the tries…
                        </p>
                      )}
                    </MorphingDisclosure>
                  </div>
                ) : null}
              </Row>
            );
          })}
        </Rows>
      )}
    </>
  );
}
