import type { WorkDetail } from "@/lib/api/query";

const NAMED_VERSION_LIMIT = 5;

const versionOrder = new Intl.Collator("en", { numeric: true });

/** installedVersionsLine says which app versions readers have an extension installed on, naming the newest few. */
export function installedVersionsLine(
  page: Pick<WorkDetail, "appFormats" | "installedAppVersions">,
): string | null {
  const app = page.appFormats[0]?.label;
  if (!app || page.installedAppVersions.length === 0) return null;
  const versions = [...page.installedAppVersions].sort((a, b) =>
    versionOrder.compare(b, a),
  );
  const named =
    versions.length > NAMED_VERSION_LIMIT
      ? [
          ...versions.slice(0, NAMED_VERSION_LIMIT - 1),
          `${versions.length - NAMED_VERSION_LIMIT + 1} other versions`,
        ]
      : versions;
  return `Installed by readers on ${app} ${joined(named)}.`;
}

function joined(items: string[]): string {
  if (items.length < 2) return items.join("");
  return `${items.slice(0, -1).join(", ")} and ${items.at(-1)}`;
}
