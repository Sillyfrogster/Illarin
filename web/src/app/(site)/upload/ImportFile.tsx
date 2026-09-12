"use client";

import { Upload } from "lucide-react";
import {
  type ChangeEvent,
  type DragEvent,
  useId,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { browserFetch } from "@/lib/api/browser-mutation";
import type { IngestOperation } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { fileWeight } from "@/lib/file-weight";

const UNCONFIRMED =
  "Confirm how to import the catalog details below, then upload the file.";

export function ImportFile({
  onAccepted,
}: {
  onAccepted: (operation: IngestOperation) => void;
}) {
  const field = useId();
  const confirmField = useId();
  const fileInput = useRef<HTMLInputElement>(null);
  const confirmInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const [over, setOver] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  function take(chosen: File | null) {
    setFile(chosen);
    setMessage("");
  }

  function choose(event: ChangeEvent<HTMLInputElement>) {
    take(event.target.files?.[0] ?? null);
  }

  function drop(event: DragEvent<HTMLLabelElement>) {
    event.preventDefault();
    setOver(false);
    take(event.dataTransfer.files?.[0] ?? null);
  }

  async function upload() {
    if (!file) return;
    if (!confirmed) {
      setMessage(UNCONFIRMED);
      confirmInput.current?.focus();
      return;
    }

    const body = new FormData();
    body.append("metadata", JSON.stringify({ confirmed: true }));
    body.append("file", file, file.name);

    setPending(true);
    setMessage("");
    try {
      const response = await browserFetch("/api/v1/assets", {
        body,
        credentials: "same-origin",
        method: "POST",
      });
      const answer = (await response.json()) as IngestOperation & {
        error?: string;
      };
      if (!response.ok) {
        setMessage(answer.error ?? "Illarin could not accept this file.");
        return;
      }
      onAccepted(answer);
    } catch {
      setMessage(
        "Illarin could not be reached. Check your connection and try again.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <section aria-labelledby={`${field}-heading`}>
      <h2
        className="font-display text-section font-medium text-ink"
        id={`${field}-heading`}
      >
        Import a file
      </h2>
      <p className="mt-2 text-ui text-mute">
        Upload a character card, lorebook, preset, theme or pack. Illarin
        detects the format and imports its details.
      </p>

      <input
        className="peer sr-only"
        id={field}
        onChange={choose}
        ref={fileInput}
        type="file"
      />
      <label
        className={cn(
          "mt-5 flex min-h-48 cursor-pointer flex-col items-center justify-center gap-3 rounded-plate border border-dashed px-6 py-8 text-center transition-colors duration-200 motion-reduce:transition-none",
          "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-accent peer-focus-visible:outline-offset-3",
          over
            ? "border-accent bg-accent-wash"
            : "border-edge bg-inset hover:border-accent hover:bg-accent-wash",
        )}
        htmlFor={field}
        onDragLeave={() => setOver(false)}
        onDragOver={(event) => {
          event.preventDefault();
          setOver(true);
        }}
        onDrop={drop}
      >
        <Upload
          aria-hidden="true"
          className="text-accent"
          size={28}
          strokeWidth={1.35}
        />
        <span className="text-ui font-medium text-ink wrap-anywhere">
          {file ? file.name : "Drop a file here, or choose one"}
        </span>
        <span className="text-meta text-mute">
          {file
            ? `${fileWeight(file.size)} · choose a different file`
            : "Your original file, exactly as it is"}
        </span>
      </label>

      {file ? (
        <div className="mt-5 flex flex-col gap-4">
          <label
            className={cn(
              "flex min-h-11 cursor-pointer items-start gap-3 rounded-control p-3 text-ui text-ink",
              confirmed ? "bg-accent-wash" : "bg-deep",
            )}
            htmlFor={confirmField}
          >
            <input
              checked={confirmed}
              className="mt-1 size-4 shrink-0 accent-[var(--v-action)]"
              id={confirmField}
              onChange={(event) => {
                setConfirmed(event.target.checked);
                setMessage("");
              }}
              ref={confirmInput}
              type="checkbox"
            />
            <span className="min-w-0">
              Use the catalog details Illarin finds in this file. I can change
              them afterwards.
            </span>
          </label>

          {message ? (
            <p
              className="rounded-control bg-stop-wash p-3 text-meta text-ink"
              role="alert"
            >
              {message}
            </p>
          ) : null}

          <div className="flex flex-wrap items-center gap-2">
            <Button loading={pending} onClick={upload} variant="primary">
              {pending ? "Uploading…" : "Upload file"}
            </Button>
            <Button
              disabled={pending}
              onClick={() => {
                take(null);
                setConfirmed(false);
                if (fileInput.current) fileInput.current.value = "";
              }}
              variant="ghost"
            >
              Clear
            </Button>
          </div>
        </div>
      ) : message ? (
        <p
          className="mt-5 rounded-control bg-stop-wash p-3 text-meta text-ink"
          role="alert"
        >
          {message}
        </p>
      ) : null}
    </section>
  );
}
