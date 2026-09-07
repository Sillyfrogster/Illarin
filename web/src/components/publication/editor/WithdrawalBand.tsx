"use client";

import { EyeOff } from "lucide-react";
import { useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import { readPostRevisions, republishPost } from "@/lib/api/posts";
import type { Post, PostRevision } from "@/lib/api/query";
import { readableMoment } from "@/lib/dates";
import { revisionWords } from "@/lib/post-history";
import { AnnouncementChoice } from "./AnnouncementChoice";
import styles from "./WithdrawalBand.module.css";

export function WithdrawalBand({
  post,
  onChanged,
  onFailure,
}: {
  post: Post;
  onChanged: (post: Post) => void;
  onFailure: (message: string) => void;
}) {
  const withdrawal = post.withdrawal;
  const [kept, setKept] = useState<PostRevision[] | null>(null);
  const [putting, setPutting] = useState(false);
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState("");
  const [sending, setSending] = useState<string[] | null>(null);
  const [note, setNote] = useState("");

  if (post.status !== "withdrawn" || !withdrawal) return null;

  async function open() {
    setChosen(post.publicRevisionId ?? "");
    setSending(null);
    setNote("");
    setPutting(true);
    if (kept) return;
    const answer = await readPostRevisions(post.id);
    if (answer.value) setKept(answer.value.revisions);
  }

  async function putBack() {
    if (!chosen) return;
    setBusy(true);
    const answer = await republishPost(post.id, post.version, chosen, {
      destinationIds: sending,
      note,
    });
    setBusy(false);
    setPutting(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "");
      return;
    }
    onChanged(answer.value);
  }

  return (
    <section aria-labelledby="out-of-view" className={styles.band}>
      <p className={styles.mark}>
        <EyeOff size={15} strokeWidth={1.9} aria-hidden="true" />
        <span id="out-of-view">Out of public view</span>
      </p>

      <p className={styles.said}>
        Taken down {readableMoment(withdrawal.at)} by @{withdrawal.by}
      </p>

      <dl className={styles.wording}>
        <dt>What readers see</dt>
        <dd>
          {withdrawal.explanation ||
            "Only the general message. You wrote nothing for them."}
        </dd>
        <dt>Illarin's record</dt>
        <dd>{withdrawal.reason}</dd>
      </dl>

      <div className={styles.actions}>
        <button
          className={styles.act}
          onClick={() => void open()}
          type="button"
        >
          Put it back
        </button>
      </div>

      <FormDialog
        busy={busy}
        commit="Put it back"
        hint="It returns to the same address under the date it first published."
        onClose={() => setPutting(false)}
        onCommit={() => void putBack()}
        open={putting}
        ready={chosen !== ""}
        title="Which edition do readers get?"
      >
        <fieldset className={styles.editions}>
          <legend>Edition</legend>
          {(kept ?? []).map((one) => (
            <label className={styles.edition} key={one.id}>
              <input
                checked={chosen === one.id}
                name="returning-edition"
                onChange={() => setChosen(one.id)}
                type="radio"
                value={one.id}
              />
              <span className={styles.number}>{one.number}</span>
              <span className={styles.editionName}>
                {revisionWords(one.capturedFor)}
                <span>{readableMoment(one.capturedAt)}</span>
              </span>
              {one.public ? (
                <span className={styles.standing}>Was on the blog</span>
              ) : null}
            </label>
          ))}
        </fieldset>
        <AnnouncementChoice
          announced
          chosen={sending}
          note={note}
          onChosen={setSending}
          onNote={setNote}
          onPinging={() => undefined}
          pinging={[]}
          postId={post.id}
          transition="republish"
        />
      </FormDialog>
    </section>
  );
}
