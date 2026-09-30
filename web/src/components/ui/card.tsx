import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";
import { Elevated } from "@/lib/elevated";

/** Card is Fluid Functionalism's card surface: one step above what it sits on, rounded, with no drawn frame. */
function Card({
  className,
  ...props
}: Omit<ComponentProps<typeof Elevated>, "offset">) {
  return (
    <Elevated
      className={cn(
        "flex min-w-0 flex-col overflow-hidden rounded-plate pb-4",
        className,
      )}
      data-slot="card"
      offset={1}
      {...props}
    />
  );
}

/** CardHeader stacks the title and description, with room on the right for one CardAction. */
function CardHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "grid auto-rows-min items-start gap-1 px-4 pt-4 has-data-[slot=card-action]:grid-cols-[1fr_auto]",
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
      className={cn("min-w-0 px-4 pt-3", className)}
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
        "mt-auto flex flex-wrap items-center gap-x-6 gap-y-2 px-4 pt-3 font-ui text-meta text-mute",
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
