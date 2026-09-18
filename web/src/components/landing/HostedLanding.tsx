import type { ReactNode } from "react";
import { EncounterHero } from "./EncounterHero";
import { InvitationChapter } from "./InvitationChapter";
import { LandingMotion, MotionControl } from "./LandingMotion";
import { TypeShowcase } from "./TypeShowcase";

export function HostedLanding({ children }: { children: ReactNode }) {
  return (
    <LandingMotion>
      <div className="bg-field text-ink">
        <EncounterHero />
        <TypeShowcase />
        {children}
        <InvitationChapter />
        <MotionControl />
      </div>
    </LandingMotion>
  );
}
