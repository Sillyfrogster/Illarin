"use client";

import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CircleSlash,
  Clock,
} from "lucide-react";
import { useState } from "react";
import { MorphingDisclosure } from "@/components/ui/morphing-disclosure";
import { Select } from "@/components/ui/select";
import { readDeliveryAttempts, replayDelivery } from "@/lib/api/publication";
import type {
  PostDelivery,
  PostDeliveryAttempt,
  PostDeliveryState,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { shortMoment } from "@/lib/dates";
import {
  type DeliveryState,
  deliveryStanding,
  deliveryState,
  eventWord,
} from "@/lib/publication-delivery";
import { canReplay, nothingDelivered } from "@/lib/publication-register";
import { Nothing, PanelHead, Row, RowAction, RowMark, Rows } from "./RowParts";

/** The views the authority narrows announcement work to. */
export const VIEWS: { key: string; state?: PostDeliveryState; word: string }[] =
  [
    { key: "all", word: "Everything" },
    { key: "failed", state: "failed", word: "Stopped short" },
    { key: "pending", state: "pending", word: "On the way" },
    { key: "delivered", state: "delivered", word: "Arrived" },
    { key: "unconfirmed", state: "unconfirmed", word: "Unconfirmed" },
  ];

const MARKS: Record<
  DeliveryState,
  { icon: typeof Clock; tone: "accent" | "quiet" | "stop" }
> = {
  arrived: { icon: CircleCheck, tone: "accent" },
  gaveUp: { icon: CircleAlert, tone: "stop" },
  stopped: { icon: CircleSlash, tone: "quiet" },
  unconfirmed: { icon: CircleHelp, tone: "quiet" },
  waiting: { icon: Clock, tone: "quiet" },
};

/** Every announcement one publication sent, and what became of it. */
export function DeliveryRows({
  deliveries,
  onChanged,
  onFailure,
  onView,
  view,
}: {
  deliveries: PostDelivery[];
  onChanged: (delivery: PostDelivery) => void;
  onFailure: (message: string) => void;
  onView: (view: string, state?: PostDeliveryState) => void;
  view: string;
}) {
  const [tried, setTried] = useState<Record<string, PostDeliveryAttempt[]>>({});
  const [working, setWorking] = useState("");

  async function look(one: PostDelivery) {
    if (tried[one.id]) return;
    const answer = await readDeliveryAttempts(one.id);
    const made = answer.value?.attempts;
    if (made) setTried((held) => ({ ...held, [one.id]: made }));
  }

  async function again(one: PostDelivery) {
    setWorking(one.id);
    const answer = await replayDelivery(one.id);
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
              htmlFor="delivery-view"
            >
              Showing
            </label>
            <Select
              id="delivery-view"
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

      {deliveries.length === 0 ? (
        <Nothing>{nothingDelivered(view)}</Nothing>
      ) : (
        <Rows>
          {deliveries.map((one) => {
            const state = deliveryState(one);
            const mark = MARKS[state];
            const attempts = tried[one.id];
            return (
              <Row
                aside={
                  canReplay(one) ? (
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
                    <span>{one.destination}</span>
                    <span>{eventWord(one.eventType)}</span>
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
                standing={deliveryStanding(one)}
                title={one.postTitle}
              >
                {one.attempts > 0 ? (
                  <div
                    className="relative mt-3"
                    onClickCapture={() => void look(one)}
                  >
                    <MorphingDisclosure
                      summary={`${one.attempts === 1 ? "1 attempt" : `${one.attempts} attempts`}${one.run > 1 ? ` over ${one.run} runs` : ""}`}
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
                                {made.detail || "It took it."}
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
                          Reading the attempts…
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
