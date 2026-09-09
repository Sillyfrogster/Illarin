import type { ReactNode } from "react";
import { EncounterHero } from "./EncounterHero";
import { InvitationChapter } from "./InvitationChapter";
import { KindShowcase } from "./KindShowcase";
import { LandingMotion, MotionControl } from "./LandingMotion";

export function HostedLanding({ children }: { children: ReactNode }) {
  return (
    <LandingMotion>
      <div className="bg-field text-ink">
        <EncounterHero />
        <KindShowcase />
        {children}
        <InvitationChapter />
        <MotionControl />
      </div>
    </LandingMotion>
  );
}
