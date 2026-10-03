"use client";

import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { Check } from "lucide-react";
import type { ComponentProps } from "react";
import { cn, popupRow, popupSurface } from "@/lib/cn";

/** DropdownMenu is a menu of actions or choices that never locks the page behind it. */
function DropdownMenu(
  props: ComponentProps<typeof DropdownMenuPrimitive.Root>,
) {
  return <DropdownMenuPrimitive.Root modal={false} {...props} />;
}

const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
const DropdownMenuGroup = DropdownMenuPrimitive.Group;
const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;

/** DropdownMenuContent is the menu's plane, flipping to whichever side of its trigger has room. */
function DropdownMenuContent({
  className,
  align = "start",
  sideOffset = 6,
  collisionPadding = 16,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Content>) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        align={align}
        className={cn(
          popupSurface,
          "flex max-h-[min(30rem,var(--radix-dropdown-menu-content-available-height))] w-max max-w-[calc(100vw-2rem)] min-w-[max(12rem,var(--radix-dropdown-menu-trigger-width))] flex-col overflow-y-auto p-1 select-none",
          className,
        )}
        collisionPadding={collisionPadding}
        sideOffset={sideOffset}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
}

function DropdownMenuItem({
  className,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.Item>) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(popupRow, "data-[current=page]:text-accent", className)}
      {...props}
    />
  );
}

function DropdownMenuRadioItem({
  className,
  children,
  ...props
}: ComponentProps<typeof DropdownMenuPrimitive.RadioItem>) {
  return (
    <DropdownMenuPrimitive.RadioItem
      className={cn(popupRow, "data-[state=checked]:text-accent", className)}
      {...props}
    >
      {children}
      <DropdownMenuPrimitive.ItemIndicator className="ml-auto flex">
        <Check aria-hidden="true" />
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
      className={cn("px-2 pt-1.5 pb-2 text-meta text-mute", className)}
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
      className={cn("-mx-1 my-1 h-px shrink-0 bg-rule", className)}
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
