import { Suspense } from "react";
import { AuthPage } from "@/components/auth/AuthPage";
import { VerificationPanel } from "@/components/auth/VerificationPanel";
import { Waiting } from "@/components/ui/waiting";

export const metadata = { title: "Verify your email" };

export default function VerifyEmailPage() {
  return (
    <AuthPage
      title="Verify your email"
      introduction="Verification is required before publishing or linking an application. You can keep browsing while the account is unverified."
    >
      <Suspense fallback={<Waiting>Opening your verification link…</Waiting>}>
        <VerificationPanel />
      </Suspense>
    </AuthPage>
  );
}
