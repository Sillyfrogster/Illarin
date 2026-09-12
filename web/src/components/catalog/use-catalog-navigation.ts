"use client";

import { useRouter } from "next/navigation";
import { useCallback, useTransition } from "react";
import type { BrowseFilters } from "@/lib/api/query";
import { buildBrowseHref } from "@/lib/browse-url";

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
