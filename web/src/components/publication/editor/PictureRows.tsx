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
import { POST_PICTURE_TEXT_LIMIT } from "@/lib/post-document";
import type { PictureAttributes } from "./picture-nodes";
import { Choice, Pair, Row } from "./RowParts";
import styles from "./Toolbar.module.css";
import type { Controls } from "./use-controls";

export type Upload = (file: File) => Promise<PostMedia | null>;

/** What choosing a file is about to do to the post. */
export type PictureIntent = "picture" | "gallery" | "add" | "replace";

const INTENT_LABELS: Record<PictureIntent, string> = {
  picture: "Picture",
  gallery: "Gallery",
  add: "Another picture",
  replace: "Replace the picture",
};

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
          className={styles.thumb}
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
        <ImageUp size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <input
        accept="image/png,image/jpeg,image/webp,image/gif"
        aria-label={`${INTENT_LABELS[intent]} file`}
        className={styles.file}
        id={`${field}-file`}
        onChange={(event) => void choose(event.target.files?.[0])}
        ref={chooser}
        type="file"
      />
      <Pair field={`${field}-alt`} label="Description">
        <input
          aria-describedby={`${field}-note`}
          aria-invalid={held !== null && !alt.trim() ? true : undefined}
          className={styles.address}
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
        <input
          className={styles.address}
          id={`${field}-caption`}
          maxLength={POST_PICTURE_TEXT_LIMIT}
          onChange={(event) => setCaption(event.target.value)}
          placeholder="Optional"
          value={caption}
        />
      </Pair>
      <Choice
        label={replacing ? "Replace" : "Place"}
        press={place}
        ready={held !== null && alt.trim().length > 0}
        strong
      />
      <output className={styles.note} id={`${field}-note`}>
        {busy
          ? "Uploading…"
          : refusal ||
            (held === null
              ? "Choose a picture to upload."
              : alt.trim()
                ? ""
                : "Every picture needs a description before it can be placed.")}
      </output>
    </Row>
  );
}

// placement answers the node an intent puts after whatever the caret is on.
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
        <Trash2 size={15} strokeWidth={1.8} aria-hidden="true" />
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
        label="Move this picture back"
        press={() => editor.chain().focus().movePicture(-1).run()}
        ready={controls.canMoveBack}
        word="Back"
      >
        <ArrowLeft size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Move this picture on"
        press={() => editor.chain().focus().movePicture(1).run()}
        ready={controls.canMoveOn}
        word="On"
      >
        <ArrowRight size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice label="Add a picture" press={() => onOpen("add")} word="Add">
        <ImagePlus size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice label="Replace" press={() => onOpen("replace")} />
      <Choice
        label="Remove"
        press={() => editor.chain().focus().deleteSelection().run()}
      >
        <Trash2 size={15} strokeWidth={1.8} aria-hidden="true" />
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
        <input
          aria-invalid={missing ? true : undefined}
          className={styles.address}
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
        <input
          className={styles.address}
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
        <p className={styles.refusal} role="alert">
          This picture needs a description before the post will save.
        </p>
      ) : null}
    </>
  );
}
