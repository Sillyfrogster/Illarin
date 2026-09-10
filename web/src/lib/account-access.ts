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
        : "Verify an address before publishing or detaching Discord.",
      settled: emailSettled,
      standing: account.email ?? "No verified address yet",
    },
    {
      canAttach: emailSettled,
      canDetach: emailSettled && passwordSettled,
      id: "discord",
      name: "Discord",
      note: account.discordLinked
        ? "Return without entering a password."
        : "Attach Discord without combining this account with another.",
      settled: account.discordLinked,
      standing: account.discordLinked ? "Attached" : "Not attached",
    },
    {
      id: "password",
      name: "Password",
      note: passwordSettled
        ? "Use password recovery if you need to replace it."
        : "An independent way back if Discord is unavailable.",
      settled: passwordSettled,
      standing: passwordSettled ? "Set" : "Not set",
    },
  ];
}
