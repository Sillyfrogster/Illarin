"use client";

import {
  motion,
  useMotionTemplate,
  useScroll,
  useTransform,
} from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandMark } from "@/components/brand/BrandMark";
import { Button } from "@/components/ui/button";
import { LineLink } from "@/components/ui/line-link";
import { useAuth } from "@/lib/auth";
import { AccountMenu } from "./AccountMenu";
import { AppearanceMenu } from "./AppearanceMenu";
import { isCurrentPage, NAV, publishAction } from "./destinations";
import { MobileNav } from "./MobileNav";
import { Notch } from "./Notch";

/** The one header every page above the blog shares */
export function SiteHeader() {
  const pathname = usePathname();
  const { account } = useAuth();
  const publish = publishAction(account);
  const { scrollY } = useScroll();
  const depth = useTransform(scrollY, [0, 40], [0, 0.32], { clamp: true });
  const lift = useMotionTemplate`drop-shadow(0 6px 10px rgb(0 0 0 / ${depth}))`;

  return (
    <motion.header
      style={{ filter: lift }}
      className="sticky top-0 z-80 h-[var(--header-height)]"
    >
      <Notch
        start={
          <>
            <MobileNav />
            <nav
              className="hidden items-center gap-6 md:flex"
              aria-label="Primary"
            >
              {NAV.map((item) => (
                <LineLink
                  key={item.href}
                  href={item.href}
                  current={isCurrentPage(pathname, item.href)}
                >
                  {item.label}
                </LineLink>
              ))}
            </nav>
          </>
        }
        centre={
          <Link
            href="/"
            aria-label="Illarin home"
            className="flex min-h-11 items-center gap-2 text-ink"
          >
            <BrandMark size={22} tone="accent" />
            <span className="font-display text-[1.375rem] leading-none font-medium tracking-[-0.03em] sm:text-brand">
              Illarin
            </span>
          </Link>
        }
        end={
          <>
            <Button
              asChild
              variant="primary"
              size="compact"
              className="hidden md:inline-flex"
            >
              <Link
                href={publish.href}
                aria-current={pathname === "/upload" ? "page" : undefined}
              >
                {publish.label}
              </Link>
            </Button>
            <AppearanceMenu className="hidden md:inline-flex" />
            <AccountMenu />
          </>
        }
      />
    </motion.header>
  );
}
