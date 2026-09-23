import { readFile } from "node:fs/promises";
import path from "node:path";
import { cache } from "react";

export const DOCS = [
  {
    slug: "app-integration",
    title: "App integration",
    description:
      "Connect an installation, receive works, and report its library.",
  },
  {
    slug: "extension-publishing",
    title: "Extension publishing",
    description: "Verify a GitHub repository and publish extension releases.",
  },
] as const;

export type DocSlug = (typeof DOCS)[number]["slug"];

export function findDoc(slug: string) {
  return DOCS.find((doc) => doc.slug === slug);
}

export const readDoc = cache((slug: DocSlug) =>
  readFile(path.join(process.cwd(), "src/content/docs", `${slug}.md`), "utf8"),
);

export function headingId(text: string) {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .trim()
    .replace(/\s+/g, "-");
}
