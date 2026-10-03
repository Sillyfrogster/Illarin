"use client";

import { ImagePlus, Plus, X } from "lucide-react";
import { useRouter } from "next/navigation";
import {
  type DragEvent,
  type KeyboardEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import { Segmented } from "@/components/ui/segmented";
import { Spinner } from "@/components/ui/spinner";
import { addWorkImage } from "@/lib/api/query";
import { cn, focusRing } from "@/lib/cn";
import { useDraftedChanges } from "@/lib/drafted-changes";
import {
  BLURB_LIMIT,
  blurbCharacterCount,
  blurbLimitMessage,
  TAG_LIMIT,
  tagTrouble,
} from "./details";
import { EditableText } from "./EditableText";
import { useWorkspace } from "./state";

type Rating = "no" | "yes";

const RATINGS: { value: Rating; label: string }[] = [
  { label: "No adult content", value: "no" },
  { label: "Adult content", value: "yes" },
];

function useDetails() {
  const workspace = useWorkspace();
  const details = workspace.details;
  return {
    details,
    rating: (details.isNsfw === null
      ? null
      : details.isNsfw
        ? "yes"
        : "no") as Rating | null,
    setBlurb: (blurb: string) => workspace.writeDetails({ ...details, blurb }),
    setRating: (rating: Rating) =>
      workspace.writeDetails({ ...details, isNsfw: rating === "yes" }),
    addTag: (tag: string) => {
      const trouble = tagTrouble(details.tags, tag);
      if (trouble) return trouble;
      workspace.writeDetails({ ...details, tags: [...details.tags, tag] });
      return "";
    },
    removeTag: (index: number) =>
      workspace.writeDetails({
        ...details,
        tags: details.tags.filter((_, at) => at !== index),
      }),
  };
}

/** CoverDrop lies over the cover while editing: drop a picture on it, or press it to choose one, and it becomes the cover. */
export function CoverDrop({ hasCover }: { hasCover: boolean }) {
  const candidate = useDraftedChanges();
  const workspace = useWorkspace();
  const router = useRouter();
  const file = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);
  const [sending, setSending] = useState(false);

  async function send(chosen: File | null | undefined) {
    if (!chosen || sending) return;
    if (!chosen.type.startsWith("image/")) {
      workspace.say("That file is not a picture. Use a PNG, JPEG or WebP.");
      return;
    }
    setSending(true);
    try {
      await addWorkImage(candidate, workspace.workId, chosen, "avatar");
      workspace.say("Cover replaced. Browse crops it to 3:4.");
      router.refresh();
    } catch (error) {
      workspace.say(
        error instanceof Error
          ? error.message
          : "The cover could not be added. Try again.",
      );
    } finally {
      setSending(false);
      if (file.current) file.current.value = "";
    }
  }

  function dropped(event: DragEvent) {
    event.preventDefault();
    setOver(false);
    void send(event.dataTransfer.files[0]);
  }

  return (
    <>
      <input
        accept="image/*"
        hidden
        onChange={(event) => void send(event.target.files?.[0])}
        ref={file}
        type="file"
      />
      <button
        aria-label={hasCover ? "Replace the cover" : "Add a cover"}
        className={cn(
          "group/cover absolute inset-0 z-10 flex cursor-pointer flex-col items-center justify-end gap-2 rounded-art p-5 text-center transition-colors duration-200",
          over || sending
            ? "bg-media/70"
            : "bg-transparent hover:bg-media/45 focus-visible:bg-media/45",
          focusRing,
          "focus-visible:ring-2",
        )}
        onClick={() => file.current?.click()}
        onDragEnter={(event) => {
          event.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDragOver={(event) => event.preventDefault()}
        onDrop={dropped}
        type="button"
      >
        <span
          className={cn(
            "inline-flex h-control items-center gap-2 rounded-control bg-field px-4 text-ui font-medium text-ink shadow-cover transition-[opacity,translate] duration-200",
            over || sending
              ? "translate-y-0 opacity-100"
              : "translate-y-1 opacity-0 group-hover/cover:translate-y-0 group-hover/cover:opacity-100 group-focus-visible/cover:translate-y-0 group-focus-visible/cover:opacity-100 pointer-coarse:translate-y-0 pointer-coarse:opacity-100",
          )}
        >
          {sending ? (
            <Spinner />
          ) : (
            <ImagePlus aria-hidden="true" className="size-4" />
          )}
          {sending
            ? "Adding the cover"
            : over
              ? "Drop to use as the cover"
              : hasCover
                ? "Replace cover"
                : "Add a cover"}
        </span>
        <span
          className={cn(
            "text-label text-on-media transition-opacity duration-200",
            over
              ? "opacity-100"
              : "opacity-0 group-hover/cover:opacity-100 group-focus-visible/cover:opacity-100",
          )}
        >
          Drop a picture here. Browse crops it to 3:4.
        </span>
      </button>
    </>
  );
}

/** InlineDetails is the arrange-map editor's way to edit the blurb, tags and rating: in place, where readers see them. */
export function InlineDetails({ typeName }: { typeName: string }) {
  const workspace = useWorkspace();
  const { details, rating, setBlurb, setRating, addTag, removeTag } =
    useDetails();
  const active = workspace.cursor === "identity:blurb";
  const count = blurbCharacterCount(details.blurb);
  const trouble = blurbLimitMessage(details.blurb);

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <EditableText
          active={active}
          activate={() => workspace.setCursor("identity:blurb")}
          className="font-prose text-lede text-ink"
          done={() => workspace.setCursor(null)}
          label="Blurb"
          live
          onChange={setBlurb}
          placeholder={`Describe this ${typeName} in a few words`}
          value={details.blurb}
        />
        {active || trouble ? (
          <p
            className={cn(
              "text-label tabular-nums",
              trouble ? "text-stop" : "text-mute",
            )}
          >
            {trouble ||
              `${count} / ${BLURB_LIMIT} characters · readers see this in Browse`}
          </p>
        ) : null}
      </div>
      <InlineTags onAdd={addTag} onRemove={removeTag} tags={details.tags} />
      <div className="flex flex-col gap-2" id="adult-content-answer">
        <Segmented
          aria-label="Adult content"
          className="self-start"
          onValueChange={setRating}
          options={RATINGS}
          value={rating}
        />
        {rating === null ? (
          <p className="text-label text-mute">
            Say whether this is adult content before you publish.
          </p>
        ) : null}
      </div>
    </>
  );
}

/** InlineTags shows the tags as readers see them, each with a cross while editing, and a chip at the end that turns into a field for the next one. */
function InlineTags({
  tags,
  onAdd,
  onRemove,
}: {
  tags: string[];
  onAdd: (tag: string) => string;
  onRemove: (index: number) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [text, setText] = useState("");
  const [trouble, setTrouble] = useState("");
  const field = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (adding) field.current?.focus();
  }, [adding]);

  function commit() {
    const tag = text.trim();
    if (!tag) return true;
    const problem = onAdd(tag);
    setTrouble(problem);
    if (problem) return false;
    setText("");
    return true;
  }

  function keys(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setText("");
      setTrouble("");
      setAdding(false);
    }
    if (event.key === "Backspace" && text === "" && tags.length > 0) {
      event.preventDefault();
      setText(tags[tags.length - 1]);
      onRemove(tags.length - 1);
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <ul aria-label="Tags" className="flex list-none flex-wrap gap-1.5">
        {tags.map((tag, index) => (
          <li
            className="inline-flex h-control-compact items-center gap-0.5 rounded-chip bg-fill pl-2.5 text-meta text-ink"
            key={tag}
          >
            {tag}
            <button
              aria-label={`Remove ${tag}`}
              className={cn(
                "inline-flex size-control-compact items-center justify-center rounded-chip text-mute hover:text-ink",
                focusRing,
              )}
              onClick={() => onRemove(index)}
              type="button"
            >
              <X aria-hidden="true" className="size-3.5" />
            </button>
          </li>
        ))}
        <li>
          {adding ? (
            <input
              aria-label="New tag"
              className="h-control-compact w-36 rounded-chip border border-accent bg-fill px-2.5 text-meta text-ink outline-none placeholder:text-mute"
              onBlur={() => {
                if (commit()) setAdding(false);
              }}
              onChange={(event) => {
                setText(event.target.value);
                setTrouble("");
              }}
              onKeyDown={keys}
              placeholder="Type, then Enter"
              ref={field}
              value={text}
            />
          ) : tags.length < TAG_LIMIT ? (
            <button
              className={cn(
                "inline-flex h-control-compact items-center gap-1 rounded-chip px-2.5 text-meta font-medium text-accent hover:bg-accent-wash",
                focusRing,
              )}
              onClick={() => setAdding(true)}
              type="button"
            >
              <Plus aria-hidden="true" className="size-3.5" />
              {tags.length === 0 ? "Add tags" : "Add a tag"}
            </button>
          ) : null}
        </li>
      </ul>
      {trouble ? <p className="text-label text-stop">{trouble}</p> : null}
    </div>
  );
}
