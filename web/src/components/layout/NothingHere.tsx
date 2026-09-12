import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";

export function NothingHere() {
  return (
    <DeadEnd
      heading="Page not found"
      line="Check the address, or browse the catalog to find an asset."
    >
      <Button asChild variant="primary">
        <Link href="/browse">Browse the catalog</Link>
      </Button>
      <Button asChild variant="ghost">
        <Link href="/">Go to Illarin</Link>
      </Button>
    </DeadEnd>
  );
}
