import { NothingHere } from "@/components/layout/NothingHere";
import { SiteChrome } from "@/components/layout/SiteChrome";
import { SiteProviders } from "./(site)/site-providers";

export default function NotFound() {
  return (
    <SiteProviders>
      <SiteChrome>
        <NothingHere />
      </SiteChrome>
    </SiteProviders>
  );
}
