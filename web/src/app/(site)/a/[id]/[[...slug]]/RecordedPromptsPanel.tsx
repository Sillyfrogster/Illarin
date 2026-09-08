"use client";

import { ChevronRight, RotateCcw } from "lucide-react";
import { useState } from "react";
import {
  fetchProtectionMismatches,
  type ProtectionMismatch,
  resolvePromptCorrespondence,
} from "@/lib/api/query";
import styles from "./RecordedPromptsPanel.module.css";

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
    <div className={styles.control}>
      <button
        className={styles.launch}
        type="button"
        aria-expanded={open}
        aria-controls="recorded-prompts-menu"
        onClick={() => (open ? setOpen(false) : void read())}
      >
        <span className={styles.launchCopy}>
          <strong>Match your sealed prompts to older versions</strong>
          <span>
            A version Illarin cannot match shows readers no prompts at all
          </span>
        </span>
        <ChevronRight
          className={open ? styles.chevronOpen : undefined}
          size={18}
          aria-hidden="true"
        />
      </button>

      {open ? (
        <div className={styles.menu} id="recorded-prompts-menu">
          <p className={styles.menuLead}>
            Your sealed prompts changed identity, so Illarin cannot tell which
            prompt in an older version they are. Until you say, that version
            keeps every prompt hidden and refuses downloads.
          </p>
          {versions === null ? (
            <p className={styles.menuState}>Reading the recorded versions…</p>
          ) : versions.length === 0 ? (
            <p className={styles.menuState}>
              Every recorded version matches your sealed prompts.
            </p>
          ) : (
            <ul className={styles.versions}>
              {versions.map((version) => (
                <li key={version.version.id}>
                  <h3>
                    Update {version.version.number}
                    {version.version.summary
                      ? `: ${version.version.summary}`
                      : ""}
                  </h3>
                  {version.unmatched.map((prompt) => (
                    <label key={prompt.id} className={styles.match}>
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
                    className={styles.settle}
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
            <p className={styles.error} role="alert">
              {message}
            </p>
          ) : null}
          {versions === null ? (
            <button
              className={styles.retry}
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
