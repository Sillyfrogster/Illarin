"use client";

import { ChevronRight, RotateCcw } from "lucide-react";
import { useState } from "react";
import {
  fetchProtectionMismatches,
  type ProtectionMismatch,
  resolvePromptCorrespondence,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";

/** ABSENT is the answer that a recorded version never carried a sealed prompt. */
const ABSENT = "absent";

/** Lets an owner say which recorded prompt each sealed prompt is, on a version Illarin cannot match. */
export function RecordedPromptsPanel({ assetId }: { assetId: string }) {
  const [open, setOpen] = useState(false);
  const [versions, setVersions] = useState<ProtectionMismatch[] | null>(null);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(0);
  const [message, setMessage] = useState("");

  async function read() {
    setOpen(true);
    if (versions !== null) return;
    setMessage("");
    setVersions(await fetchProtectionMismatches(assetId));
  }

  async function settle(version: ProtectionMismatch) {
    if (pending) return;
    setPending(version.version.number);
    setMessage("");
    try {
      await resolvePromptCorrespondence(
        assetId,
        version.version.number,
        version.unmatched.map((prompt) => ({
          current: prompt.id,
          recorded: answerFor(answers, version, prompt.id),
        })),
      );
      setVersions(
        (current) =>
          current?.filter(
            (held) => held.version.number !== version.version.number,
          ) ?? current,
      );
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "That version could not be settled. Try again.",
      );
    } finally {
      setPending(0);
    }
  }

  return (
    <div className={"mt-3.5 border-rule border-t"}>
      <button
        className={
          "flex min-h-16 w-full items-center justify-between gap-3.5 py-3 text-left text-ink outline-offset-3 hover:text-accent"
        }
        type="button"
        aria-expanded={open}
        aria-controls="recorded-prompts-menu"
        onClick={() => (open ? setOpen(false) : void read())}
      >
        <span
          className={
            "grid gap-1 [&>span]:text-meta [&>span]:text-mute [&>strong]:text-ui [&>strong]:font-medium"
          }
        >
          <strong>Match your sealed prompts to older versions</strong>
          <span>
            A version Illarin cannot match shows readers no prompts at all
          </span>
        </span>
        <ChevronRight
          className={cn(
            "shrink-0 text-mute transition-transform duration-200 motion-reduce:transition-none",
            open && "rotate-90",
          )}
          size={18}
          aria-hidden="true"
        />
      </button>

      {open ? (
        <div className={"pb-3.5"} id="recorded-prompts-menu">
          <p className={"text-meta text-mute"}>
            Your sealed prompts changed identity, so Illarin cannot tell which
            prompt in an older version they are. Until you say, that version
            keeps every prompt hidden and refuses downloads.
          </p>
          {versions === null ? (
            <p className={"mt-3 text-meta text-mute italic"}>
              Reading the recorded versions…
            </p>
          ) : versions.length === 0 ? (
            <p className={"mt-3 text-meta text-mute italic"}>
              Every recorded version matches your sealed prompts.
            </p>
          ) : (
            <ul
              className={
                "mt-3 flex list-none flex-col gap-5 [&_h3]:text-ui [&_h3]:font-medium [&_h3]:text-ink"
              }
            >
              {versions.map((version) => (
                <li key={version.version.id}>
                  <h3>
                    Update {version.version.number}
                    {version.version.summary
                      ? `: ${version.version.summary}`
                      : ""}
                  </h3>
                  {version.unmatched.map((prompt) => (
                    <label
                      key={prompt.id}
                      className={
                        "mt-2 grid gap-1 text-meta text-mute [&_select]:h-11 [&_select]:rounded-control [&_select]:bg-deep [&_select]:px-3 [&_select]:text-ui [&_select]:text-ink"
                      }
                    >
                      <span>{prompt.name}</span>
                      <select
                        value={answerKey(answers, version, prompt.id)}
                        onChange={(event) =>
                          setAnswers((current) => ({
                            ...current,
                            [`${version.version.id}:${prompt.id}`]:
                              event.target.value,
                          }))
                        }
                      >
                        <option value={ABSENT}>
                          This version did not carry it
                        </option>
                        {version.recorded.map((choice) => (
                          <option key={choice.id} value={choice.id}>
                            {choice.name}
                          </option>
                        ))}
                      </select>
                    </label>
                  ))}
                  <button
                    className={
                      "mt-3 inline-flex min-h-11 items-center rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3 hover:bg-rule/45 disabled:opacity-45"
                    }
                    type="button"
                    disabled={pending !== 0}
                    onClick={() => void settle(version)}
                  >
                    {pending === version.version.number
                      ? "Settling…"
                      : `Settle update ${version.version.number}`}
                  </button>
                </li>
              ))}
            </ul>
          )}
          {message ? (
            <p className={"mt-3 text-meta text-stop"} role="alert">
              {message}
            </p>
          ) : null}
          {versions === null ? (
            <button
              className={
                "mt-3 inline-flex min-h-11 items-center gap-2 rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3"
              }
              type="button"
              onClick={() => {
                setVersions(null);
                void read();
              }}
            >
              <RotateCcw size={14} aria-hidden="true" />
              Try again
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

/** answerKey is what the select shows, defaulting to the absent answer. */
function answerKey(
  answers: Record<string, string>,
  version: ProtectionMismatch,
  promptId: string,
): string {
  return answers[`${version.version.id}:${promptId}`] ?? ABSENT;
}

/** answerFor is the recorded prompt chosen, and nothing where none was. */
function answerFor(
  answers: Record<string, string>,
  version: ProtectionMismatch,
  promptId: string,
): string | undefined {
  const chosen = answerKey(answers, version, promptId);
  return chosen === ABSENT ? undefined : chosen;
}
