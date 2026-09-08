"use client";

import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { Check } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const DropdownMenu = DropdownMenuPrimitive.Root;
const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
const DropdownMenuGroup = DropdownMenuPrimitive.Group;
const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

/** The surface a menu opens onto, sized to its longest entry */
function DropdownMenuContent({
  className,
  sideOffset = 10,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        sideOffset={sideOffset}
        collisionPadding={16}
        className={cn(
          "z-90 max-h-[var(--radix-dropdown-menu-content-available-height)] min-w-[min(17rem,calc(100vw-2rem))] origin-[var(--radix-dropdown-menu-content-transform-origin)] overflow-y-auto rounded-plate bg-plane p-2 font-ui text-ink shadow-popover outline-none",
          "inset-ring inset-ring-rule/70 motion-safe:data-[state=closed]:animate-pop-out motion-safe:data-[state=open]:animate-pop-in",
          className,
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}

/** One row a pointer or the keyboard can land on */
function DropdownMenuItem({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Item>) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(
        "flex min-h-11 cursor-pointer items-center gap-2.5 rounded-control px-3 text-ui outline-none select-none",
        "text-ink focus-visible:outline-none data-[highlighted]:bg-accent-wash data-[highlighted]:text-ink data-[current=page]:text-accent data-[disabled]:pointer-events-none data-[disabled]:opacity-45 [&_svg]:size-4 [&_svg]:shrink-0",
        className,
      )}
      {...props}
    />
  );
}

/** One choice in a set where exactly one is in force */
function DropdownMenuRadioItem({
  className,
  children,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.RadioItem>) {
  return (
    <DropdownMenuPrimitive.RadioItem
      className={cn(
        "flex min-h-11 cursor-pointer items-center gap-2.5 rounded-control px-3 text-ui outline-none select-none",
        "focus-visible:outline-none data-[highlighted]:bg-accent-wash data-[highlighted]:text-ink data-[disabled]:pointer-events-none data-[disabled]:opacity-45 data-[state=checked]:text-accent [&_svg]:size-4 [&_svg]:shrink-0",
        className,
      )}
      {...props}
    >
      {children}
      <DropdownMenuPrimitive.ItemIndicator className="ml-auto">
        <Check />
      </DropdownMenuPrimitive.ItemIndicator>
    </DropdownMenuPrimitive.RadioItem>
  );
}

function DropdownMenuLabel({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Label>) {
  return (
    <DropdownMenuPrimitive.Label
      className={cn("px-3 pt-2 pb-3", className)}
      {...props}
    />
  );
}

function DropdownMenuSeparator({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return (
    <DropdownMenuPrimitive.Separator
      className={cn("my-2 h-px bg-rule/70", className)}
      {...props}
    />
  );
}

export {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
};
