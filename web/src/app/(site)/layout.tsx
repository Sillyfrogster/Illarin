import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";

export default function SiteLayout({ children }: LayoutProps<"/">) {
  return (
    <>
      <a
        href="#main-content"
        className="sr-only rounded-control bg-plane p-4 text-ui text-ink shadow-popover focus:not-sr-only focus:absolute focus:top-3 focus:left-3 focus:z-90"
      >
        Skip to content
      </a>
      <SiteHeader />
      <main id="main-content">{children}</main>
      <SiteFooter />
    </>
  );
}
