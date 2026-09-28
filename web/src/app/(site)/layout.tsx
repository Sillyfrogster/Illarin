import { SiteChrome } from "@/components/layout/SiteChrome";
import { VisitBeacon } from "@/components/layout/VisitBeacon";
import { SiteProviders } from "./site-providers";

export default function SiteLayout({ children }: LayoutProps<"/">) {
  return (
    <SiteProviders>
      <VisitBeacon />
      <SiteChrome>{children}</SiteChrome>
    </SiteProviders>
  );
}
