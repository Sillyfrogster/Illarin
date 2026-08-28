"use client";

import { useState } from "react";
import { revokeToken } from "@/lib/api/publication";
import type { PublicationToken } from "@/lib/api/query";
import { readableDate } from "@/lib/dates";
import styles from "./TokenRows.module.css";

export function TokenRows({
  tokens,
  revocable,
  onRevoked,
  onFailure,
}: {
  tokens: PublicationToken[];
  revocable: boolean;
  onRevoked: (id: string) => void;
  onFailure: (message: string) => void;
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
    <ol className={styles.tokens}>
      {tokens.map((token) => {
        const asking = confirming === token.id;
        return (
          <li
            className={styles.token}
            data-spent={token.active ? undefined : "true"}
            key={token.id}
          >
            <span className={styles.name}>
              {token.name} <span className={styles.prefix}>{token.prefix}</span>
            </span>
            <span className={styles.detail}>{describe(token)}</span>
            <span className={styles.actions}>
              {token.active && revocable ? (
                <>
                  {asking ? (
                    <button
                      type="button"
                      className={styles.stand}
                      onClick={() => setConfirming("")}
                    >
                      Keep it
                    </button>
                  ) : null}
                  <button
                    type="button"
                    className={asking ? styles.confirm : styles.revoke}
                    disabled={Boolean(revoking)}
                    aria-expanded={asking}
                    aria-describedby={
                      asking ? `${token.id}-consequence` : undefined
                    }
                    onClick={() =>
                      asking ? revoke(token.id) : setConfirming(token.id)
                    }
                  >
                    {revoking === token.id
                      ? "Revoking…"
                      : asking
                        ? "Revoke for good"
                        : "Revoke"}
                  </button>
                </>
              ) : (
                <span className={styles.spent}>{ended(token)}</span>
              )}
            </span>
            {asking ? (
              <p className={styles.consequence} id={`${token.id}-consequence`}>
                Anything carrying it stops publishing at once. Your other tokens
                and your approval are untouched.
              </p>
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}

function describe(token: PublicationToken) {
  const said = [`Made ${readableDate(token.createdAt)}`];
  said.push(
    token.lastUsedAt
      ? `last used ${readableDate(token.lastUsedAt)}`
      : "never used",
  );
  if (token.expiresAt && token.active) {
    said.push(`expires ${readableDate(token.expiresAt)}`);
  }
  return said.join(" · ");
}

function ended(token: PublicationToken) {
  if (token.revokedAt) return `Revoked ${readableDate(token.revokedAt)}`;
  if (token.expiresAt) return `Expired ${readableDate(token.expiresAt)}`;
  return "Spent";
}
