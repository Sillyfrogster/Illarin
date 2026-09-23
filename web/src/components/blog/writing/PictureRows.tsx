"use client";

import type { Editor } from "@tiptap/react";
import {
  ArrowLeft,
  ArrowRight,
  ImagePlus,
  ImageUp,
  Trash2,
} from "lucide-react";
import Image from "next/image";
import { useId, useRef, useState } from "react";
import type { PostMedia } from "@/lib/api/query";
import { POST_PICTURE_TEXT_LIMIT } from "@/lib/post-body";
import type { PictureAttributes } from "./picture-nodes";
import { Choice, Pair, Row, RowInput, RowNote } from "./RowParts";
import type { Controls } from "./use-controls";

export type Upload = (file: File) => Promise<PostMedia | null>;

export type PictureIntent = "picture" | "gallery" | "add" | "replace";

const INTENT_LABELS: Record<PictureIntent, string> = {
  picture: "Picture",
  gallery: "Gallery",
  add: "Another picture",
  replace: "Replace the picture",
};

const FILE = "sr-only";

export function PictureRow({
  editor,
  intent,
  said,
  onClose,
  onUpload,
}: {
  editor: Editor;
  intent: PictureIntent;
  said: Controls["picture"];
  onClose: () => void;
  onUpload: Upload;
}) {
  const field = useId();
  const chooser = useRef<HTMLInputElement>(null);
  const description = useRef<HTMLInputElement>(null);
  const replacing = intent === "replace";
  const [held, setHeld] = useState<PostMedia | null>(null);
  const [alt, setAlt] = useState(replacing ? said.alt : "");
  const [caption, setCaption] = useState(replacing ? said.caption : "");
  const [busy, setBusy] = useState(false);
  const [refusal, setRefusal] = useState("");

  async function choose(file: File | undefined) {
    if (!file) return;
    setBusy(true);
    setRefusal("");
    const added = await onUpload(file);
    setBusy(false);
    if (!added) {
      setRefusal("That picture could not be uploaded. Try another file.");
      return;
    }
    setHeld(added);
    description.current?.focus();
  }

  function place() {
    if (!held || !alt.trim()) return;
    const picture: PictureAttributes = {
      mediaId: held.id,
      alt: alt.trim(),
      caption: caption.trim(),
      src: held.url,
      width: held.width,
      height: held.height,
    };
    const writing = editor.chain().focus();
    if (replacing) {
      const node = editor.isActive("galleryImage") ? "galleryImage" : "image";
      writing.updateAttributes(node, picture).run();
    } else {
      writing
        .insertContentAt(editor.state.selection.to, placement(intent, picture))
        .run();
    }
    onClose();
  }

  return (
    <Row label={INTENT_LABELS[intent]} onClose={onClose}>
      {held ? (
        <Image
          alt=""
          className="size-11 shrink-0 rounded-control bg-deep object-cover"
          height={held.height}
          src={held.thumbUrl}
          unoptimized
          width={held.width}
        />
      ) : null}
      <Choice
        label={held ? "Choose another file" : "Choose a file"}
        press={() => chooser.current?.click()}
        ready={!busy}
        word={held ? "Change" : "Choose a file"}
      >
        <ImageUp aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <input
        accept="image/png,image/jpeg,image/webp,image/gif"
        aria-label={`${INTENT_LABELS[intent]} file`}
        className={FILE}
        id={`${field}-file`}
        onChange={(event) => void choose(event.target.files?.[0])}
        ref={chooser}
        type="file"
      />
      <Pair field={`${field}-alt`} label="Description">
        <RowInput
          aria-describedby={`${field}-note`}
          aria-invalid={held !== null && !alt.trim() ? true : undefined}
          id={`${field}-alt`}
          maxLength={POST_PICTURE_TEXT_LIMIT}
          onChange={(event) => setAlt(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              place();
            }
          }}
          placeholder="What the picture shows"
          ref={description}
          value={alt}
        />
      </Pair>
      <Pair field={`${field}-caption`} label="Caption">
        <RowInput
          id={`${field}-caption`}
          maxLength={POST_PICTURE_TEXT_LIMIT}
          onChange={(event) => setCaption(event.target.value)}
          placeholder="Optional"
          value={caption}
        />
      </Pair>
      <Choice
        label={replacing ? "Replace" : "Insert image"}
        press={place}
        ready={held !== null && alt.trim().length > 0}
        strong
      />
      <RowNote id={`${field}-note`} refused={Boolean(refusal)}>
        {busy
          ? "Uploading…"
          : refusal ||
            (held === null
              ? "Choose a picture to upload."
              : alt.trim()
                ? ""
                : "Every picture needs a description before it can be placed.")}
      </RowNote>
    </Row>
  );
}

function placement(intent: PictureIntent, picture: PictureAttributes) {
  if (intent === "picture") return { type: "image", attrs: picture };
  if (intent === "gallery") {
    return {
      type: "gallery",
      content: [{ type: "galleryImage", attrs: picture }],
    };
  }
  return { type: "galleryImage", attrs: picture };
}

export function PictureFieldsRow({
  controls,
  editor,
  onOpen,
}: {
  controls: Controls;
  editor: Editor;
  onOpen: (intent: PictureIntent) => void;
}) {
  return (
    <Row label="Picture">
      <PictureText controls={controls} editor={editor} node="image" />
      <Choice label="Replace" press={() => onOpen("replace")} />
      <Choice
        label="Remove"
        press={() => editor.chain().focus().deleteSelection().run()}
      >
        <Trash2 aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
    </Row>
  );
}

export function GalleryPictureRow({
  controls,
  editor,
  onOpen,
}: {
  controls: Controls;
  editor: Editor;
  onOpen: (intent: PictureIntent) => void;
}) {
  return (
    <Row label="Gallery picture">
      <PictureText controls={controls} editor={editor} node="galleryImage" />
      <Choice
        label="Move image earlier"
        press={() => editor.chain().focus().movePicture(-1).run()}
        ready={controls.canMoveBack}
        word="Earlier"
      >
        <ArrowLeft aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Move image later"
        press={() => editor.chain().focus().movePicture(1).run()}
        ready={controls.canMoveOn}
        word="Later"
      >
        <ArrowRight aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice label="Add a picture" press={() => onOpen("add")} word="Add">
        <ImagePlus aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice label="Replace" press={() => onOpen("replace")} />
      <Choice
        label="Remove"
        press={() => editor.chain().focus().deleteSelection().run()}
      >
        <Trash2 aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
    </Row>
  );
}

function PictureText({
  controls,
  editor,
  node,
}: {
  controls: Controls;
  editor: Editor;
  node: "image" | "galleryImage";
}) {
  const field = useId();
  const missing = controls.picture.alt.trim().length === 0;
  return (
    <>
      <Pair field={`${field}-alt`} label="Description">
        <RowInput
          aria-invalid={missing ? true : undefined}
          id={`${field}-alt`}
          maxLength={POST_PICTURE_TEXT_LIMIT}
          onChange={(event) =>
            editor
              .chain()
              .updateAttributes(node, { alt: event.target.value })
              .run()
          }
          placeholder="What the picture shows"
          value={controls.picture.alt}
        />
      </Pair>
      <Pair field={`${field}-caption`} label="Caption">
        <RowInput
          id={`${field}-caption`}
          maxLength={POST_PICTURE_TEXT_LIMIT}
          onChange={(event) =>
            editor
              .chain()
              .updateAttributes(node, { caption: event.target.value })
              .run()
          }
          placeholder="Optional"
          value={controls.picture.caption}
        />
      </Pair>
      {missing ? (
        <RowNote refused>
          This picture needs a description before the post will save.
        </RowNote>
      ) : null}
    </>
  );
}
