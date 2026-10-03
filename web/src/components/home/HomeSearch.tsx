"use client";

import { useRouter } from "next/navigation";
import { BrowseSearch } from "@/components/browse/BrowseSearch";
import { buildBrowseHref } from "@/lib/browse-url";

/** HomeSearch runs Browse's search, tag suggestions included, and opens Browse on the results. */
export function HomeSearch() {
  const router = useRouter();
  return (
    <BrowseSearch
      id="home-search"
      label="Search works"
      onSearch={(q) => router.push(buildBrowseHref({ q }))}
      placeholder="Search characters, lorebooks and more"
      tags
      value=""
    />
  );
}
