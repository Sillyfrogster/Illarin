"use client";

import * as CollapsiblePrimitive from "@radix-ui/react-collapsible";
import type { ComponentProps } from "react";

const Collapsible = CollapsiblePrimitive.Root;
const CollapsibleTrigger = CollapsiblePrimitive.Trigger;

/** FOLD opens a panel to its own height on the fast tier, as Fluid Functionalism's accordion does, and hides it from keys and readers once shut. */
const FOLD =
  "grid transition-[grid-template-rows,opacity,visibility] duration-80 data-[state=closed]:invisible data-[state=closed]:grid-rows-[0fr] data-[state=closed]:opacity-0 data-[state=open]:grid-rows-[1fr] motion-reduce:transition-none";

/** CollapsibleContent is shadcn's collapsible panel, folding on the fast tier. */
function CollapsibleContent({
  className,
  children,
  ...props
}: ComponentProps<typeof CollapsiblePrimitive.Content>) {
  return (
    <CollapsiblePrimitive.Content className={FOLD} forceMount {...props}>
      <div className="min-h-0 overflow-hidden">
        <div className={className}>{children}</div>
      </div>
    </CollapsiblePrimitive.Content>
  );
}

export { Collapsible, CollapsibleContent, CollapsibleTrigger, FOLD };
