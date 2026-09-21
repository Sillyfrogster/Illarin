import { cookies } from "next/headers";
import { AccountForm } from "@/components/auth/AccountForm";
import { AuthPage } from "@/components/auth/AuthPage";
import { fetchApps } from "@/lib/api/query";
import { safeInternalReturnPath } from "@/lib/internal-return";

export const metadata = { title: "Create an account" };

export default async function SignUpPage({
  searchParams,
}: {
  searchParams: Promise<{ returnTo?: string | string[] }>;
}) {
  const query = await searchParams;
  const returnTo = safeInternalReturnPath(query.returnTo);
  const apps = await fetchApps().catch(() => []);
  const remembered = (await cookies()).get("illarin_app")?.value ?? null;
  const known =
    remembered === "any" || apps.some((app) => app.id === remembered);

  return (
    <AuthPage
      returnTo={returnTo}
      switchTo={{ href: "/sign-in", label: "Sign in instead" }}
      title="Create an account"
    >
      <AccountForm
        apps={apps}
        initialApp={known ? remembered : null}
        mode="sign-up"
        returnTo={returnTo}
      />
    </AuthPage>
  );
}
