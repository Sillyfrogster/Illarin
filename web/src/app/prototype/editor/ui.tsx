"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { type ClassValue, clsx } from "clsx";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { X } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
  useId,
} from "react";
import { extendTailwindMerge } from "tailwind-merge";

/** Portals must land inside the workspace root, or none of its theme tokens resolve */
const PortalTarget = createContext<HTMLElement | null>(null);
export const PortalTargetProvider = PortalTarget.Provider;
export function usePortalTarget() {
  return useContext(PortalTarget) ?? undefined;
}

const merge = extendTailwindMerge({ prefix: "ws" });
export function cn(...values: ClassValue[]) {
  return merge(clsx(values));
}

const buttonVariants = cva(
  "ws:relative ws:inline-flex ws:min-h-11 ws:shrink-0 ws:items-center ws:justify-center ws:gap-2 ws:rounded-full ws:px-5 ws:text-sm ws:font-semibold ws:tracking-tight ws:transition ws:duration-200 ws:disabled:cursor-not-allowed ws:disabled:opacity-40 ws:[&_svg]:size-4 ws:[&_svg]:shrink-0 ws:motion-reduce:transition-none",
  {
    variants: {
      variant: {
        primary:
          "ws:bg-emphasis ws:text-on-emphasis ws:hover:-translate-y-px ws:hover:shadow-lg ws:active:translate-y-0",
        quiet: "ws:text-ink ws:hover:bg-ink/8",
        outline:
          "ws:bg-card ws:text-ink ws:shadow-[inset_0_0_0_1px_var(--w-line)] ws:hover:bg-raised",
        critical:
          "ws:bg-critical-field ws:text-critical ws:shadow-[inset_0_0_0_1px_currentColor]",
      },
      size: {
        default: "",
        icon: "ws:size-11 ws:px-0",
        small: "ws:min-h-9 ws:px-3.5 ws:text-[0.8125rem]",
      },
    },
    defaultVariants: { variant: "quiet", size: "default" },
  },
);

export function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & { asChild?: boolean }) {
  const Component = asChild ? Slot : "button";
  return (
    <Component
      type="button"
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}

/** The publication action carries the spectral gradient */
export function SpectralButton({
  className,
  children,
  ...props
}: ComponentProps<"button">) {
  return (
    <button
      type="button"
      {...props}
      className={cn(
        "ws:group ws:relative ws:inline-flex ws:min-h-11 ws:shrink-0 ws:items-center ws:justify-center ws:gap-2 ws:overflow-hidden ws:rounded-full ws:bg-emphasis ws:px-6 ws:text-sm ws:font-semibold ws:text-on-emphasis ws:transition ws:duration-200 ws:hover:-translate-y-px ws:hover:shadow-lg ws:disabled:cursor-not-allowed ws:disabled:opacity-40 ws:[&_svg]:relative ws:[&_svg]:size-4 ws:motion-reduce:transition-none",
        className,
      )}
    >
      <span
        aria-hidden="true"
        className="ws:absolute ws:inset-x-0 ws:bottom-0 ws:h-[2px] ws:opacity-80 ws:transition-all ws:duration-300 ws:group-hover:h-full ws:group-hover:opacity-25 ws:motion-reduce:transition-none"
        style={{ background: "var(--w-spectrum)" }}
      />
      <span className="ws:relative ws:inline-flex ws:items-center ws:gap-2">
        {children}
      </span>
    </button>
  );
}

const fieldClass =
  "ws:block ws:w-full ws:min-w-0 ws:min-h-11 ws:rounded-xl ws:bg-card ws:px-3.5 ws:py-2.5 ws:text-base ws:text-ink ws:shadow-[inset_0_0_0_1px_var(--w-line)] ws:outline-none ws:placeholder:text-mute ws:focus:shadow-[inset_0_0_0_2px_var(--w-ink)] ws:disabled:opacity-55";

export function Input(props: ComponentProps<"input">) {
  return <input {...props} className={cn(fieldClass, props.className)} />;
}
export function Textarea(props: ComponentProps<"textarea">) {
  return (
    <textarea
      {...props}
      className={cn(fieldClass, "ws:resize-y ws:leading-7", props.className)}
    />
  );
}
/** A textarea with no box, because the panel around it is the field */
export function WritingArea(props: ComponentProps<"textarea">) {
  return (
    <textarea
      {...props}
      className={cn(
        "w-writing w-field ws:mx-auto ws:block ws:w-full ws:max-w-[68ch] ws:resize-none ws:bg-transparent ws:text-ink ws:shadow-none ws:outline-none ws:placeholder:text-mute ws:placeholder:italic ws:focus:shadow-none ws:focus-visible:shadow-none ws:disabled:opacity-55",
        props.className,
      )}
    />
  );
}

export function Select(props: ComponentProps<"select">) {
  return (
    <select {...props} className={cn(fieldClass, "ws:pr-9", props.className)} />
  );
}

export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: ReactNode;
  hint?: string;
}) {
  const id = useId();
  return (
    <div className="ws:min-w-0 ws:space-y-2">
      <label
        htmlFor={id}
        className="ws:block ws:text-[0.8125rem] ws:font-semibold ws:tracking-tight"
      >
        {label}
      </label>
      <Slot id={id} aria-describedby={hint ? `${id}-hint` : undefined}>
        {children}
      </Slot>
      {hint && (
        <span id={`${id}-hint`} className="ws:block ws:text-sm ws:text-mute">
          {hint}
        </span>
      )}
    </div>
  );
}

export function Eyebrow({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <p
      className={cn(
        "ws:text-[0.6875rem] ws:font-bold ws:uppercase ws:tracking-[0.18em] ws:text-mute",
        className,
      )}
    >
      {children}
    </p>
  );
}

export function Notice({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "amber" | "critical";
}) {
  return (
    <div
      role={tone === "critical" ? "alert" : "status"}
      className={cn(
        "ws:rounded-2xl ws:px-4 ws:py-3.5 ws:text-sm ws:leading-6",
        tone === "neutral" && "ws:bg-ink/6 ws:text-ink",
        tone === "amber" && "ws:bg-amber-field ws:text-ink",
        tone === "critical" && "ws:bg-critical-field ws:text-ink",
      )}
    >
      {children}
    </div>
  );
}

/** A pane that rises over the page and leaves it in view behind */
export function Sheet({
  open,
  onOpenChange,
  title,
  description,
  children,
  wide = false,
}: {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  title: string;
  description?: string;
  children: ReactNode;
  wide?: boolean;
}) {
  const reduced = useReducedMotion();
  const container = usePortalTarget();
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open && (
          <Dialog.Portal forceMount container={container}>
            <Dialog.Overlay asChild forceMount>
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: reduced ? 0 : 0.24 }}
                className="ws:fixed ws:inset-0 ws:z-60 ws:bg-black/35 ws:backdrop-blur-[3px]"
              />
            </Dialog.Overlay>
            <Dialog.Content asChild forceMount>
              <motion.div
                initial={{ y: reduced ? 0 : 40, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                exit={{ y: reduced ? 0 : 24, opacity: 0 }}
                transition={{
                  duration: reduced ? 0 : 0.34,
                  ease: [0.22, 1, 0.36, 1],
                }}
                className={cn(
                  "ws:fixed ws:inset-x-0 ws:bottom-0 ws:z-70 ws:mx-auto ws:flex ws:max-h-[92dvh] ws:w-full ws:flex-col ws:overflow-hidden ws:rounded-t-[28px] ws:bg-paper ws:text-ink ws:shadow-[var(--w-lift)]",
                  wide ? "ws:max-w-6xl" : "ws:max-w-3xl",
                )}
              >
                <div
                  aria-hidden="true"
                  className="w-spectral-rule ws:shrink-0"
                />
                <div className="ws:flex ws:shrink-0 ws:items-start ws:justify-between ws:gap-6 ws:px-6 ws:pt-6 ws:pb-4 ws:md:px-10 ws:md:pt-8">
                  <div className="ws:min-w-0">
                    <Dialog.Title className="ws:font-display ws:text-3xl ws:leading-tight ws:font-medium ws:wrap-anywhere ws:md:text-4xl">
                      {title}
                    </Dialog.Title>
                    {description && (
                      <Dialog.Description className="ws:mt-2 ws:max-w-2xl ws:text-sm ws:leading-6 ws:text-mute">
                        {description}
                      </Dialog.Description>
                    )}
                  </div>
                  <Dialog.Close asChild>
                    <Button size="icon" aria-label={`Close ${title}`}>
                      <X />
                    </Button>
                  </Dialog.Close>
                </div>
                <div className="ws:min-h-0 ws:flex-1 ws:overflow-y-auto ws:overscroll-contain ws:px-6 ws:pb-10 ws:md:px-10">
                  {children}
                </div>
              </motion.div>
            </Dialog.Content>
          </Dialog.Portal>
        )}
      </AnimatePresence>
    </Dialog.Root>
  );
}
