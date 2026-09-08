"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ArrowUpRight, Menu, Moon, Sun, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "./button";

export function NotchNav({
  theme,
  onTheme,
  blog,
  href,
}: {
  theme: "light" | "dark";
  onTheme: () => void;
  blog: boolean;
  href: (surface: "blog" | "article" | "asset") => string;
}) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const reduced = useReducedMotion();
  useEffect(() => {
    if (!open) return;
    const dismiss = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) return;
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
    };
    document.addEventListener("keydown", dismiss);
    return () => document.removeEventListener("keydown", dismiss);
  }, [open]);
  const links = blog
    ? [
        { label: "Blog", href: href("blog") },
        { label: "Articles", href: `${href("blog")}&category=article` },
        { label: "Releases", href: `${href("blog")}&category=release` },
      ]
    : [
        { label: "Explore", href: "/browse" },
        { label: "Blog", href: href("blog") },
        { label: "Publish", href: "/upload" },
      ];
  return (
    <header className="vd:relative vd:z-40 vd:bg-plane vd:px-5 vd:pt-3 vd:md:px-10">
      <div className="vd:flex vd:min-h-16 vd:items-center vd:justify-between vd:gap-4">
        <nav
          aria-label="Main navigation"
          className="vd:hidden vd:flex-1 vd:gap-7 vd:md:flex"
        >
          {links.map((link) => (
            <a
              key={link.label}
              href={link.href}
              className="vd:group vd:relative vd:flex vd:min-h-11 vd:items-center vd:text-ui vd:font-medium vd:text-mute vd:hover:text-ink"
            >
              {link.label}
              <span className="vd:absolute vd:inset-x-0 vd:bottom-1 vd:h-0.5 vd:origin-left vd:scale-x-0 vd:bg-accent vd:transition-transform vd:duration-300 vd:group-hover:scale-x-100 vd:group-focus-visible:scale-x-100" />
            </a>
          ))}
        </nav>
        <Button
          ref={triggerRef}
          variant="ghost"
          size="icon"
          className="vd:md:hidden"
          aria-label={open ? "Close navigation" : "Open navigation"}
          aria-expanded={open}
          aria-controls="direction-nav"
          onClick={() => setOpen(!open)}
        >
          {open ? <X /> : <Menu />}
        </Button>
        <a
          href="/"
          aria-label="Illarin home"
          className="vd:flex vd:min-h-11 vd:items-center vd:gap-2 vd:font-display vd:text-[1.75rem] vd:font-medium vd:tracking-[-0.04em]"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 32 32"
            className="vd:size-7 vd:text-accent"
          >
            <path
              d="M16 2C16 12 12 16 2 16C12 16 16 20 16 30C16 20 20 16 30 16C20 16 16 12 16 2Z"
              fill="currentColor"
            />
          </svg>
          illarin<span className="vd:text-accent">.</span>
        </a>
        <div className="vd:flex vd:flex-1 vd:items-center vd:justify-end vd:gap-3">
          <Button
            size="icon"
            variant="ghost"
            aria-label={`Use ${theme === "dark" ? "light" : "dark"} theme`}
            onClick={onTheme}
          >
            {theme === "dark" ? <Sun /> : <Moon />}
          </Button>
          <Button
            asChild
            variant="secondary"
            className="vd:hidden vd:md:inline-flex"
          >
            <a href={blog ? "/browse" : "/sign-in"}>
              {blog ? "The collection" : "Sign in"}
              <ArrowUpRight />
            </a>
          </Button>
        </div>
      </div>
      <div
        aria-hidden="true"
        className="vd:pointer-events-none vd:absolute vd:top-full vd:left-1/2 vd:flex vd:h-5 vd:-translate-x-1/2 vd:text-plane"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 30 20"
          className="vd:h-5 vd:w-[30px]"
        >
          <path d="M0 0H30V20C14 20 16 0 0 0" fill="currentColor" />
        </svg>
        <div className="vd:w-40 vd:bg-plane" />
        <svg
          aria-hidden="true"
          viewBox="0 0 30 20"
          className="vd:h-5 vd:w-[30px]"
        >
          <path d="M0 0H30C14 0 16 20 0 20" fill="currentColor" />
        </svg>
      </div>
      <AnimatePresence>
        {open && (
          <motion.nav
            id="direction-nav"
            aria-label="Mobile navigation"
            initial={{ opacity: 0, y: reduced ? 0 : -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reduced ? 0 : 0.2 }}
            className="vd:absolute vd:inset-x-0 vd:top-full vd:z-50 vd:bg-plane vd:p-5 vd:shadow-lg vd:md:hidden"
          >
            {links.map((link) => (
              <a
                key={link.label}
                href={link.href}
                className="vd:flex vd:min-h-12 vd:items-center vd:justify-between vd:rounded-control vd:px-3 vd:hover:bg-deep"
              >
                {link.label}
                <ArrowUpRight className="vd:size-4" />
              </a>
            ))}
            <a
              href="/browse"
              className="vd:flex vd:min-h-12 vd:items-center vd:px-3"
            >
              The collection
            </a>
          </motion.nav>
        )}
      </AnimatePresence>
    </header>
  );
}
