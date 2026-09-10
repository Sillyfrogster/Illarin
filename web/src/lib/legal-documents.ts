export const LEGAL_DOCUMENTS = [
  { href: "/legal/terms", title: "Terms of Service" },
  { href: "/legal/privacy", title: "Privacy Policy" },
  { href: "/legal/acceptable-use", title: "Acceptable Use" },
  { href: "/legal/dmca", title: "DMCA / Copyright" },
] as const;

export type LegalHref = (typeof LEGAL_DOCUMENTS)[number]["href"];

export const LEGAL_EFFECTIVE_DATE = "23 August 2026";

export type LegalDocument = (typeof LEGAL_DOCUMENTS)[number];

export function clauseAnchor(heading: string): string {
  return heading
    .replace(/^[\d.]+\s+/, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export function nextDocument(href: LegalHref): LegalDocument | null {
  const at = LEGAL_DOCUMENTS.findIndex((one) => one.href === href);
  if (at < 0) return null;
  return LEGAL_DOCUMENTS[at + 1] ?? null;
}
