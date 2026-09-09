"use client";

import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { saveAssetIdentity } from "@/lib/api/query";
import { useWorkingCopy } from "@/lib/working-copy";

/** The three states the adult content question has while an asset is a draft. */
const ANSWERS: { value: boolean | null; label: string }[] = [
  { value: null, label: "Not yet" },
  { value: false, label: "No" },
  { value: true, label: "Yes" },
];

export function IdentityPanel({
  assetId,
  initialName,
  initialIsNsfw,
  isDraft,
}: {
  assetId: string;
  initialName: string;
  initialIsNsfw: boolean | null;
  isDraft: boolean;
}) {
  const candidate = useWorkingCopy();
  const router = useRouter();
  const [name, setName] = useState(initialName);
  const [isNsfw, setIsNsfw] = useState(initialIsNsfw);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [saved, setSaved] = useState(false);

  const answers = isDraft ? ANSWERS : ANSWERS.filter((a) => a.value !== null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;
    setPending(true);
    setMessage("");
    setSaved(false);
    try {
      await saveAssetIdentity(candidate, assetId, { name, isNsfw });
      setSaved(true);
      router.refresh();
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "The details could not be saved. Try again.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <section
      aria-labelledby="identity-heading"
      className="rounded-plate bg-deep p-4"
    >
      <h2 className="text-ui font-medium text-ink" id="identity-heading">
        Name and rating
      </h2>
      <form onSubmit={submit}>
        <label
          className="mt-4 block text-meta font-medium text-ink"
          htmlFor="asset-name"
        >
          Name
        </label>
        <input
          className="mt-2 h-11 w-full rounded-control bg-plane px-3 text-ui text-ink outline-offset-3 disabled:opacity-45"
          id="asset-name"
          value={name}
          placeholder="Name this page"
          onChange={(event) => {
            setSaved(false);
            setName(event.target.value);
          }}
          disabled={pending}
        />

        <fieldset className="mt-4" id="adult-content-answer">
          <legend className="text-meta font-medium text-ink">
            Adult content
          </legend>
          <div className="mt-2 flex flex-wrap gap-2">
            {answers.map((answer) => (
              <button
                key={answer.label}
                type="button"
                className="min-h-11 rounded-control bg-plane px-4 text-ui text-ink outline-offset-3 aria-pressed:bg-action aria-pressed:text-on-accent disabled:opacity-45"
                aria-pressed={answer.value === isNsfw}
                onClick={() => {
                  setSaved(false);
                  setIsNsfw(answer.value);
                }}
                disabled={pending}
              >
                {answer.label}
              </button>
            ))}
          </div>
        </fieldset>
        <p className="mt-3 text-meta text-mute">
          {isNsfw === null
            ? "Publishing will not go through until this is answered. There is no default."
            : "You can change this after publishing."}
        </p>

        <button
          className="mt-4 min-h-11 rounded-control bg-action px-5 text-ui font-medium text-on-accent outline-offset-3 disabled:opacity-45"
          disabled={pending}
          type="submit"
        >
          {pending ? "Saving…" : "Save"}
        </button>
        {saved ? <p className="mt-2 text-meta text-accent">Saved.</p> : null}
        {message ? (
          <p className="mt-2 text-meta text-stop" role="alert">
            {message}
          </p>
        ) : null}
      </form>
    </section>
  );
}
