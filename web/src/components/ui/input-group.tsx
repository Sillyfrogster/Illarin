"use client";

import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps, MouseEvent } from "react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/cn";

/** InputGroup is shadcn's input group: one outlined field that holds an icon, a prefix, a key hint or a button beside the text. */
function InputGroup({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "group/input-group relative flex h-control w-full min-w-0 items-center rounded-control border border-edge font-ui text-ui shadow-[0_1px_2px_0_rgb(0_0_0/0.05)] transition-[background-color,border-color,box-shadow] duration-80 hover:border-accent/50 hover:bg-hover dark:bg-edge/10",
        "has-[>[data-align=inline-start]]:[&>input]:pl-1.5 has-[>[data-align=inline-end]]:[&>input]:pr-1.5",
        "has-[input:focus-visible]:border-accent has-[input:focus-visible]:bg-transparent",
        "has-[[aria-invalid=true]]:border-stop has-[input:disabled]:opacity-50",
        className,
      )}
      data-slot="input-group"
      {...props}
    />
  );
}

const addonVariants = cva(
  "flex h-auto cursor-text items-center justify-center gap-2 text-mute select-none [&>svg:not([class*='size-'])]:size-4",
  {
    variants: {
      align: {
        "inline-start": "order-first pl-3 has-[>button]:ml-[-0.4rem]",
        "inline-end": "order-last pr-3 has-[>button]:mr-[-0.4rem]",
      },
    },
    defaultVariants: { align: "inline-start" },
  },
);

function focusInput(event: MouseEvent<HTMLDivElement>) {
  if ((event.target as HTMLElement).closest("button")) return;
  event.currentTarget.parentElement?.querySelector("input")?.focus();
}

function InputGroupAddon({
  className,
  align = "inline-start",
  ...props
}: ComponentProps<"div"> & VariantProps<typeof addonVariants>) {
  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: a click on the addon only forwards focus to the field, which the keyboard reaches directly.
    // biome-ignore lint/a11y/noStaticElementInteractions: see above.
    <div
      className={cn(addonVariants({ align }), className)}
      data-align={align}
      onClick={focusInput}
      {...props}
    />
  );
}

function InputGroupInput({ className, ...props }: ComponentProps<"input">) {
  return (
    <Input
      className={cn(
        "h-full flex-1 rounded-none border-0 bg-transparent shadow-none hover:bg-transparent dark:bg-transparent",
        className,
      )}
      {...props}
    />
  );
}

export { InputGroup, InputGroupAddon, InputGroupInput };
