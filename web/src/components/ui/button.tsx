import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { LoaderCircle } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

const buttonVariants = cva(
  "relative inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-control font-ui text-ui font-medium tracking-tight transition duration-200 outline-offset-3 disabled:pointer-events-none disabled:opacity-45 motion-reduce:transition-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        primary:
          "bg-action text-on-accent shadow-[0_4px_14px_-5px_var(--v-action),inset_0_1px_0_rgb(255_255_255/0.18)] hover:bg-action/90 hover:text-on-accent motion-safe:hover:-translate-y-px active:translate-y-0",
        secondary: "bg-deep text-ink hover:bg-rule/45",
        outline: "text-ink inset-ring inset-ring-edge hover:bg-deep",
        ghost: "text-mute hover:bg-deep hover:text-ink",
        stop: "bg-stop text-on-stop hover:-translate-y-px active:translate-y-0",
        link: "px-0 text-accent underline-offset-4 hover:underline",
      },
      size: {
        default: "min-h-11 px-5",
        large: "min-h-12 px-7 text-base",
        compact: "min-h-11 px-3",
        icon: "size-11 px-0",
      },
    },
    defaultVariants: { variant: "secondary", size: "default" },
  },
);

type ButtonProps = ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    loading?: boolean;
  };

/** Every action on the site, in one hierarchy */
export function Button({
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
      {loading ? (
        <LoaderCircle
          aria-hidden="true"
          className="animate-spin motion-reduce:animate-none"
        />
      ) : null}
      {children}
    </button>
  );
}

export { buttonVariants };
