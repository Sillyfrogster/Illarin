import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";

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
    <div className={cn("max-w-[34rem] rounded-plate bg-deep p-7", className)}>
      <ShieldCheck
        aria-hidden="true"
        className="size-7 text-accent"
        strokeWidth={1.4}
      />
      <h2 className="mt-4 font-display text-section font-medium tracking-tight text-ink">
        {heading}
      </h2>
      <p className="mt-2 font-prose text-prose text-mute">{line}</p>
      <div className="mt-6 flex flex-wrap items-center gap-3">
        <Button asChild variant="primary">
          <Link href={href}>{action}</Link>
        </Button>
        {children}
      </div>
    </div>
  );
}
