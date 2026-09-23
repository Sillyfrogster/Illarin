import { Extension } from "@tiptap/core";

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
