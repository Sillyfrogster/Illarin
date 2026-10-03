import type { ViewTransitionProps } from "react";

/** workCoverTransition names a work's cover so it morphs between Browse's open panel and the card on its page. */
export function workCoverTransition(id: string): ViewTransitionProps {
  return { name: `work-cover-${id}`, share: "morph", default: "none" };
}

/** holdScroll records how far the page is scrolled as a link leaves it, so the leaving page's picture stays in place during the move. */
export function holdScroll() {
  document.documentElement.style.setProperty(
    "--leave-scroll",
    `${window.scrollY}px`,
  );
}
