"use client";

import { Placeholder } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import type { PostDocument } from "@/lib/post-document";
import styles from "./BodyEditor.module.css";
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
        codeBlock: false,
        strike: false,
        underline: false,
        hardBreak: false,
        heading: { levels: [2, 3, 4] },
        link: {
          openOnClick: false,
          autolink: false,
          protocols: ["https", "mailto"],
        },
      }),
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
