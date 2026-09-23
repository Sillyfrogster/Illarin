import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { api } from "@/lib/api/client";
import type { SessionState } from "@/lib/api/shapes";

/** The profile is edited on the profile page itself, so this old address sends the owner there. */
export default async function PublicProfileSettings() {
  const { data: session } = await api<SessionState>("GET", "/v1/auth/session", {
    headers: { cookie: (await cookies()).toString() },
    cache: "no-store",
  });
  redirect(
    session?.user
      ? `/@${session.user.handle}`
      : `/sign-in?returnTo=${encodeURIComponent("/settings/profile")}`,
  );
}
