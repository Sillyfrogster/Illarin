import { mergeAttributes, Node } from "@tiptap/core";
import type { DOMOutputSpec, Node as ProseMirrorNode } from "@tiptap/pm/model";
import { NodeSelection } from "@tiptap/pm/state";
import {
  WRITING_GALLERY,
  WRITING_GALLERY_PICTURE,
  WRITING_PICTURE,
} from "./writing-surface";

export type PictureAttributes = {
  mediaId: string;
  alt: string;
  caption: string;
  src: string;
  width: number | null;
  height: number | null;
};

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    pictures: {
      movePicture: (places: -1 | 1) => ReturnType;
    };
  }
}

const pictureAttributes = {
  mediaId: { default: "" },
  alt: { default: "" },
  caption: { default: "" },
  src: { default: "" },
  width: { default: null },
  height: { default: null },
};

function figure(
  attributes: Record<string, unknown>,
  marker: string,
  className: string,
): DOMOutputSpec {
  const caption = String(attributes.caption ?? "");
  const frame = { class: className, [marker]: "" };
  const picture = [
    "img",
    {
      src: String(attributes.src ?? ""),
      alt: String(attributes.alt ?? ""),
      width: attributes.width ?? undefined,
      height: attributes.height ?? undefined,
    },
  ];
  return caption
    ? ["figure", frame, picture, ["figcaption", caption]]
    : ["figure", frame, picture];
}

export const Picture = Node.create({
  name: "image",
  group: "block",
  atom: true,
  draggable: true,
  selectable: true,

  addAttributes: () => pictureAttributes,

  parseHTML() {
    return [{ tag: "figure[data-picture]" }];
  },

  renderHTML({ node }) {
    return figure(node.attrs, "data-picture", WRITING_PICTURE);
  },
});

export const GalleryPicture = Node.create({
  name: "galleryImage",
  atom: true,
  selectable: true,

  addAttributes: () => pictureAttributes,

  parseHTML() {
    return [{ tag: "figure[data-gallery-picture]" }];
  },

  renderHTML({ node }) {
    return figure(node.attrs, "data-gallery-picture", WRITING_GALLERY_PICTURE);
  },

  addCommands() {
    return {
      movePicture:
        (places) =>
        ({ state, tr, dispatch }) => {
          const selected = state.selection;
          if (
            !(selected instanceof NodeSelection) ||
            selected.node.type.name !== this.name
          ) {
            return false;
          }
          const at = selected.$from.index();
          const target = at + places;
          const gallery = selected.$from.parent;
          if (target < 0 || target >= gallery.childCount) return false;
          if (!dispatch) return true;
          const pictures: ProseMirrorNode[] = [];
          gallery.forEach((picture) => {
            pictures.push(picture);
          });
          const [moved] = pictures.splice(at, 1);
          pictures.splice(target, 0, moved);
          const start = selected.$from.start();
          tr.replaceWith(start, selected.$from.end(), pictures);
          let landed = start;
          for (let before = 0; before < target; before += 1) {
            landed += pictures[before].nodeSize;
          }
          tr.setSelection(NodeSelection.create(tr.doc, landed));
          dispatch(tr);
          return true;
        },
    };
  },
});

export const Gallery = Node.create({
  name: "gallery",
  group: "block",
  content: "galleryImage+",
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: "div[data-gallery]" }];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "div",
      mergeAttributes(HTMLAttributes, {
        "data-gallery": "",
        class: WRITING_GALLERY,
      }),
      0,
    ];
  },
});
