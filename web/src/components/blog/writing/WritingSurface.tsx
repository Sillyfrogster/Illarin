"use client";

import { TaskItem, TaskList } from "@tiptap/extension-list";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import type { PostMedia } from "@/lib/api/query";
import type { PostBody } from "@/lib/post-body";
import { Callout } from "./callout-node";
import { HeadingAnchor } from "./heading-anchor";
import { Gallery, GalleryPicture, Picture } from "./picture-nodes";
import { fromEditor, toEditor } from "./tiptap-document";
import { WritingToolbar } from "./WritingToolbar";
import { WRITING_SURFACE } from "./writing-surface";

export function WritingSurface({
  body,
  media,
  onUpload,
  onChange,
}: {
  body: PostBody;
  media: PostMedia[];
  onUpload: (file: File) => Promise<PostMedia | null>;
  onChange: (body: PostBody) => void;
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
      Picture,
      Gallery,
      GalleryPicture,
      Placeholder.configure({
        placeholder: "Write the post. Formatting is on the bar above.",
      }),
    ],
    content: toEditor(body, media),
    editorProps: {
      attributes: {
        class: WRITING_SURFACE,
        "aria-label": "Post body",
        role: "textbox",
        "aria-multiline": "true",
      },
    },
    onUpdate: ({ editor: written }) => onChange(fromEditor(written.getJSON())),
  });

  return (
    <div className="mt-group">
      <WritingToolbar editor={editor} onUpload={onUpload} />
      <EditorContent className="pt-8 pb-14" editor={editor} />
    </div>
  );
}
