"use client";

import { useRouter } from "next/navigation";
import { useCallback, useTransition } from "react";
import type { BrowseFilters } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";

/** Every narrowing is a page address, so the back button and a shared link both work. */
export function useCatalogNavigation(basePath: string) {
  const router = useRouter();
  const [pending, start] = useTransition();

  const navigate = useCallback(
    (next: BrowseFilters) => {
      start(() => {
        router.push(buildBrowseHref(next, basePath), { scroll: false });
      });
    },
    [basePath, router],
  );

  return { navigate, pending };
}
