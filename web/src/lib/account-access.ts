import type { SignedInAccount } from "./auth";

export type WayInId = "email" | "discord" | "password";

export type WayIn = {
  id: WayInId;
  name: string;
  standing: string;
  note: string;
  settled: boolean;
  canAttach?: boolean;
  canDetach?: boolean;
};

export function waysIn(account: SignedInAccount): WayIn[] {
  const emailSettled = account.emailVerified;
  const passwordSettled = account.hasPassword;

  return [
    {
      id: "email",
      name: "Email address",
      note: emailSettled
        ? "Verified and available for recovery."
        : "Verify your email before publishing or disconnecting Discord.",
      settled: emailSettled,
      standing: account.email ?? "No verified address yet",
    },
    {
      canAttach: emailSettled,
      canDetach: emailSettled && passwordSettled,
      id: "discord",
      name: "Discord",
      note: account.discordLinked
        ? "Sign in with your Discord account."
        : "Connect a Discord account to use it for sign-in.",
      settled: account.discordLinked,
      standing: account.discordLinked ? "Connected" : "Not connected",
    },
    {
      id: "password",
      name: "Password",
      note: passwordSettled
        ? "Use password recovery if you need to replace it."
        : "Sign in with email if you cannot access Discord.",
      settled: passwordSettled,
      standing: passwordSettled ? "Set" : "Not set",
    },
  ];
}
