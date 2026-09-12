export type DocPage = {
  slug: string;
  title: string;
  summary: string;
  file: string;
};

export type DocCollection = {
  name: string;
  href: string;
  directory: string;
  pages: readonly DocPage[];
};

/** The one public reference: the versioned Publication API and nothing else. */
export const PUBLICATION_DOCS: DocCollection = {
  name: "Publication API",
  href: "/developers/publication",
  directory: "publication",
  pages: [
    {
      slug: "",
      title: "Overview",
      summary: "What the API is for, how to authenticate and how a post moves.",
      file: "overview.md",
    },
    {
      slug: "writing",
      title: "Writing",
      summary:
        "List, read, create and save posts, upload pictures and keep editions.",
      file: "writing.md",
    },
    {
      slug: "publishing",
      title: "Publishing",
      summary:
        "Publish now, schedule, withdraw, delete and read what was sent.",
      file: "publishing.md",
    },
    {
      slug: "document",
      title: "Post document",
      summary:
        "Every block, mark and rule of the structured body Illarin stores.",
      file: "document.md",
    },
    {
      slug: "markdown",
      title: "Markdown import",
      summary: "The Markdown subset an import reads once, and what it refuses.",
      file: "markdown.md",
    },
    {
      slug: "requests",
      title: "Requests and errors",
      summary:
        "Idempotency keys, version conflicts, stable error codes and limits.",
      file: "requests.md",
    },
    {
      slug: "webhooks",
      title: "Webhooks",
      summary:
        "The signed events a destination receives and how to verify them.",
      file: "webhooks.md",
    },
  ],
};

/** Every collection the developer portal carries, in sidebar order. */
export const DOCS: readonly DocCollection[] = [PUBLICATION_DOCS];

export function docHref(collection: DocCollection, page: DocPage): string {
  return page.slug ? `${collection.href}/${page.slug}` : collection.href;
}

export function findDocPage(
  collection: DocCollection,
  slug: string,
): DocPage | null {
  return collection.pages.find((page) => page.slug === slug) ?? null;
}

export function nextDocPage(
  collection: DocCollection,
  page: DocPage,
): DocPage | null {
  const at = collection.pages.indexOf(page);
  return collection.pages[at + 1] ?? null;
}

export function previousDocPage(
  collection: DocCollection,
  page: DocPage,
): DocPage | null {
  const at = collection.pages.indexOf(page);
  return at > 0 ? collection.pages[at - 1] : null;
}
