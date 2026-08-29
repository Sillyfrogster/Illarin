"use client";

import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import type { PostDocument } from "@/lib/post-document";
import styles from "./BodyEditor.module.css";
import { Callout } from "./callout-node";
import { HeadingAnchor } from "./heading-anchor";
import { Toolbar } from "./Toolbar";
import { fromEditor, toEditor } from "./tiptap-document";

export function BodyEditor({
  document,
  onChange,
}: {
  document: PostDocument;
  onChange: (document: PostDocument) => void;
}) {
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({
        underline: false,
        hardBreak: false,
        heading: { levels: [2, 3, 4] },
        codeBlock: { defaultLanguage: "plain" },
        link: {
          openOnClick: false,
          autolink: false,
          protocols: ["https", "mailto"],
        },
      }),
      HeadingAnchor,
      TaskList,
      TaskItem.configure({ nested: true }),
      TableKit.configure({ table: { resizable: false } }),
      Callout,
      Placeholder.configure({
        placeholder: "Write the post. Formatting is on the bar above.",
      }),
    ],
    content: toEditor(document),
    editorProps: {
      attributes: {
        class: styles.surface,
        "aria-label": "Post body",
        role: "textbox",
        "aria-multiline": "true",
      },
    },
    onUpdate: ({ editor: written }) => onChange(fromEditor(written.getJSON())),
  });

  return (
    <div className={styles.editor}>
      <Toolbar editor={editor} />
      <EditorContent editor={editor} />
    </div>
  );
}
