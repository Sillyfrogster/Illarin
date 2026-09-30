import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import {
  type ComponentProps,
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
} from "react";
import { Spinner } from "@/components/ui/spinner";
import { cn, focusRing } from "@/lib/cn";

const buttonVariants = cva(
  `group relative isolate inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-control font-ui text-ui font-medium transition-colors duration-80 disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:stroke-[1.75] [&_svg]:transition-[stroke-width] [&_svg]:duration-80 hover:[&_svg]:stroke-2 ${focusRing}`,
  {
    variants: {
      variant: {
        primary:
          "text-on-accent hover:text-on-accent focus-visible:ring-offset-1 focus-visible:ring-offset-field",
        secondary:
          "text-ink hover:text-accent focus-visible:ring-0 opened:text-accent",
        ghost: "text-mute hover:text-ink opened:text-ink",
        stop: "text-on-stop hover:text-on-stop focus-visible:ring-offset-1 focus-visible:ring-offset-field",
        link: "text-accent underline-offset-4 hover:text-accent hover:underline",
      },
      size: {
        default: "h-control px-4 has-[>span>svg:first-child]:pl-3",
        compact:
          "h-control-compact gap-1.5 px-3 text-meta has-[>span>svg:first-child]:pl-2",
        icon: "size-control px-0",
        "icon-compact": "size-control-compact px-0 [&_svg]:size-3.5",
      },
    },
    compoundVariants: [{ variant: "link", className: "h-auto px-0" }],
    defaultVariants: { variant: "secondary", size: "default" },
  },
);

type ButtonVariant = NonNullable<
  VariantProps<typeof buttonVariants>["variant"]
>;

// The fill sits 1px inside the button and a same-colour spread fills it back out, so a press shrinks it by exactly 1px a side
const SURFACES: Record<ButtonVariant, string> = {
  primary:
    "[--fill:var(--v-action)] group-hover:[--fill:color-mix(in_oklab,var(--v-action)_88%,var(--v-field))] group-active:[--fill:color-mix(in_oklab,var(--v-action)_78%,var(--v-field))] bg-(--fill) shadow-[0_0_0_1px_var(--fill)] group-active:shadow-[0_0_0_0px_var(--fill)]",
  secondary:
    "[--ring:var(--v-edge)] group-focus-visible:[--ring:var(--v-accent)] group-hover:[--ring:color-mix(in_oklab,var(--v-accent)_60%,transparent)] group-opened:[--ring:color-mix(in_oklab,var(--v-accent)_60%,transparent)] shadow-[0_0_0_1px_var(--ring),inset_0_0_0_0px_var(--ring)] group-hover:bg-hover group-active:bg-active group-opened:bg-active group-active:shadow-[0_0_0_0px_var(--ring),inset_0_0_0_1px_var(--ring)]",
  ghost:
    "shadow-[0_0_0_1px_transparent] group-hover:bg-hover group-hover:shadow-[0_0_0_1px_var(--v-hover)] group-active:bg-active group-opened:bg-active group-active:shadow-[0_0_0_0px_var(--v-active)]",
  stop: "[--fill:var(--v-stop)] group-hover:[--fill:color-mix(in_oklab,var(--v-stop)_88%,var(--v-field))] group-active:[--fill:color-mix(in_oklab,var(--v-stop)_78%,var(--v-field))] bg-(--fill) shadow-[0_0_0_1px_var(--fill)] group-active:shadow-[0_0_0_0px_var(--fill)]",
  link: "hidden",
};

type ButtonProps = ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    loading?: boolean;
  };

/** Button is Fluid Functionalism's button in the site's colours: a fill that presses in by a pixel, outlined and neutral at rest for the default variant. */
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
  const child =
    asChild && isValidElement(children)
      ? (children as ReactElement<{ children?: ReactNode }>)
      : null;
  const label = child ? child.props.children : children;
  const internals = (
    <>
      <span
        aria-hidden="true"
        className={cn(
          "absolute inset-px -z-10 rounded-[inherit] transition-[box-shadow,background-color] duration-160 group-active:duration-80",
          SURFACES[variant ?? "secondary"],
        )}
      />
      <span className="flex min-w-0 flex-1 items-center [justify-content:inherit] [gap:inherit]">
        {loading ? <Spinner /> : null}
        {label}
      </span>
    </>
  );
  const classes = cn(buttonVariants({ variant, size }), className);

  if (child) {
    return (
      <Slot className={classes} {...props}>
        {cloneElement(child, undefined, internals)}
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
      {internals}
    </button>
  );
}

export { Button, buttonVariants };
