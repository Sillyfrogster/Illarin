import { mergeAttributes, Node } from "@tiptap/core";
import { isPostCalloutKind, type PostCalloutKind } from "@/lib/post-document";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    callout: {
      toggleCallout: (kind: PostCalloutKind) => ReturnType;
      setCalloutKind: (kind: PostCalloutKind) => ReturnType;
    };
  }
}

/** The editor's callout is Illarin's node, not a Tiptap one with a class on it. */
export const Callout = Node.create({
  name: "callout",
  group: "block",
  content: "(paragraph|bulletList|orderedList|taskList|codeBlock)+",
  defining: true,

  addAttributes() {
    return {
      kind: {
        default: "note" as PostCalloutKind,
        parseHTML: (element) => {
          const kind = element.getAttribute("data-kind") ?? "";
          return isPostCalloutKind(kind) ? kind : "note";
        },
        renderHTML: (attributes) => ({ "data-kind": attributes.kind }),
      },
    };
  },

  parseHTML() {
    return [{ tag: "div[data-kind]" }, { tag: "aside[data-kind]" }];
  },

  renderHTML({ HTMLAttributes }) {
    return ["div", mergeAttributes(HTMLAttributes, { role: "note" }), 0];
  },

  addCommands() {
    return {
      toggleCallout:
        (kind) =>
        ({ commands }) =>
          commands.toggleWrap(this.name, { kind }),
      setCalloutKind:
        (kind) =>
        ({ commands }) =>
          commands.updateAttributes(this.name, { kind }),
    };
  },
});
