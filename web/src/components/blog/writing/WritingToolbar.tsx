"use client";

import type { Editor } from "@tiptap/react";
import {
  Bold,
  Code,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  ListTodo,
  Pilcrow,
  Plus,
  Redo2,
  Strikethrough,
  Undo2,
} from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { cn } from "@/lib/cn";
import { BlockRow, InsertRow, LinkRow } from "./EditorRows";
import {
  GalleryPictureRow,
  PictureFieldsRow,
  type PictureIntent,
  PictureRow,
  type Upload,
} from "./PictureRows";
import { type Controls, useControls } from "./use-controls";
import { useToolbarKeys } from "./use-toolbar-keys";

type Opened = "none" | "link" | "insert" | PictureIntent;

const CONTROL =
  "grid size-11 shrink-0 place-items-center rounded-control text-mute outline-offset-3 transition-colors duration-200 hover:bg-deep hover:text-ink disabled:pointer-events-none disabled:opacity-40 motion-reduce:transition-none";

const ACTIVE = "bg-action text-on-accent hover:bg-action hover:text-on-accent";

export function WritingToolbar({
  editor,
  onUpload,
}: {
  editor: Editor | null;
  onUpload: Upload;
}) {
  const [row, setRow] = useState<Opened>("none");
  const controls = useControls(editor);
  const keys = useToolbarKeys();

  useEffect(() => {
    if (!editor) return;
    const close = () => setRow("none");
    editor.on("selectionUpdate", close);
    return () => {
      editor.off("selectionUpdate", close);
    };
  }, [editor]);

  if (!editor || !controls) {
    return (
      <div aria-busy="true" className="h-15 border-b border-rule/60 bg-field" />
    );
  }

  function open(which: Exclude<Opened, "none">) {
    setRow((current) => (current === which ? "none" : which));
  }

  return (
    <div className="sticky top-[var(--site-header-offset)] z-2 -mx-2 border-b border-rule/60 bg-field">
      <div
        aria-label="Post formatting"
        className="flex flex-wrap items-center gap-x-3 gap-y-1 py-1.5"
        onFocus={keys.onFocus}
        onKeyDown={keys.onKeyDown}
        ref={keys.bar}
        role="toolbar"
      >
        <Group label="Text style">
          <Control
            active={controls.paragraph}
            controls={controls}
            label="Paragraph"
            press={() => editor.chain().focus().setParagraph().run()}
          >
            <Pilcrow aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
          {[2, 3, 4].map((level) => (
            <Control
              active={controls.heading === level}
              controls={controls}
              key={level}
              label={`Heading ${level}`}
              press={() =>
                editor
                  .chain()
                  .focus()
                  .toggleHeading({ level: level as 2 | 3 | 4 })
                  .run()
              }
            >
              <span className="font-display text-ui leading-none font-semibold">
                H{level}
              </span>
            </Control>
          ))}
        </Group>
        <Group label="Emphasis">
          <Control
            active={controls.bold}
            controls={controls}
            label="Bold"
            press={() => editor.chain().focus().toggleBold().run()}
            shortcut="Ctrl B"
          >
            <Bold aria-hidden="true" className="size-4" strokeWidth={1.9} />
          </Control>
          <Control
            active={controls.italic}
            controls={controls}
            label="Italic"
            press={() => editor.chain().focus().toggleItalic().run()}
            shortcut="Ctrl I"
          >
            <Italic aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
          <Control
            active={controls.strike}
            controls={controls}
            label="Strike-through"
            press={() => editor.chain().focus().toggleStrike().run()}
            shortcut="Ctrl Shift S"
          >
            <Strikethrough
              aria-hidden="true"
              className="size-4"
              strokeWidth={1.7}
            />
          </Control>
          <Control
            active={controls.code}
            controls={controls}
            label="Inline code"
            press={() => editor.chain().focus().toggleCode().run()}
            shortcut="Ctrl E"
          >
            <Code aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
          <Control
            active={controls.link}
            controls={controls}
            expanded={row === "link"}
            label="Link"
            press={() => open("link")}
            shortcut="Ctrl K"
          >
            <LinkIcon aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
        </Group>
        <Group label="Lists">
          <Control
            active={controls.bulletList}
            controls={controls}
            label="Bulleted list"
            press={() => editor.chain().focus().toggleBulletList().run()}
          >
            <List aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
          <Control
            active={controls.orderedList}
            controls={controls}
            label="Numbered list"
            press={() => editor.chain().focus().toggleOrderedList().run()}
          >
            <ListOrdered
              aria-hidden="true"
              className="size-4"
              strokeWidth={1.7}
            />
          </Control>
          <Control
            active={controls.taskList}
            controls={controls}
            label="Task list"
            press={() => editor.chain().focus().toggleTaskList().run()}
          >
            <ListTodo aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
        </Group>
        <Group label="Insert content">
          <Control
            active={false}
            controls={controls}
            expanded={row === "insert"}
            label="Insert content"
            press={() => open("insert")}
          >
            <Plus
              aria-hidden="true"
              className="size-[18px]"
              strokeWidth={1.9}
            />
          </Control>
        </Group>
        <Group label="History">
          <Control
            active={false}
            controls={controls}
            label="Undo"
            press={() => editor.chain().focus().undo().run()}
            ready={controls.canUndo}
            shortcut="Ctrl Z"
          >
            <Undo2 aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
          <Control
            active={false}
            controls={controls}
            label="Redo"
            press={() => editor.chain().focus().redo().run()}
            ready={controls.canRedo}
            shortcut="Ctrl Shift Z"
          >
            <Redo2 aria-hidden="true" className="size-4" strokeWidth={1.7} />
          </Control>
        </Group>
      </div>
      <OpenRow
        close={() => setRow("none")}
        controls={controls}
        editor={editor}
        onOpen={open}
        onUpload={onUpload}
        row={row}
      />
    </div>
  );
}

function OpenRow({
  close,
  controls,
  editor,
  onOpen,
  onUpload,
  row,
}: {
  close: () => void;
  controls: Controls;
  editor: Editor;
  onOpen: (intent: PictureIntent) => void;
  onUpload: Upload;
  row: Opened;
}) {
  if (row === "link") return <LinkRow editor={editor} onClose={close} />;
  if (row === "insert") {
    return (
      <InsertRow
        controls={controls}
        editor={editor}
        onClose={close}
        onOpen={onOpen}
      />
    );
  }
  if (row !== "none") {
    return (
      <PictureRow
        editor={editor}
        intent={row}
        onClose={close}
        onUpload={onUpload}
        said={controls.picture}
      />
    );
  }
  if (controls.block === "picture") {
    return (
      <PictureFieldsRow controls={controls} editor={editor} onOpen={onOpen} />
    );
  }
  if (controls.block === "galleryPicture") {
    return (
      <GalleryPictureRow controls={controls} editor={editor} onOpen={onOpen} />
    );
  }
  return <BlockRow controls={controls} editor={editor} />;
}

function Group({ label, children }: { label: string; children: ReactNode }) {
  return (
    <fieldset className="flex items-center gap-0.5 border-0">
      <legend className="sr-only">{label}</legend>
      {children}
    </fieldset>
  );
}

function Control({
  active,
  children,
  controls,
  expanded,
  label,
  press,
  ready,
  shortcut,
}: {
  active: boolean;
  children: ReactNode;
  controls: Controls;
  expanded?: boolean;
  label: string;
  press: () => void;
  ready?: boolean;
  shortcut?: string;
}) {
  return (
    <button
      aria-expanded={expanded}
      aria-label={label}
      aria-pressed={expanded === undefined ? active : undefined}
      className={cn(CONTROL, (active || expanded) && ACTIVE)}
      disabled={!controls.editable || ready === false}
      onClick={press}
      title={shortcut ? `${label} (${shortcut})` : label}
      type="button"
    >
      {children}
    </button>
  );
}
