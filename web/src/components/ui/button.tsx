import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { Spinner } from "@/components/ui/spinner";
import { cn, focusRing } from "@/lib/cn";

const FILLED_FOCUS =
  "focus-visible:ring-offset-1 focus-visible:ring-offset-field";

const buttonVariants = cva(
  `inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-control font-ui text-ui font-medium transition-colors duration-80 disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 ${focusRing}`,
  {
    variants: {
      variant: {
        primary: `bg-action text-on-accent hover:bg-action-hover hover:text-on-accent ${FILLED_FOCUS}`,
        secondary: `bg-fill text-ink hover:bg-fill-hover hover:text-ink opened:bg-fill-hover ${FILLED_FOCUS}`,
        ghost:
          "text-mute hover:bg-fill hover:text-ink opened:bg-fill opened:text-ink",
        stop: `bg-stop text-on-stop hover:bg-stop-hover hover:text-on-stop ${FILLED_FOCUS}`,
        link: "text-accent underline-offset-4 hover:text-accent hover:underline",
      },
      size: {
        default: "h-control px-4 has-[>svg:first-child]:pl-3",
        compact:
          "h-control-compact gap-1.5 px-3 text-meta has-[>svg:first-child]:pl-2",
        icon: "size-control px-0",
        "icon-compact": "size-control-compact px-0 [&_svg]:size-3.5",
      },
    },
    compoundVariants: [{ variant: "link", className: "h-auto px-0" }],
    defaultVariants: { variant: "secondary", size: "default" },
  },
);

type ButtonProps = ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    loading?: boolean;
  };

/** Button is a filled control: violet for the main action, grey for the rest, one step lighter in dark or darker in light on hover. */
function Button({
  className,
  variant,
  size,
  asChild = false,
  loading = false,
  disabled,
  children,
  ...props
}: ButtonProps) {
  const classes = cn(buttonVariants({ variant, size }), className);
  if (asChild) {
    return (
      <Slot className={classes} {...props}>
        {children}
      </Slot>
    );
  }
  return (
    <button
      type="button"
      aria-busy={loading || undefined}
      disabled={disabled || loading}
      className={classes}
      {...props}
    >
      {loading ? <Spinner /> : null}
      {children}
    </button>
  );
}

export { Button, buttonVariants };
