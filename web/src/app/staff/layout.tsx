import type { ReactNode } from "react";
import { SiteProviders } from "@/app/(site)/site-providers";
import { StaffConsole } from "@/components/staff/StaffConsole";

export default function StaffLayout({ children }: { children: ReactNode }) {
  return (
    <SiteProviders>
      <StaffConsole>{children}</StaffConsole>
    </SiteProviders>
  );
}
