"use client";

import { PanelLeft } from "lucide-react";
import Link from "next/link";
import {
  Children,
  type ComponentProps,
  createContext,
  isValidElement,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { FluidHoverHighlight } from "@/components/ui/fluid-hover-highlight";
import { MobileDrawer } from "@/components/ui/mobile-drawer";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import { fontWeights } from "@/lib/font-weight";
import {
  useFluidHover,
  useRegisterFluidHoverItem,
} from "@/lib/use-fluid-hover";

const PHONE_WIDTH = 768;
const REMEMBERED = "illarin.sidebar";

type SidebarState = {
  open: boolean;
  onPhone: boolean;
  openOnPhone: boolean;
  setOpenOnPhone: (open: boolean) => void;
  toggle: () => void;
};

const SidebarContext = createContext<SidebarState | null>(null);

export function useSidebar() {
  const state = useContext(SidebarContext);
  if (!state) throw new Error("useSidebar needs a SidebarProvider");
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

/** SidebarProvider holds whether the sidebar is open, remembers it, answers Ctrl+B, and swaps it for a drawer on a phone. */
export function SidebarProvider({
  className,
  children,
  ...props
}: ComponentProps<"div">) {
  const onPhone = useOnPhone();
  const [open, setOpen] = useState(true);
  const [openOnPhone, setOpenOnPhone] = useState(false);

  useEffect(() => {
    try {
      setOpen(window.localStorage.getItem(REMEMBERED) !== "closed");
    } catch {
      // A browser that refuses storage keeps the sidebar open.
    }
  }, []);

  const toggle = useCallback(() => {
    if (onPhone) {
      setOpenOnPhone((was) => !was);
      return;
    }
    setOpen((was) => {
      try {
        window.localStorage.setItem(REMEMBERED, was ? "closed" : "open");
      } catch {
        // Not remembering it is no reason to refuse the toggle.
      }
      return !was;
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
    () => ({ open, onPhone, openOnPhone, setOpenOnPhone, toggle }),
    [open, onPhone, openOnPhone, toggle],
  );

  return (
    <SidebarContext.Provider value={state}>
      <div className={cn("flex min-h-svh w-full", className)} {...props}>
        {children}
      </div>
    </SidebarContext.Provider>
  );
}

/** Sidebar is Fluid Functionalism's sidebar: a fixed column that slides fully away when closed, and a drawer on a phone. */
export function Sidebar({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  const { open, onPhone, openOnPhone, setOpenOnPhone } = useSidebar();

  if (onPhone) {
    return (
      <MobileDrawer
        onClose={() => setOpenOnPhone(false)}
        open={openOnPhone}
        title={label}
      >
        {children}
      </MobileDrawer>
    );
  }

  return (
    <>
      <div
        className={cn(
          "hidden shrink-0 transition-[width] duration-240 ease-[var(--ease-wipe)] motion-reduce:transition-none md:block",
          open ? "w-60" : "w-0",
        )}
      />
      <aside
        aria-label={label}
        className={cn(
          "fixed inset-y-0 left-0 z-30 hidden w-60 flex-col border-r border-rule bg-inset transition-[translate,visibility] duration-240 ease-[var(--ease-wipe)] motion-reduce:transition-none md:flex",
          open ? "translate-x-0" : "invisible -translate-x-full",
        )}
      >
        {children}
      </aside>
    </>
  );
}

export function SidebarTrigger() {
  const { toggle, open, onPhone } = useSidebar();
  const name = "Show or hide the sections";
  return (
    <Tooltip content={name} side="bottom">
      <Button
        aria-expanded={onPhone ? undefined : open}
        aria-label={name}
        onClick={toggle}
        size="icon"
        variant="ghost"
      >
        <PanelLeft aria-hidden="true" />
      </Button>
    </Tooltip>
  );
}

export function SidebarHeader({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("flex h-13 shrink-0 items-center gap-2 px-3", className)}
      {...props}
    />
  );
}

export function SidebarBody({ className, ...props }: ComponentProps<"nav">) {
  return (
    <nav
      className={cn(
        "flex min-h-0 flex-1 flex-col gap-5 overflow-x-hidden overflow-y-auto py-3",
        className,
      )}
      {...props}
    />
  );
}

export function SidebarFooter({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn("shrink-0 border-t border-rule p-3", className)}
      {...props}
    />
  );
}

const GroupContext = createContext<
  ((index: number, element: HTMLElement | null) => void) | undefined
>(undefined);

/** SidebarGroup names a run of items; the hover plate follows the pointer between them. */
export function SidebarGroup({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  const list = useRef<HTMLUListElement>(null);
  const hover = useFluidHover(list);
  let index = 0;
  const items = Children.map(children, (child) =>
    isValidElement(child) ? (
      <IndexContext value={index++}>{child}</IndexContext>
    ) : (
      child
    ),
  );

  return (
    <div className="px-3">
      <p className="px-2 pb-1 font-ui text-meta text-mute">{label}</p>
      <GroupContext value={hover.registerItem}>
        <ul
          className="relative flex list-none flex-col"
          onMouseEnter={hover.handlers.onMouseEnter}
          onMouseLeave={hover.handlers.onMouseLeave}
          onMouseMove={hover.handlers.onMouseMove}
          ref={list}
        >
          <FluidHoverHighlight className="rounded-control" hover={hover} />
          {items}
        </ul>
      </GroupContext>
    </div>
  );
}

const IndexContext = createContext<number | undefined>(undefined);

/** SidebarItem is one section: the current one sits on the violet wash in violet at the heavier weight. */
export function SidebarItem({
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
  const row = useRef<HTMLLIElement>(null);
  const index = useContext(IndexContext);
  useRegisterFluidHoverItem(useContext(GroupContext), index, row);

  return (
    <li className="relative z-10" data-fluid-hover-index={index} ref={row}>
      <Link
        aria-current={current ? "page" : undefined}
        className={cn(
          "flex min-h-control w-full items-center gap-2.5 rounded-control px-2 font-ui text-ui whitespace-nowrap text-mute transition-colors duration-80 hover:text-ink",
          "aria-[current=page]:bg-accent-wash aria-[current=page]:text-accent",
          "[&_svg]:size-4 [&_svg]:shrink-0",
          className,
        )}
        style={{
          fontVariationSettings: current
            ? fontWeights.semibold
            : fontWeights.normal,
        }}
        {...props}
      >
        {icon}
        <span className="min-w-0 truncate">{children}</span>
        {badge ? <span className="ml-auto">{badge}</span> : null}
      </Link>
    </li>
  );
}

export function SidebarInset({ className, ...props }: ComponentProps<"main">) {
  return (
    <main
      className={cn("flex min-w-0 flex-1 flex-col bg-field", className)}
      {...props}
    />
  );
}
