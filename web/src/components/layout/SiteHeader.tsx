"use client";

import {
  motion,
  useMotionTemplate,
  useScroll,
  useTransform,
} from "framer-motion";
import { Plus } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandLogo } from "@/components/brand/BrandLogo";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { Button } from "@/components/ui/button";
import { LineLink } from "@/components/ui/line-link";
import { useAuth } from "@/lib/auth";
import { useOrigins } from "@/lib/origins";
import { AccountMenu } from "./AccountMenu";
import {
  isCurrentPage,
  primaryDestinations,
  publishAction,
} from "./destinations";
import { MobileNav } from "./MobileNav";
import { Notch } from "./Notch";

export function SiteHeader() {
  const pathname = usePathname();
  const { account } = useAuth();
  const { blog } = useOrigins();
  const publish = publishAction(account);
  const { scrollY } = useScroll();
  const depth = useTransform(scrollY, [0, 40], [0.16, 0.36], { clamp: true });
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
              className="hidden items-center gap-2 md:flex"
              aria-label="Primary"
            >
              {primaryDestinations(blog).map((item) => (
                <LineLink
                  key={item.href}
                  href={item.href}
                  current={isCurrentPage(pathname, item.href)}
                  className="rounded-control px-3 text-ink hover:bg-deep aria-[current=page]:bg-deep"
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
            <BrandLogo className="sm:w-36" />
          </Link>
        }
        end={
          <>
            <Button asChild variant="primary" className="hidden md:inline-flex">
              <Link
                href={publish.href}
                aria-current={pathname === "/upload" ? "page" : undefined}
              >
                <Plus aria-hidden="true" />
                {publish.label}
              </Link>
            </Button>
            <div className="flex items-center gap-1">
              <NotificationBell />
              <AccountMenu />
            </div>
          </>
        }
      />
    </motion.header>
  );
}
