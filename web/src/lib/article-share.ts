export type ShareState = "ready" | "copied" | "refused";

export type ShareReport = { said: string; reveal: boolean };

const REPORTS: Record<ShareState, ShareReport> = {
  ready: { said: "", reveal: false },
  copied: { said: "Link copied.", reveal: false },
  refused: {
    said: "Your browser would not let us copy. The link is here to take.",
    reveal: true,
  },
};

/** Whether the device carries its own share sheet, which is the only reason to offer one. */
export function hasNativeShare(agent: Navigator): boolean {
  return typeof agent.share === "function";
}

/** What a reader is told after asking for the link, and whether it is handed to them. */
export function shareReport(state: ShareState): ShareReport {
  return REPORTS[state];
}
