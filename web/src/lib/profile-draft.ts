import type { ProfileLink } from "@/lib/api/query";

/** How many addresses a profile may carry. */
export const LINK_LIMIT = 6;

/** How long a biography may run. */
export const BIOGRAPHY_LIMIT = 400;

/** Appends the empty row the next address is typed into. */
export function addLink(links: ProfileLink[]): ProfileLink[] {
  if (links.length >= LINK_LIMIT) return links;
  return [...links, { address: "", label: "" }];
}

/** Replaces one field of one address, leaving every other row untouched. */
export function writeLink(
  links: ProfileLink[],
  index: number,
  field: keyof ProfileLink,
  value: string,
): ProfileLink[] {
  return links.map((link, position) =>
    position === index ? { ...link, [field]: value } : link,
  );
}

export function removeLink(links: ProfileLink[], index: number): ProfileLink[] {
  return links.filter((_, position) => position !== index);
}

/** Swaps an address with its neighbour; a move off either end changes nothing. */
export function moveLink(
  links: ProfileLink[],
  index: number,
  step: number,
): ProfileLink[] {
  const destination = index + step;
  if (destination < 0 || destination >= links.length) return links;
  const next = [...links];
  [next[index], next[destination]] = [next[destination], next[index]];
  return next;
}

type PublicFields = {
  avatar: boolean;
  biography: string;
  contactEmail: string;
  displayName: string;
  links: ProfileLink[];
};

/**
 * Names what a visitor would find on the profile, so the account page can say
 * what is showing without repeating the whole editor.
 */
export function whatIsPublic(fields: PublicFields): string[] {
  const shown = fields.links.filter((link) => link.label || link.address);
  return [
    fields.displayName && "display name",
    fields.avatar && "avatar",
    fields.biography && "biography",
    fields.contactEmail && "contact",
    shown.length > 0 &&
      `${shown.length} ${shown.length === 1 ? "link" : "links"}`,
  ].filter((one): one is string => typeof one === "string" && one !== "");
}
