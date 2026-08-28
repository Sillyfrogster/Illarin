"use client";

import { Plus } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import rows from "@/components/console/Console.module.css";
import { readTokens } from "@/lib/api/publication";
import type { PublicationGrant, PublicationToken } from "@/lib/api/query";
import styles from "./GrantTokens.module.css";
import { TokenDialog } from "./TokenDialog";
import { TokenRows } from "./TokenRows";

export function GrantTokens({ grant }: { grant: PublicationGrant }) {
  const [tokens, setTokens] = useState<PublicationToken[] | null>(null);
  const [failure, setFailure] = useState("");
  const [making, setMaking] = useState(false);

  const load = useCallback(async () => {
    const answer = await readTokens(grant.id);
    if (answer.error || !answer.value) {
      setFailure(answer.error ?? "");
      return;
    }
    setFailure("");
    setTokens(answer.value.tokens);
  }, [grant.id]);

  useEffect(() => {
    void load();
  }, [load]);

  const live = tokens?.filter((token) => token.active) ?? [];
  const spent = tokens?.filter((token) => !token.active) ?? [];

  return (
    <div className={styles.tokens}>
      <header className={styles.heading}>
        <h3>Tokens for the publication API</h3>
        {tokens ? <span className={rows.count}>{live.length}</span> : null}
        <button
          type="button"
          className={styles.add}
          onClick={() => setMaking(true)}
        >
          <Plus size={15} strokeWidth={2} aria-hidden="true" />
          New token
        </button>
      </header>

      {failure ? (
        <p className={rows.failure} role="alert">
          {failure}
        </p>
      ) : null}

      {tokens === null ? (
        <p className={styles.quiet} aria-live="polite">
          Reading your tokens…
        </p>
      ) : live.length === 0 ? (
        <p className={styles.quiet}>
          Nothing carries your approval yet. A token lets a script or an editor
          of your own publish for {grant.app.name} without your password.
        </p>
      ) : (
        <TokenRows
          tokens={live}
          revocable
          onRevoked={() => void load()}
          onFailure={setFailure}
        />
      )}

      {spent.length > 0 ? (
        <details className={styles.spent}>
          <summary>{spent.length} no longer works</summary>
          <TokenRows
            tokens={spent}
            revocable={false}
            onRevoked={() => void load()}
            onFailure={setFailure}
          />
        </details>
      ) : null}

      {making ? (
        <TokenDialog
          grantId={grant.id}
          appName={grant.app.name}
          onClose={() => {
            setMaking(false);
            void load();
          }}
          onIssued={() => void load()}
          onFailure={setFailure}
        />
      ) : null}
    </div>
  );
}
