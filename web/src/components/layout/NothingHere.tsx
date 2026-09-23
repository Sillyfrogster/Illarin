import Link from "next/link";
import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";

export function NothingHere() {
  return (
    <DeadEnd
      heading="Page not found"
      line="Check the address, or browse to find what you were looking for."
    >
      <Button asChild variant="primary">
        <Link href="/browse">Browse</Link>
      </Button>
      <Button asChild variant="ghost">
        <Link href="/">Go to Illarin</Link>
      </Button>
    </DeadEnd>
  );
}
