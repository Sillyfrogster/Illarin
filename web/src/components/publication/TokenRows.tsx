"use client";

import { useState } from "react";
import { revokeToken } from "@/lib/api/publication";
import type { PublicationToken } from "@/lib/api/query";
import { tokenEnded, tokenStanding } from "@/lib/publication-register";

export function TokenRows({
  onFailure,
  onRevoked,
  revocable,
  tokens,
}: {
  onFailure: (message: string) => void;
  onRevoked: (id: string) => void;
  revocable: boolean;
  tokens: PublicationToken[];
}) {
  const [confirming, setConfirming] = useState("");
  const [revoking, setRevoking] = useState("");

  async function revoke(id: string) {
    setRevoking(id);
    const answer = await revokeToken(id);
    setRevoking("");
    if (answer.error) {
      onFailure(answer.error);
      return;
    }
    setConfirming("");
    onRevoked(id);
  }

  return (
    <ol className="flex list-none flex-col">
      {tokens.map((token) => {
        const asking = confirming === token.id;
        return (
          <li
            className="flex flex-col gap-1 border-rule/45 py-3 not-first:border-t"
            key={token.id}
          >
            <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
              <span className="min-w-0 font-ui text-ui text-ink wrap-anywhere">
                {token.name}{" "}
                <span className="font-mono text-label text-mute">
                  {token.prefix}
                </span>
              </span>
              {token.active && revocable ? (
                <span className="flex shrink-0 items-center gap-1">
                  {asking ? (
                    <button
                      className="inline-flex min-h-11 items-center rounded-control px-3 font-ui text-meta font-medium text-mute outline-offset-3 hover:text-ink"
                      onClick={() => setConfirming("")}
                      type="button"
                    >
                      Cancel
                    </button>
                  ) : null}
                  <button
                    aria-describedby={
                      asking ? `${token.id}-consequence` : undefined
                    }
                    aria-expanded={asking}
                    className="inline-flex min-h-11 items-center rounded-control bg-stop-wash px-3 font-ui text-meta font-medium text-stop outline-offset-3 hover:opacity-85 disabled:opacity-45"
                    disabled={Boolean(revoking)}
                    onClick={() =>
                      asking ? revoke(token.id) : setConfirming(token.id)
                    }
                    type="button"
                  >
                    {revoking === token.id
                      ? "Revoking…"
                      : asking
                        ? "Revoke token"
                        : "Revoke"}
                  </button>
                </span>
              ) : (
                <span className="shrink-0 font-prose text-meta text-mute">
                  {tokenEnded(token)}
                </span>
              )}
            </div>
            <span className="font-prose text-meta text-mute">
              {tokenStanding(token)}
            </span>
            {asking ? (
              <p
                className="max-w-[52ch] font-prose text-meta text-stop"
                id={`${token.id}-consequence`}
              >
                Tools using this token will immediately lose publishing access.
                Other tokens will keep working.
              </p>
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}
