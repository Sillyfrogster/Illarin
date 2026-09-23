import { Suspense } from "react";
import { ConnectionApproval } from "@/components/connect/ConnectionApproval";
import { Shell } from "@/components/layout/Shell";
import { Waiting } from "@/components/ui/waiting";
import { pageMetadata } from "@/lib/site-metadata";

export const metadata = pageMetadata(
  "Connect an app",
  "Review what an app is asking for before it reaches your account.",
);

export default function ConnectPage() {
  return (
    <div className="relative isolate overflow-hidden">
      <Shell className="flex min-h-[calc(100svh-var(--header-height))] flex-col justify-center pt-10 pb-16 lg:pt-16">
        <div className="mx-auto w-full max-w-[42rem]">
          <Suspense fallback={<Waiting>Opening…</Waiting>}>
            <ConnectionApproval />
          </Suspense>
          <p className="mt-12 border-t border-rule pt-5 font-ui text-meta text-mute">
            You can revoke a connected app at any time from account settings.
          </p>
        </div>
      </Shell>
    </div>
  );
}
