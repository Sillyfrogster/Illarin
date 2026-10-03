"use client";

import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog";
import type { ComponentProps } from "react";
import { PANEL, SCRIM } from "@/components/ui/dialog";
import { cn } from "@/lib/cn";

/** AlertDialog confirms what cannot be undone; focus starts on the safe choice and Escape cancels. */
const AlertDialog = AlertDialogPrimitive.Root;
const AlertDialogCancel = AlertDialogPrimitive.Cancel;
const AlertDialogAction = AlertDialogPrimitive.Action;

function AlertDialogContent({
  className,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Content>) {
  return (
    <AlertDialogPrimitive.Portal>
      <AlertDialogPrimitive.Overlay className={SCRIM} />
      <AlertDialogPrimitive.Content
        className={cn(
          PANEL,
          "top-1/2 max-w-[32rem] -translate-y-1/2 gap-3 p-6",
          className,
        )}
        {...props}
      />
    </AlertDialogPrimitive.Portal>
  );
}

function AlertDialogTitle({
  className,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return (
    <AlertDialogPrimitive.Title
      className={cn("text-section font-medium text-ink", className)}
      {...props}
    />
  );
}

function AlertDialogDescription({
  className,
  ...props
}: ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return (
    <AlertDialogPrimitive.Description
      className={cn("text-ui text-mute", className)}
      {...props}
    />
  );
}

function AlertDialogFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("mt-3 flex flex-wrap justify-end gap-2", className)}
      {...props}
    />
  );
}

export {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
};
