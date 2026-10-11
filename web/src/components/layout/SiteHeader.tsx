"use client";

import { SiKofi } from "@icons-pack/react-simple-icons";
import { motion, useMotionValueEvent, useScroll } from "framer-motion";
import { Plus } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/cn";
import { KOFI_PAGE } from "@/lib/contact";
import { timing } from "@/lib/timing";
import { AccountMenu } from "./AccountMenu";
import { BrowseMenu } from "./BrowseMenu";
import { PUBLISH } from "./destinations";
import { shellClasses } from "./Shell";

const ROW =
  "flex h-[var(--header-height)] items-center gap-x-2 sm:gap-x-3 md:gap-x-5";

export function SiteHeader() {
  const pathname = usePathname();
  const { scrollY } = useScroll();
  const header = useRef<HTMLElement>(null);
  const scrollDirection = useRef({ direction: 0, anchor: 0 });
  const [hidden, setHidden] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useMotionValueEvent(scrollY, "change", (current) => {
    const previous = scrollY.getPrevious() ?? 0;
    const direction = Math.sign(current - previous);
    setScrolled(current > 0);
    if (current < 120) {
      setHidden(false);
      scrollDirection.current = { direction, anchor: current };
      return;
    }
    if (direction !== scrollDirection.current.direction) {
      scrollDirection.current = { direction, anchor: previous };
    }
    if (Math.abs(current - scrollDirection.current.anchor) < 12) return;
    if (direction < 0) setHidden(false);
    if (direction > 0 && !header.current?.contains(document.activeElement))
      setHidden(true);
  });

  useEffect(() => {
    if (pathname) setHidden(false);
  }, [pathname]);

  useEffect(() => setScrolled(scrollY.get() > 0), [scrollY]);

  return (
    <motion.header
      animate={{ y: hidden ? -160 : 0 }}
      className="sticky top-0 z-80 border-b border-transparent transition-colors duration-240 [view-transition-name:site-header] data-[scrolled=true]:border-rule data-[scrolled=true]:bg-field"
      data-scrolled={scrolled}
      data-site-header-hidden={hidden}
      initial={false}
      onFocusCapture={() => setHidden(false)}
      ref={header}
      transition={timing.settle}
    >
      <div className={cn(shellClasses, ROW)}>
        <Link
          href="/"
          aria-label="Illarin home"
          className="flex min-h-control items-center text-ink"
        >
          <BrandLogo className="w-8 sm:hidden" shape="mark" />
          <BrandLogo className="w-28 max-sm:hidden" />
        </Link>
        <BrowseMenu />
        <div className="ml-auto flex items-center sm:gap-1">
          <Tooltip content="Support Illarin on Ko-fi">
            <Button
              asChild
              className="max-sm:size-control max-sm:p-0"
              size="compact"
              variant="ghost"
            >
              <a
                aria-label="Support Illarin on Ko-fi"
                href={KOFI_PAGE}
                rel="noopener"
                target="_blank"
              >
                <SiKofi aria-hidden="true" title="" />
                <span className="max-sm:hidden">Support</span>
              </a>
            </Button>
          </Tooltip>
          <Button
            asChild
            className="publish max-sm:px-3 sm:mr-1 max-[380px]:size-control max-[380px]:p-0"
            variant="primary"
          >
            <Link
              aria-current={pathname === PUBLISH.href ? "page" : undefined}
              href={PUBLISH.href}
            >
              <Plus aria-hidden="true" className="min-[381px]:max-sm:hidden" />
              <span className="max-[380px]:sr-only">{PUBLISH.label}</span>
            </Link>
          </Button>
          <NotificationBell />
          <AccountMenu />
        </div>
      </div>
    </motion.header>
  );
}
