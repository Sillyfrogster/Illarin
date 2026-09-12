export type ShareState = "ready" | "copied" | "refused";

export type ShareReport = { said: string; reveal: boolean };

const REPORTS: Record<ShareState, ShareReport> = {
  ready: { said: "", reveal: false },
  copied: { said: "Link copied.", reveal: false },
  refused: {
    said: "The link could not be copied. Select it below and copy it manually.",
    reveal: true,
  },
};

export function hasNativeShare(agent: Navigator): boolean {
  return typeof agent.share === "function";
}

export function shareReport(state: ShareState): ShareReport {
  return REPORTS[state];
}
