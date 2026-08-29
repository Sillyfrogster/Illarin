"use client";

import { type Editor, useEditorState } from "@tiptap/react";
import {
  isPostCalloutKind,
  isPostLanguage,
  type PostCalloutKind,
  type PostLanguage,
} from "@/lib/post-document";

export type Controls = {
  editable: boolean;
  paragraph: boolean;
  heading: number | null;
  bold: boolean;
  italic: boolean;
  strike: boolean;
  code: boolean;
  link: boolean;
  bulletList: boolean;
  orderedList: boolean;
  taskList: boolean;
  canUndo: boolean;
  canRedo: boolean;
  block: "codeBlock" | "table" | "callout" | null;
  language: PostLanguage;
  kind: PostCalloutKind;
  canQuote: boolean;
  canCodeBlock: boolean;
  canTable: boolean;
  canCallout: boolean;
  canDivider: boolean;
  canAddRow: boolean;
  canDeleteRow: boolean;
  canAddColumn: boolean;
  canDeleteColumn: boolean;
  canHeadingRow: boolean;
  canHeadingColumn: boolean;
  canDeleteTable: boolean;
};

const NEW_TABLE = { rows: 3, cols: 3, withHeaderRow: true };

/** What the bar and the row read. The live editor answers until Tiptap's snapshot has the editor. */
export function useControls(editor: Editor | null): Controls | null {
  const watched = useEditorState({
    editor,
    selector: ({ editor: live }) => (live ? read(live) : null),
  });
  if (watched) return watched;
  return editor ? read(editor) : null;
}

function read(editor: Editor): Controls {
  const table = editor.isActive("table");
  const language = editor.getAttributes("codeBlock").language;
  const kind = editor.getAttributes("callout").kind;
  return {
    editable: editor.isEditable,
    paragraph: editor.isActive("paragraph"),
    heading:
      [2, 3, 4].find((level) => editor.isActive("heading", { level })) ?? null,
    bold: editor.isActive("bold"),
    italic: editor.isActive("italic"),
    strike: editor.isActive("strike"),
    code: editor.isActive("code"),
    link: editor.isActive("link"),
    bulletList: editor.isActive("bulletList"),
    orderedList: editor.isActive("orderedList"),
    taskList: editor.isActive("taskList"),
    canUndo: editor.can().undo(),
    canRedo: editor.can().redo(),
    block: editor.isActive("codeBlock")
      ? "codeBlock"
      : table
        ? "table"
        : editor.isActive("callout")
          ? "callout"
          : null,
    language:
      typeof language === "string" && isPostLanguage(language)
        ? language
        : "plain",
    kind: typeof kind === "string" && isPostCalloutKind(kind) ? kind : "note",
    canQuote: editor.can().toggleBlockquote(),
    canCodeBlock: editor.can().toggleCodeBlock(),
    canTable: editor.can().insertTable(NEW_TABLE),
    canCallout: editor.can().toggleCallout("note"),
    canDivider: editor.can().setHorizontalRule(),
    canAddRow: table && editor.can().addRowAfter(),
    canDeleteRow: table && editor.can().deleteRow(),
    canAddColumn: table && editor.can().addColumnAfter(),
    canDeleteColumn: table && editor.can().deleteColumn(),
    canHeadingRow: table && editor.can().toggleHeaderRow(),
    canHeadingColumn: table && editor.can().toggleHeaderColumn(),
    canDeleteTable: table && editor.can().deleteTable(),
  };
}
