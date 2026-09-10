"use client";

import type { Editor } from "@tiptap/react";
import {
  Columns3,
  Image,
  Images,
  Minus,
  Plus,
  Quote,
  Rows3,
  SquareCode,
  Table2,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { LANGUAGE_LABELS } from "@/lib/code-highlight";
import {
  isPostCalloutKind,
  POST_CALLOUT_KINDS,
  POST_LANGUAGES,
  type PostCalloutKind,
  type PostLanguage,
} from "@/lib/post-document";
import { isSafeAddress } from "@/lib/post-link";
import { Choice, Row, RowInput, RowNote, RowSelect } from "./RowParts";
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
  onOpen,
}: {
  controls: Controls;
  editor: Editor;
  onClose: () => void;
  onOpen: (intent: "picture" | "gallery") => void;
}) {
  return (
    <Row label="Insert" onClose={onClose}>
      <Choice
        label="Quotation"
        press={() => {
          editor.chain().focus().toggleBlockquote().run();
          onClose();
        }}
        ready={controls.canQuote}
      >
        <Quote aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Code block"
        press={() => {
          editor.chain().focus().toggleCodeBlock().run();
          onClose();
        }}
        ready={controls.canCodeBlock}
      >
        <SquareCode aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Table"
        press={() => {
          editor
            .chain()
            .focus()
            .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
            .run();
          onClose();
        }}
        ready={controls.canTable}
      >
        <Table2 aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Callout"
        press={() => {
          editor.chain().focus().toggleCallout("note").run();
          onClose();
        }}
        ready={controls.canCallout}
      >
        <TriangleAlert
          aria-hidden="true"
          className="size-4"
          strokeWidth={1.8}
        />
      </Choice>
      <Choice
        label="Picture"
        press={() => onOpen("picture")}
        ready={controls.canPicture}
      >
        <Image aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Gallery"
        press={() => onOpen("gallery")}
        ready={controls.canPicture}
      >
        <Images aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Divider"
        press={() => {
          editor.chain().focus().setHorizontalRule().run();
          onClose();
        }}
        ready={controls.canDivider}
      >
        <Minus aria-hidden="true" className="size-4" strokeWidth={2} />
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
      <label className="shrink-0 font-ui text-meta text-mute" htmlFor={field}>
        Language
      </label>
      <RowSelect
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
      </RowSelect>
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
      <label className="shrink-0 font-ui text-meta text-mute" htmlFor={field}>
        Kind
      </label>
      <RowSelect
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
      </RowSelect>
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
        <Plus aria-hidden="true" className="size-4" strokeWidth={2} />
      </Choice>
      <Choice
        label="Delete this row"
        press={() => editor.chain().focus().deleteRow().run()}
        ready={controls.canDeleteRow}
        word="Row"
      >
        <Minus aria-hidden="true" className="size-4" strokeWidth={2} />
      </Choice>
      <Choice
        label="Add a column after"
        press={() => editor.chain().focus().addColumnAfter().run()}
        ready={controls.canAddColumn}
        word="Column"
      >
        <Plus aria-hidden="true" className="size-4" strokeWidth={2} />
      </Choice>
      <Choice
        label="Delete this column"
        press={() => editor.chain().focus().deleteColumn().run()}
        ready={controls.canDeleteColumn}
        word="Column"
      >
        <Minus aria-hidden="true" className="size-4" strokeWidth={2} />
      </Choice>
      <Choice
        label="Heading row"
        press={() => editor.chain().focus().toggleHeaderRow().run()}
        ready={controls.canHeadingRow}
      >
        <Rows3 aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Heading column"
        press={() => editor.chain().focus().toggleHeaderColumn().run()}
        ready={controls.canHeadingColumn}
      >
        <Columns3 aria-hidden="true" className="size-4" strokeWidth={1.8} />
      </Choice>
      <Choice
        label="Remove table"
        press={() => editor.chain().focus().deleteTable().run()}
        ready={controls.canDeleteTable}
        word="Remove"
      >
        <Trash2 aria-hidden="true" className="size-4" strokeWidth={1.8} />
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
  const [refusal, setRefusal] = useState("");

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
    if (!isSafeAddress(trimmed)) {
      setRefusal("A link goes to an https address or a mailto address.");
      input.current?.focus();
      return;
    }
    editor.chain().focus().setLink({ href: trimmed }).run();
    onClose();
  }

  return (
    <Row label="Link" onClose={onClose}>
      <label className="shrink-0 font-ui text-meta text-mute" htmlFor={field}>
        Address
      </label>
      <span className="min-w-0 flex-[1_1_15rem]">
        <RowInput
          aria-describedby={refusal ? `${field}-refusal` : undefined}
          aria-invalid={refusal ? true : undefined}
          id={field}
          onChange={(event) => {
            setAddress(event.target.value);
            setRefusal("");
          }}
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
      </span>
      <Choice label="Apply" press={apply} strong />
      <Choice
        label="Remove"
        press={() => {
          editor.chain().focus().unsetLink().run();
          onClose();
        }}
      />
      {refusal ? (
        <RowNote id={`${field}-refusal`} refused>
          {refusal}
        </RowNote>
      ) : null}
    </Row>
  );
}
