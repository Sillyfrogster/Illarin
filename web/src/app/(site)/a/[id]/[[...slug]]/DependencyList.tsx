"use client";

import Link from "next/link";
import { Run, RunItem } from "@/components/ui/run";
import { assetHref } from "@/lib/asset-url";
import {
  dependencyLinks,
  useExtensionDependencies,
} from "@/lib/extension-dependencies";
import { ITEM_META } from "./element-runs";

export function DependencyList({
  texts,
  itemLimit,
}: {
  texts: { text: string }[];
  itemLimit?: number;
}) {
  const links = dependencyLinks(
    texts.slice(0, itemLimit),
    useExtensionDependencies(),
  );
  return (
    <Run>
      {links.map((link, index) => (
        <RunItem itemKey={`${index}`} key={`${index}-${link.name}`}>
          <p className="font-mono text-meta text-ink [overflow-wrap:anywhere]">
            {link.name}
          </p>
          {link.works.map((found) => (
            <p className={ITEM_META} key={found.id}>
              <Link
                className="font-ui text-ui font-medium text-ink underline decoration-accent/55 underline-offset-[3px] [overflow-wrap:anywhere] hover:decoration-accent"
                href={assetHref(found.id, found.name)}
              >
                {found.name}
              </Link>{" "}
              by {found.creator}
            </p>
          ))}
        </RunItem>
      ))}
    </Run>
  );
}
