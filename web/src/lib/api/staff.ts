import { ask } from "./request";
import type { Report, ReportDay, ReportWork } from "./shapes";
export type { Report, ReportDay, ReportWork };

export const staffKeys = { report: ["staff", "report"] as const };

/** Reads the last 30 complete days of visits, downloads, sends, sign-ups and publishes. */
export async function readReport(signal?: AbortSignal): Promise<Report> {
  const answer = await ask<Report>("GET", "/staff/report", { signal });
  if (answer.value) return answer.value;
  throw new Error(answer.error ?? "The report could not be loaded.");
}
