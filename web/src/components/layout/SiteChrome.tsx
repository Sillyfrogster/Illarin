import type { ReactNode } from "react";
import { SiteFooter } from "./SiteFooter";
import { SiteHeader } from "./SiteHeader";

export function SiteChrome({ children }: { children: ReactNode }) {
  return (
    <>
      <a
        className="sr-only rounded-control bg-plane p-4 text-ui text-ink shadow-popover focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-90"
        href="#main-content"
      >
        Skip to content
      </a>
      <SiteHeader />
      <main id="main-content">{children}</main>
      <SiteFooter />
    </>
  );
}
