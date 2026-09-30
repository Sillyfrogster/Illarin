"use client";

import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronRight } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { FOLD } from "@/components/ui/collapsible";
import { cn, focusRing } from "@/lib/cn";

const Accordion = AccordionPrimitive.Root;

function AccordionItem({
  className,
  ...props
}: ComponentProps<typeof AccordionPrimitive.Item>) {
  return (
    <AccordionPrimitive.Item className={cn("min-w-0", className)} {...props} />
  );
}

/** AccordionTrigger is Fluid Functionalism's accordion row: the whole row presses, the chevron turns and the title firms up while open. */
function AccordionTrigger({
  children,
  trailing,
  className,
  ...props
}: ComponentProps<typeof AccordionPrimitive.Trigger> & {
  trailing?: ReactNode;
}) {
  return (
    <AccordionPrimitive.Header className="flex font-normal">
      <AccordionPrimitive.Trigger
        className={cn(
          "group/accordion -mx-3 flex min-h-control w-[calc(100%+1.5rem)] min-w-0 cursor-pointer flex-wrap items-center gap-x-3 gap-y-1 rounded-control px-3 py-1.5 text-left font-ui text-ui text-ink transition-colors duration-80 hover:bg-hover",
          focusRing,
          className,
        )}
        {...props}
      >
        <span className="flex min-w-0 flex-1 items-center gap-2">
          <ChevronRight
            aria-hidden="true"
            className="size-4 shrink-0 text-mute transition-transform duration-80 group-hover/accordion:text-ink group-data-[state=open]/accordion:rotate-90 motion-reduce:transition-none"
          />
          <span className="[font-variation-settings:'wght'_400] transition-[font-variation-settings] duration-80 group-data-[state=open]/accordion:[font-variation-settings:'wght'_600]">
            {children}
          </span>
        </span>
        {trailing ? (
          <span className="font-ui text-meta text-mute">{trailing}</span>
        ) : null}
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  );
}

/** AccordionContent unfolds to its own height under its row. */
function AccordionContent({
  className,
  children,
  ...props
}: ComponentProps<typeof AccordionPrimitive.Content>) {
  return (
    <AccordionPrimitive.Content className={FOLD} forceMount {...props}>
      <div className="min-h-0 overflow-hidden">
        <div className={className}>{children}</div>
      </div>
    </AccordionPrimitive.Content>
  );
}

export { Accordion, AccordionContent, AccordionItem, AccordionTrigger };
