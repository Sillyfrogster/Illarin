import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Card groups a section on a flat quiet fill with the card corners, and no frame or shadow. */
function Card({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex min-w-0 flex-col overflow-hidden rounded-card bg-inset pb-5",
        className,
      )}
      data-slot="card"
      {...props}
    />
  );
}

/** CardHeader stacks the title and description, with room on the right for one CardAction. */
function CardHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "grid auto-rows-min items-start gap-1 px-5 pt-5 has-data-[slot=card-action]:grid-cols-[1fr_auto]",
        className,
      )}
      data-slot="card-header"
      {...props}
    />
  );
}

function CardTitle({ className, ...props }: ComponentProps<"h2">) {
  return (
    <h2
      className={cn("font-ui text-ui font-medium text-ink", className)}
      data-slot="card-title"
      {...props}
    />
  );
}

function CardDescription({ className, ...props }: ComponentProps<"p">) {
  return (
    <p
      className={cn("font-ui text-meta text-mute", className)}
      data-slot="card-description"
      {...props}
    />
  );
}

function CardAction({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "col-start-2 row-span-2 row-start-1 self-start justify-self-end",
        className,
      )}
      data-slot="card-action"
      {...props}
    />
  );
}

function CardContent({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("min-w-0 px-5 pt-3", className)}
      data-slot="card-content"
      {...props}
    />
  );
}

/** CardFooter holds a legend, a total or the card's actions, at the bottom however tall the card grows. */
function CardFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mt-auto flex flex-wrap items-center gap-x-6 gap-y-2 px-5 pt-3 font-ui text-meta text-mute",
        className,
      )}
      data-slot="card-footer"
      {...props}
    />
  );
}

export {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
};
