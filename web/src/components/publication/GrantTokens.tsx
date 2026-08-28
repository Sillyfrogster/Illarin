"use client";

import { Plus } from "lucide-react";
import { useState } from "react";
import rows from "@/components/console/Console.module.css";
import type { PublicationGrant } from "@/lib/api/query";
import styles from "./GrantTokens.module.css";
import { TokenDialog } from "./TokenDialog";
import { TokenRows } from "./TokenRows";
import { useGrantTokens } from "./use-grant-tokens";

export function GrantTokens({ grant }: { grant: PublicationGrant }) {
  const [failure, setFailure] = useState("");
  const [making, setMaking] = useState(false);
  const { tokens, live, spent, reread } = useGrantTokens(grant.id, setFailure);

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
          onRevoked={() => reread()}
          onFailure={setFailure}
        />
      )}

      {spent.length > 0 ? (
        <details className={styles.spent}>
          <summary>{spent.length} no longer works</summary>
          <TokenRows
            tokens={spent}
            revocable={false}
            onRevoked={() => reread()}
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
            reread();
          }}
          onIssued={() => reread()}
          onFailure={setFailure}
        />
      ) : null}
    </div>
  );
}
