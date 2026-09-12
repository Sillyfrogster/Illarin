"use client";

import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { FONT_VARIABLES } from "@/lib/fonts";
import "./globals.css";

export default function GlobalError({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  return (
    <html className={FONT_VARIABLES} lang="en" suppressHydrationWarning>
      <head>
        <title>This page could not load</title>
      </head>
      <body>
        <main className="flex min-h-svh flex-col justify-center">
          <DeadEnd
            heading="This page could not load"
            line="Illarin could not load this page. Try again."
            note={error.digest ? `Reference ${error.digest}` : undefined}
          >
            <Button onClick={() => retry()} variant="primary">
              Try again
            </Button>
            <Button asChild variant="ghost">
              <a href="/">Go to Illarin</a>
            </Button>
          </DeadEnd>
        </main>
      </body>
    </html>
  );
}
