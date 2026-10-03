import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { cn } from "@/lib/cn";

/** Gate stands in for a part of a page that needs one step first, such as signing in. */
export function Gate({
  action,
  children,
  className,
  heading,
  href,
  line,
}: {
  action: string;
  children?: ReactNode;
  className?: string;
  heading: string;
  href: string;
  line: string;
}) {
  return (
    <Empty className={cn("max-w-[34rem] bg-deep", className)}>
      <EmptyHeader>
        <EmptyMedia>
          <ShieldCheck aria-hidden="true" />
        </EmptyMedia>
        <EmptyTitle>{heading}</EmptyTitle>
        <EmptyDescription>{line}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button asChild variant="primary">
          <Link href={href}>{action}</Link>
        </Button>
        {children}
      </EmptyContent>
    </Empty>
  );
}
