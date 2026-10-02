"use client";

import { useQuery } from "@tanstack/react-query";
import { ChevronDown, LayoutGrid } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type ReactNode, useState } from "react";
import { TypeMark } from "@/components/browse/TypeMark";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { type BrowseType, fetchWorks, workKeys } from "@/lib/api/query";
import { useAuth } from "@/lib/auth";
import { buildBrowseHref } from "@/lib/browse-url";
import { readSessionPreference } from "@/lib/nsfw-preference";
import { TYPE_PLURALS, WORK_TYPES } from "@/lib/work-types";
import { BROWSE } from "./destinations";

/** BrowseMenu opens Browse and every type with its count, so a reader can jump straight to one. */
export function BrowseMenu() {
  const pathname = usePathname();
  const { account } = useAuth();
  const [open, setOpen] = useState(false);
  const nsfw = account === null ? readSessionPreference() : undefined;
  const counts = useQuery({
    enabled: open,
    queryFn: ({ signal }) => fetchWorks({ limit: 1, nsfw }, undefined, signal),
    queryKey: [...workKeys.all, "type-counts", nsfw],
  });
  const count = (type: BrowseType) =>
    counts.data?.types.find((held) => held.value === type)?.count ?? 0;

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          className="gap-1 px-3 text-ink data-[current=page]:text-accent"
          data-current={pathname === BROWSE.href ? "page" : undefined}
          variant="ghost"
        >
          {BROWSE.label}
          <ChevronDown aria-hidden="true" className="text-mute" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-60">
        <Row
          count={counts.data?.allTypes}
          href={BROWSE.href}
          icon={<LayoutGrid aria-hidden="true" strokeWidth={1.5} />}
          label="Everything"
        />
        <DropdownMenuSeparator />
        {WORK_TYPES.map((type) => (
          <Row
            count={counts.data ? count(type) : undefined}
            href={buildBrowseHref({ type })}
            icon={<TypeMark type={type} />}
            key={type}
            label={TYPE_PLURALS[type]}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Row({
  count,
  href,
  icon,
  label,
}: {
  count: number | undefined;
  href: string;
  icon: ReactNode;
  label: string;
}) {
  return (
    <DropdownMenuItem asChild className="[&_svg]:text-mute">
      <Link href={href}>
        {icon}
        <span className="flex-1">{label}</span>
        <span className="text-meta text-mute tabular-nums">{count}</span>
      </Link>
    </DropdownMenuItem>
  );
}
