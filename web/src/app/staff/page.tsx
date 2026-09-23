import type { Metadata } from "next";
import { StaffReport } from "@/components/staff/StaffReport";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata: Metadata = {
  ...pageMetadata(
    "Staff report",
    "Visits, downloads, sends, sign-ups and publishes by day.",
  ),
  robots: { index: false, follow: false },
};

export default function StaffReportPage() {
  return <StaffReport />;
}
