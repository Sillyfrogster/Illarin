import { Extension } from "@tiptap/core";

/** The editor carries a heading's stored address and leaves a new one to Go. */
export const HeadingAnchor = Extension.create({
  name: "headingAnchor",

  addGlobalAttributes() {
    return [
      {
        types: ["heading"],
        attributes: {
          anchor: { default: null, rendered: false },
        },
      },
    ];
  },
});
