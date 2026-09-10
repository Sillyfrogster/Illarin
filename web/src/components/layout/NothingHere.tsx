import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";

/** The one answer Illarin gives for an address it holds nothing at, whatever the reason. */
export function NothingHere() {
  return (
    <DeadEnd
      heading="Nothing is here"
      line="Illarin answers the same way for work that was taken down, work its creator made private, and an address that never existed."
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
