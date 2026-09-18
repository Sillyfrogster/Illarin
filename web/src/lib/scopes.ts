import type { Scope } from "@/lib/api/shapes";
export type { Scope };

export type ScopeCopy = {
  title: string;
  detail: string;
};

const SCOPES: Record<Scope, ScopeCopy> = {
  "asset:receive": {
    title: "Receive works you send it",
    detail:
      "You choose what goes across. It cannot browse or take anything on its own.",
  },
  "library:sync": {
    title: "Report what it has installed",
    detail:
      "So Illarin can show what you already have, and when a newer version exists.",
  },
};

export function describeScope(scope: Scope): ScopeCopy {
  return (
    SCOPES[scope] ?? {
      title: scope,
      detail: "This version of Illarin does not recognise this permission.",
    }
  );
}
