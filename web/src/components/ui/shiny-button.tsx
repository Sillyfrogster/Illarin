"use client";

import {
  animate,
  type HTMLMotionProps,
  type MotionStyle,
  motion,
  useMotionValue,
  useReducedMotion,
} from "framer-motion";
import Link from "next/link";
import type { ReactNode } from "react";
import { Spinner } from "@/components/ui/spinner";
import { cn, focusRing } from "@/lib/cn";
import { spring } from "@/lib/springs";

const MotionLink = motion.create(Link);

const SHINY = `relative inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-control bg-action font-ui text-ui font-medium text-on-accent transition-shadow duration-160 hover:text-on-accent hover:shadow-[0_0_20px_color-mix(in_oklab,var(--v-action)_40%,transparent)] disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 ${focusRing}`;

const SIZES = {
  default: "h-control px-4 has-[>span>svg:first-child]:pl-3",
  compact: "h-control-compact gap-1.5 px-3 text-meta",
};

const TEXT_MASK =
  "linear-gradient(-75deg,#000 calc(var(--x) + 20%),rgb(0 0 0/0.55) calc(var(--x) + 30%),#000 calc(var(--x) + 100%))";

const EDGE_MASK =
  "linear-gradient(rgb(0,0,0), rgb(0,0,0)) content-box exclude,linear-gradient(rgb(0,0,0), rgb(0,0,0))";

type ShinyButtonProps = Omit<HTMLMotionProps<"button">, "children"> & {
  children: ReactNode;
  href?: string;
  loading?: boolean;
  size?: keyof typeof SIZES;
};

/** ShinyButton is the one Publish button: Magic UI's shiny button in the action violet, shining once each time the pointer arrives. */
function ShinyButton({
  children,
  className,
  href,
  loading = false,
  size = "default",
  disabled,
  ...props
}: ShinyButtonProps) {
  const x = useMotionValue("100%");
  const reduceMotion = useReducedMotion();
  const shared = {
    className: cn(SHINY, SIZES[size], className),
    onHoverStart: () => {
      if (!reduceMotion) animate(x, ["100%", "-100%"], spring.slow);
    },
    style: { "--x": x } as MotionStyle,
    whileTap: reduceMotion ? undefined : { scale: 0.97 },
    transition: spring.fast,
  };
  const inside = (
    <>
      <span
        className="relative flex size-full items-center justify-center [gap:inherit]"
        style={{ maskImage: TEXT_MASK }}
      >
        {loading ? <Spinner /> : null}
        {children}
      </span>
      <span
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 z-10 block rounded-[inherit] p-px"
        style={{
          mask: EDGE_MASK,
          WebkitMask: EDGE_MASK,
          backgroundImage:
            "linear-gradient(-75deg,rgb(255 255 255/0.1) calc(var(--x) + 20%),rgb(255 255 255/0.5) calc(var(--x) + 25%),rgb(255 255 255/0.1) calc(var(--x) + 100%))",
        }}
      />
    </>
  );

  if (href) {
    const { "aria-current": current } = props;
    return (
      <MotionLink aria-current={current} href={href} {...shared}>
        {inside}
      </MotionLink>
    );
  }

  return (
    <motion.button
      aria-busy={loading || undefined}
      disabled={disabled || loading}
      type="button"
      {...shared}
      {...props}
    >
      {inside}
    </motion.button>
  );
}

export { ShinyButton };
