"use client";

import type { Editor } from "@tiptap/react";
import {
  Columns3,
  Minus,
  Plus,
  Quote,
  Rows3,
  SquareCode,
  Table2,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { type ReactNode, useEffect, useId, useRef, useState } from "react";
import { LANGUAGE_LABELS } from "@/lib/code-highlight";
import {
  isPostCalloutKind,
  POST_CALLOUT_KINDS,
  POST_LANGUAGES,
  type PostCalloutKind,
  type PostLanguage,
} from "@/lib/post-document";
import styles from "./Toolbar.module.css";
import type { Controls } from "./use-controls";

const CALLOUT_LABELS: Record<string, string> = {
  note: "Note",
  tip: "Tip",
  important: "Important",
  warning: "Warning",
};

export function InsertRow({
  controls,
  editor,
  onClose,
}: {
  controls: Controls;
  editor: Editor;
  onClose: () => void;
}) {
  return (
    <Row label="Insert a structure" onClose={onClose}>
      <Choice
        label="Quotation"
        ready={controls.canQuote}
        press={() => {
          editor.chain().focus().toggleBlockquote().run();
          onClose();
        }}
      >
        <Quote size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Code block"
        ready={controls.canCodeBlock}
        press={() => {
          editor.chain().focus().toggleCodeBlock().run();
          onClose();
        }}
      >
        <SquareCode size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Table"
        ready={controls.canTable}
        press={() => {
          editor
            .chain()
            .focus()
            .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
            .run();
          onClose();
        }}
      >
        <Table2 size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Callout"
        ready={controls.canCallout}
        press={() => {
          editor.chain().focus().toggleCallout("note").run();
          onClose();
        }}
      >
        <TriangleAlert size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Divider"
        ready={controls.canDivider}
        press={() => {
          editor.chain().focus().setHorizontalRule().run();
          onClose();
        }}
      >
        <Minus size={15} strokeWidth={2} aria-hidden="true" />
      </Choice>
    </Row>
  );
}

export function BlockRow({
  controls,
  editor,
}: {
  controls: Controls;
  editor: Editor;
}) {
  if (controls.block === "codeBlock") {
    return <CodeRow chosen={controls.language} editor={editor} />;
  }
  if (controls.block === "table") {
    return <TableRow controls={controls} editor={editor} />;
  }
  if (controls.block === "callout") {
    return <CalloutRow chosen={controls.kind} editor={editor} />;
  }
  return null;
}

function CodeRow({ chosen, editor }: { chosen: PostLanguage; editor: Editor }) {
  const field = useId();
  return (
    <Row label="Code block">
      <label className={styles.pickerLabel} htmlFor={field}>
        Language
      </label>
      <select
        className={styles.picker}
        id={field}
        onChange={(event) =>
          editor
            .chain()
            .focus()
            .updateAttributes("codeBlock", { language: event.target.value })
            .run()
        }
        value={chosen}
      >
        {POST_LANGUAGES.map((name) => (
          <option key={name} value={name}>
            {LANGUAGE_LABELS[name]}
          </option>
        ))}
      </select>
      <Choice
        label="Remove code block"
        press={() => editor.chain().focus().toggleCodeBlock().run()}
      />
    </Row>
  );
}

function CalloutRow({
  chosen,
  editor,
}: {
  chosen: PostCalloutKind;
  editor: Editor;
}) {
  const field = useId();
  return (
    <Row label="Callout">
      <label className={styles.pickerLabel} htmlFor={field}>
        Kind
      </label>
      <select
        className={styles.picker}
        id={field}
        onChange={(event) => {
          const kind = event.target.value;
          if (isPostCalloutKind(kind)) {
            editor.chain().focus().setCalloutKind(kind).run();
          }
        }}
        value={chosen}
      >
        {POST_CALLOUT_KINDS.map((kind) => (
          <option key={kind} value={kind}>
            {CALLOUT_LABELS[kind]}
          </option>
        ))}
      </select>
      <Choice
        label="Remove callout"
        press={() => editor.chain().focus().lift("callout").run()}
      />
    </Row>
  );
}

function TableRow({
  controls,
  editor,
}: {
  controls: Controls;
  editor: Editor;
}) {
  return (
    <Row label="Table">
      <Choice
        label="Add a row below"
        press={() => editor.chain().focus().addRowAfter().run()}
        ready={controls.canAddRow}
        word="Row"
      >
        <Plus size={15} strokeWidth={2} aria-hidden="true" />
      </Choice>
      <Choice
        label="Delete this row"
        press={() => editor.chain().focus().deleteRow().run()}
        ready={controls.canDeleteRow}
        word="Row"
      >
        <Minus size={15} strokeWidth={2} aria-hidden="true" />
      </Choice>
      <Choice
        label="Add a column after"
        press={() => editor.chain().focus().addColumnAfter().run()}
        ready={controls.canAddColumn}
        word="Column"
      >
        <Plus size={15} strokeWidth={2} aria-hidden="true" />
      </Choice>
      <Choice
        label="Delete this column"
        press={() => editor.chain().focus().deleteColumn().run()}
        ready={controls.canDeleteColumn}
        word="Column"
      >
        <Minus size={15} strokeWidth={2} aria-hidden="true" />
      </Choice>
      <Choice
        label="Heading row"
        press={() => editor.chain().focus().toggleHeaderRow().run()}
        ready={controls.canHeadingRow}
      >
        <Rows3 size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Heading column"
        press={() => editor.chain().focus().toggleHeaderColumn().run()}
        ready={controls.canHeadingColumn}
      >
        <Columns3 size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
      <Choice
        label="Remove table"
        press={() => editor.chain().focus().deleteTable().run()}
        ready={controls.canDeleteTable}
        word="Remove"
      >
        <Trash2 size={15} strokeWidth={1.8} aria-hidden="true" />
      </Choice>
    </Row>
  );
}

export function LinkRow({
  editor,
  onClose,
}: {
  editor: Editor;
  onClose: () => void;
}) {
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
    <Row label="Link address" onClose={onClose}>
      <label className={styles.pickerLabel} htmlFor={field}>
        Address
      </label>
      <input
        className={styles.address}
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
      <Choice label="Apply" press={apply} strong />
      <Choice
        label="Remove"
        press={() => {
          editor.chain().focus().unsetLink().run();
          onClose();
        }}
      />
    </Row>
  );
}

function Row({
  children,
  label,
  onClose,
}: {
  children: ReactNode;
  label: string;
  onClose?: () => void;
}) {
  return (
    <fieldset
      aria-label={label}
      className={styles.row}
      onKeyDown={(event) => {
        if (event.key === "Escape" && onClose) onClose();
      }}
    >
      <span aria-hidden="true" className={styles.rowMark}>
        {label}
      </span>
      {children}
    </fieldset>
  );
}

function Choice({
  children,
  label,
  press,
  ready,
  strong,
  word,
}: {
  children?: ReactNode;
  label: string;
  press: () => void;
  ready?: boolean;
  strong?: boolean;
  word?: string;
}) {
  return (
    <button
      aria-label={label}
      className={strong ? `${styles.choice} ${styles.strong}` : styles.choice}
      disabled={ready === false}
      onClick={press}
      title={label}
      type="button"
    >
      {children}
      <span aria-hidden="true">{word ?? label}</span>
    </button>
  );
}
