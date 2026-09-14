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

/** Describes the public Publication API pages. */
export const PUBLICATION_DOCS: DocCollection = {
  name: "Publication API",
  href: "/developers/publication",
  directory: "publication",
  pages: [
    {
      slug: "",
      title: "Overview",
      summary:
        "Get access, create a draft, save it and publish your first post.",
      file: "overview.md",
    },
    {
      slug: "writing",
      title: "Writing",
      summary:
        "Create and edit drafts, upload pictures, and restore saved revisions.",
      file: "writing.md",
    },
    {
      slug: "publishing",
      title: "Publishing",
      summary:
        "Publish or schedule a post, withdraw it, and check announcements.",
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
      summary: "Write a post body in Markdown and check import warnings.",
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

/** Describes the contract an app follows to install the extensions readers send it. */
export const APP_DOCS: DocCollection = {
  name: "Connect an app to Illarin",
  href: "/developers/apps",
  directory: "apps",
  pages: [
    {
      slug: "",
      title: "Overview",
      summary:
        "Declare that your app installs extensions and receive the ones readers send it.",
      file: "overview.md",
    },
    {
      slug: "installing",
      title: "Installing and updating",
      summary:
        "Install switched off, ask before first run, and keep grants through an update.",
      file: "installing.md",
    },
    {
      slug: "reporting",
      title: "Reporting back",
      summary:
        "Report where an extension opens and your app version, and show withheld notices.",
      file: "reporting.md",
    },
    {
      slug: "checklist",
      title: "Conformance checklist",
      summary: "Every rule an app that installs extensions must pass.",
      file: "checklist.md",
    },
  ],
};

/** Lists documentation collections in sidebar order. */
export const DOCS: readonly DocCollection[] = [PUBLICATION_DOCS, APP_DOCS];

export function docHref(collection: DocCollection, page: DocPage): string {
  return page.slug ? `${collection.href}/${page.slug}` : collection.href;
}

export function findDocCollection(directory: string): DocCollection | null {
  return DOCS.find((collection) => collection.directory === directory) ?? null;
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
