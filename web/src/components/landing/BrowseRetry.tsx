"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { Button } from "@/components/ui/button";

export function BrowseRetry() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  return (
    <Button
      loading={pending}
      onClick={() => startTransition(() => router.refresh())}
    >
      Try again
    </Button>
  );
}
