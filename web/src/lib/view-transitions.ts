import type { ViewTransitionProps } from "react";

/** workCoverTransition names a work's cover so it morphs between Browse's open panel and the card on its page. */
export function workCoverTransition(id: string): ViewTransitionProps {
  return { name: `work-cover-${id}`, share: "morph", default: "none" };
}
