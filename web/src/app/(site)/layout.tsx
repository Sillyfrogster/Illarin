import { SiteChrome } from "@/components/layout/SiteChrome";
import { SiteProviders } from "./site-providers";

export default function SiteLayout({ children }: LayoutProps<"/">) {
  return (
    <SiteProviders>
      <SiteChrome>{children}</SiteChrome>
    </SiteProviders>
  );
}
