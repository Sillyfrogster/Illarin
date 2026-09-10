import { NothingHere } from "@/components/layout/NothingHere";
import { SiteChrome } from "@/components/layout/SiteChrome";

/** The same answer for an address no route matches at all, which stands outside the site layout. */
export default function NotFound() {
  return (
    <SiteChrome>
      <NothingHere />
    </SiteChrome>
  );
}
