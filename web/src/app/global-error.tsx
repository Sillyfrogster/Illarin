"use client";

import { Button } from "@/components/ui/button";
import { DeadEnd } from "@/components/ui/dead-end";
import { FONT_VARIABLES } from "@/lib/fonts";
import "./globals.css";

/**
 * What is left when even the page frame failed, so it carries its own document.
 * The appearance script does not run here and the router is gone with the frame,
 * so it follows the system setting and leaves by a plain link.
 */
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
        <title>Illarin stopped short</title>
      </head>
      <body>
        <main className="flex min-h-svh flex-col justify-center">
          <DeadEnd
            heading="Illarin stopped short"
            line="The whole page stopped, not just the part you asked for. It is recorded, and trying again is often all it takes."
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
