"use client";

import type { Editor } from "@tiptap/react";
import {
  Bold,
  Code,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Minus,
  Pilcrow,
  Quote,
  Redo2,
  Undo2,
} from "lucide-react";
import { type ReactNode, useEffect, useId, useRef, useState } from "react";
import styles from "./Toolbar.module.css";

export function Toolbar({ editor }: { editor: Editor | null }) {
  const [linking, setLinking] = useState(false);
  if (!editor) return <div className={styles.toolbar} aria-busy="true" />;
  return (
    <div className={styles.bar}>
      <div
        className={styles.toolbar}
        role="toolbar"
        aria-label="Post formatting"
      >
        <Group label="Text style">
          <Control
            editor={editor}
            label="Paragraph"
            active={editor.isActive("paragraph")}
            press={() => editor.chain().focus().setParagraph().run()}
          >
            <Pilcrow size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          {[2, 3, 4].map((level) => (
            <Control
              editor={editor}
              key={level}
              label={`Heading ${level}`}
              active={editor.isActive("heading", { level })}
              press={() =>
                editor
                  .chain()
                  .focus()
                  .toggleHeading({ level: level as 2 | 3 | 4 })
                  .run()
              }
            >
              <span className={styles.heading}>H{level}</span>
            </Control>
          ))}
        </Group>
        <Group label="Emphasis">
          <Control
            editor={editor}
            label="Bold"
            shortcut="Ctrl B"
            active={editor.isActive("bold")}
            press={() => editor.chain().focus().toggleBold().run()}
          >
            <Bold size={16} strokeWidth={1.9} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Italic"
            shortcut="Ctrl I"
            active={editor.isActive("italic")}
            press={() => editor.chain().focus().toggleItalic().run()}
          >
            <Italic size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Inline code"
            shortcut="Ctrl E"
            active={editor.isActive("code")}
            press={() => editor.chain().focus().toggleCode().run()}
          >
            <Code size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Link"
            shortcut="Ctrl K"
            active={editor.isActive("link")}
            press={() => setLinking((open) => !open)}
            expanded={linking}
          >
            <LinkIcon size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
        </Group>
        <Group label="Blocks">
          <Control
            editor={editor}
            label="Bulleted list"
            active={editor.isActive("bulletList")}
            press={() => editor.chain().focus().toggleBulletList().run()}
          >
            <List size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Numbered list"
            active={editor.isActive("orderedList")}
            press={() => editor.chain().focus().toggleOrderedList().run()}
          >
            <ListOrdered size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Quotation"
            active={editor.isActive("blockquote")}
            press={() => editor.chain().focus().toggleBlockquote().run()}
          >
            <Quote size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Divider"
            active={false}
            press={() => editor.chain().focus().setHorizontalRule().run()}
          >
            <Minus size={16} strokeWidth={1.9} aria-hidden="true" />
          </Control>
        </Group>
        <Group label="History">
          <Control
            editor={editor}
            label="Undo"
            shortcut="Ctrl Z"
            active={false}
            press={() => editor.chain().focus().undo().run()}
          >
            <Undo2 size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
          <Control
            editor={editor}
            label="Redo"
            shortcut="Ctrl Shift Z"
            active={false}
            press={() => editor.chain().focus().redo().run()}
          >
            <Redo2 size={16} strokeWidth={1.7} aria-hidden="true" />
          </Control>
        </Group>
      </div>
      {linking ? (
        <LinkRow editor={editor} onClose={() => setLinking(false)} />
      ) : null}
    </div>
  );
}

function Group({ label, children }: { label: string; children: ReactNode }) {
  return (
    <fieldset className={styles.group}>
      <legend>{label}</legend>
      {children}
    </fieldset>
  );
}

function Control({
  editor,
  label,
  shortcut,
  active,
  expanded,
  press,
  children,
}: {
  editor: Editor;
  label: string;
  shortcut?: string;
  active: boolean;
  expanded?: boolean;
  press: () => void;
  children: ReactNode;
}) {
  return (
    <button
      aria-expanded={expanded}
      aria-label={label}
      aria-pressed={expanded === undefined ? active : undefined}
      className={styles.control}
      data-active={active || undefined}
      disabled={!editor.isEditable}
      onClick={press}
      title={shortcut ? `${label} (${shortcut})` : label}
      type="button"
    >
      {children}
    </button>
  );
}

function LinkRow({ editor, onClose }: { editor: Editor; onClose: () => void }) {
  const field = useId();
  const input = useRef<HTMLInputElement>(null);
  const [address, setAddress] = useState<string>(
    () => (editor.getAttributes("link").href as string) ?? "",
  );

  useEffect(() => {
    input.current?.focus();
  }, []);

  function apply() {
    const trimmed = address.trim();
    if (!trimmed) {
      editor.chain().focus().unsetLink().run();
      onClose();
      return;
    }
    editor.chain().focus().setLink({ href: trimmed }).run();
    onClose();
  }

  return (
    <fieldset className={styles.link}>
      <legend>Link address</legend>
      <label htmlFor={field}>Address</label>
      <input
        id={field}
        onChange={(event) => setAddress(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            apply();
          }
          if (event.key === "Escape") onClose();
        }}
        placeholder="https://"
        ref={input}
        type="url"
        value={address}
      />
      <button onClick={apply} type="button">
        Apply
      </button>
      <button
        className={styles.clear}
        onClick={() => {
          editor.chain().focus().unsetLink().run();
          onClose();
        }}
        type="button"
      >
        Remove
      </button>
    </fieldset>
  );
}
