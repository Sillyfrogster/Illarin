"use client";

import { PanelLeft } from "lucide-react";
import Link from "next/link";
import {
  type ComponentProps,
  type CSSProperties,
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/cn";

const WIDE = "15rem";
const NARROW = "3.75rem";
const PHONE_WIDTH = 768;
const REMEMBERED = "illarin.rail";

type RailState = {
  wide: boolean;
  openOnPhone: boolean;
  setOpenOnPhone: (open: boolean) => void;
  onPhone: boolean;
  toggle: () => void;
};

const RailContext = createContext<RailState | null>(null);

export function useRail() {
  const state = useContext(RailContext);
  if (!state) throw new Error("useRail needs a RailProvider");
  return state;
}

function useOnPhone() {
  const [onPhone, setOnPhone] = useState(false);
  useEffect(() => {
    const query = window.matchMedia(`(max-width: ${PHONE_WIDTH - 1}px)`);
    const update = () => setOnPhone(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return onPhone;
}

/** Holds whether the rail is wide or narrow, remembers it, and turns it into a drawer on a phone. */
export function RailProvider({
  className,
  children,
  ...props
}: ComponentProps<"div">) {
  const onPhone = useOnPhone();
  const [wide, setWide] = useState(true);
  const [openOnPhone, setOpenOnPhone] = useState(false);

  useEffect(() => {
    try {
      setWide(window.localStorage.getItem(REMEMBERED) !== "narrow");
    } catch {
      // A browser that refuses storage keeps the wide rail.
    }
  }, []);

  const toggle = useCallback(() => {
    if (onPhone) {
      setOpenOnPhone((open) => !open);
      return;
    }
    setWide((open) => {
      try {
        window.localStorage.setItem(REMEMBERED, open ? "narrow" : "wide");
      } catch {
        // Not remembering it is no reason to refuse the toggle.
      }
      return !open;
    });
  }, [onPhone]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "b" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        toggle();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [toggle]);

  const state = useMemo(
    () => ({ wide, openOnPhone, setOpenOnPhone, onPhone, toggle }),
    [wide, openOnPhone, onPhone, toggle],
  );

  return (
    <RailContext.Provider value={state}>
      <div
        className={cn("flex min-h-svh w-full", className)}
        data-rail={wide ? "wide" : "narrow"}
        style={{ "--rail": wide ? WIDE : NARROW } as CSSProperties}
        {...props}
      >
        {children}
      </div>
    </RailContext.Provider>
  );
}

export function Rail({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  const { onPhone, openOnPhone, setOpenOnPhone } = useRail();

  if (onPhone) {
    return (
      <Sheet open={openOnPhone} onOpenChange={setOpenOnPhone}>
        <SheetContent className="w-[17rem] bg-inset p-0" side="left">
          <SheetTitle className="sr-only">{label}</SheetTitle>
          <SheetDescription className="sr-only">
            The console's sections.
          </SheetDescription>
          <div className="flex h-full flex-col">{children}</div>
        </SheetContent>
      </Sheet>
    );
  }

  return (
    <>
      <div className="hidden w-[var(--rail)] shrink-0 transition-[width] duration-200 ease-[var(--ease-wipe)] motion-reduce:transition-none md:block" />
      <aside
        aria-label={label}
        className="fixed inset-y-0 left-0 z-30 hidden w-[var(--rail)] flex-col border-r border-rule bg-inset transition-[width] duration-200 ease-[var(--ease-wipe)] motion-reduce:transition-none md:flex"
      >
        {children}
      </aside>
    </>
  );
}

export function RailToggle({
  className,
  onClick,
  ...props
}: ComponentProps<typeof Button>) {
  const { toggle, wide, onPhone } = useRail();
  return (
    <Button
      aria-expanded={onPhone ? undefined : wide}
      className={className}
      onClick={(event) => {
        onClick?.(event);
        toggle();
      }}
      size="icon"
      variant="ghost"
      {...props}
    >
      <PanelLeft aria-hidden="true" />
      <span className="sr-only">Show or hide the sections</span>
    </Button>
  );
}

export function RailHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "flex h-14 shrink-0 items-center gap-2 border-b border-rule px-3",
        className,
      )}
      {...props}
    />
  );
}

export function RailBody({ className, ...props }: ComponentProps<"nav">) {
  return (
    <nav
      className={cn(
        "flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto overflow-x-hidden py-4",
        className,
      )}
      {...props}
    />
  );
}

export function RailFoot({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "shrink-0 border-t border-rule px-3 py-3 group-data-[rail=narrow]/rail:px-2",
        className,
      )}
      {...props}
    />
  );
}

export function RailGroup({
  label,
  className,
  children,
}: {
  label: string;
  className?: string;
  children: ReactNode;
}) {
  const { wide, onPhone } = useRail();
  const showLabel = wide || onPhone;
  return (
    <div className={cn("px-2", className)}>
      <p
        aria-hidden={showLabel ? undefined : "true"}
        className={cn(
          "px-2 pb-1.5 font-ui text-label font-medium tracking-[0.02em] text-mute",
          showLabel ? "" : "sr-only",
        )}
      >
        {label}
      </p>
      <ul className="flex list-none flex-col gap-0.5">{children}</ul>
    </div>
  );
}

export function RailItem({
  current = false,
  icon,
  badge,
  className,
  children,
  ...props
}: ComponentProps<typeof Link> & {
  current?: boolean;
  icon: ReactNode;
  badge?: ReactNode;
}) {
  const { wide, onPhone } = useRail();
  const showLabel = wide || onPhone;
  return (
    <li>
      <Link
        aria-current={current ? "page" : undefined}
        className={cn(
          "group/item relative flex min-h-11 w-full items-center gap-3 rounded-control px-2.5 font-ui text-ui whitespace-nowrap text-mute outline-offset-2 transition-colors duration-150 hover:bg-deep hover:text-ink motion-reduce:transition-none",
          "aria-[current=page]:bg-accent-wash aria-[current=page]:font-medium aria-[current=page]:text-ink",
          "[&_svg]:size-[1.15rem] [&_svg]:shrink-0 [&_svg]:text-mute aria-[current=page]:[&_svg]:text-accent",
          className,
        )}
        title={showLabel ? undefined : String(children)}
        {...props}
      >
        <span
          aria-hidden="true"
          className="absolute top-2 bottom-2 -left-2.5 w-[3px] rounded-full bg-accent opacity-0 group-aria-[current=page]/item:opacity-100"
        />
        {icon}
        <span className={showLabel ? "min-w-0 truncate" : "sr-only"}>
          {children}
        </span>
        {badge && showLabel ? <span className="ml-auto">{badge}</span> : null}
      </Link>
    </li>
  );
}

export function RailBadge({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      className={cn(
        "flex h-5 min-w-5 items-center justify-center rounded-full bg-accent-wash px-1.5 font-ui text-label font-medium text-accent tabular-nums",
        className,
      )}
      {...props}
    />
  );
}

export function RailInset({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex min-w-0 flex-1 flex-col bg-field", className)}
      {...props}
    />
  );
}
