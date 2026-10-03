"use client";

import { RotateCcw } from "lucide-react";
import { useState } from "react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import {
  fetchPrivatePromptMismatches,
  type PrivatePromptMismatch,
  resolvePromptCorrespondence,
} from "@/lib/api/query";

const ABSENT = "absent";

export function RecordedPromptsPanel({ workId }: { workId: string }) {
  const [open, setOpen] = useState(false);
  const [versions, setVersions] = useState<PrivatePromptMismatch[] | null>(
    null,
  );
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(0);
  const [message, setMessage] = useState("");

  async function read() {
    setOpen(true);
    if (versions !== null) return;
    setMessage("");
    setVersions(await fetchPrivatePromptMismatches(workId));
  }

  async function settle(version: PrivatePromptMismatch) {
    if (pending) return;
    setPending(version.version.number);
    setMessage("");
    try {
      await resolvePromptCorrespondence(
        workId,
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
          : "The prompt matches could not be saved. Try again.",
      );
    } finally {
      setPending(0);
    }
  }

  return (
    <Accordion
      className="mt-3.5 border-rule border-t pt-2"
      collapsible
      onValueChange={(value) => (value ? void read() : setOpen(false))}
      type="single"
      value={open ? "open" : ""}
    >
      <AccordionItem value="open">
        <AccordionTrigger>
          Match your private prompts to older versions
        </AccordionTrigger>
        <p className="pl-6 text-meta text-mute">
          Unmatched versions hide all prompts and block downloads
        </p>
        <AccordionContent>
          <div className="pt-3 pb-3.5 pl-6">
            <p className={"text-meta text-mute"}>
              Your private prompts changed identity, so Illarin cannot tell
              which prompt in an older version they are. Until you say, that
              version keeps every prompt hidden and refuses downloads.
            </p>
            {versions === null ? (
              <p className={"mt-3 text-meta text-mute italic"}>
                Loading older versions…
              </p>
            ) : versions.length === 0 ? (
              <p className={"mt-3 text-meta text-mute italic"}>
                Every recorded version matches your private prompts.
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
                      Version {version.version.number}
                      {version.version.summary
                        ? `: ${version.version.summary}`
                        : ""}
                    </h3>
                    {version.unmatched.map((prompt) => (
                      <Field
                        className="mt-2"
                        key={prompt.id}
                        label={prompt.name}
                      >
                        <Select
                          onValueChange={(answer) =>
                            setAnswers((current) => ({
                              ...current,
                              [`${version.version.id}:${prompt.id}`]: answer,
                            }))
                          }
                          options={[
                            {
                              value: ABSENT,
                              label: "This version did not carry it",
                            },
                            ...version.recorded.map((choice) => ({
                              value: choice.id,
                              label: choice.name,
                            })),
                          ]}
                          value={answerKey(answers, version, prompt.id)}
                        />
                      </Field>
                    ))}
                    <Button
                      className="mt-3"
                      disabled={pending !== 0}
                      onClick={() => void settle(version)}
                    >
                      {pending === version.version.number
                        ? "Saving matches…"
                        : `Save matches for version ${version.version.number}`}
                    </Button>
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
              <Button
                className="mt-3"
                onClick={() => {
                  setVersions(null);
                  void read();
                }}
              >
                <RotateCcw aria-hidden="true" />
                Try again
              </Button>
            ) : null}
          </div>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}

function answerKey(
  answers: Record<string, string>,
  version: PrivatePromptMismatch,
  promptId: string,
): string {
  return answers[`${version.version.id}:${promptId}`] ?? ABSENT;
}

function answerFor(
  answers: Record<string, string>,
  version: PrivatePromptMismatch,
  promptId: string,
): string | undefined {
  const chosen = answerKey(answers, version, promptId);
  return chosen === ABSENT ? undefined : chosen;
}
