"use client";

import { EyeOff } from "lucide-react";
import { useState } from "react";
import { Field } from "@/components/console/Field";
import { FormDialog } from "@/components/console/FormDialog";
import { withdrawPost } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import { AnnouncementChoice } from "./AnnouncementChoice";
import styles from "./TakeDown.module.css";

const SAID_LIMIT = 500;

export function TakeDown({
  post,
  onWithdrawn,
}: {
  post: Post;
  onWithdrawn: (post: Post) => void;
}) {
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [reason, setReason] = useState("");
  const [explanation, setExplanation] = useState("");
  const [sending, setSending] = useState<string[] | null>(null);
  const [note, setNote] = useState("");
  const [refusal, setRefusal] = useState("");

  function open() {
    setReason("");
    setExplanation("");
    setSending(null);
    setNote("");
    setRefusal("");
    setAsking(true);
  }

  async function commit() {
    setBusy(true);
    const answer = await withdrawPost(
      post.id,
      post.version,
      reason.trim(),
      explanation.trim(),
      { destinationIds: sending, note },
    );
    setBusy(false);
    if (answer.error || !answer.value) {
      setRefusal(answer.error ?? "The post is still on the blog.");
      return;
    }
    setAsking(false);
    onWithdrawn(answer.value);
  }

  return (
    <>
      <button className={styles.open} onClick={open} type="button">
        <EyeOff size={15} strokeWidth={1.8} aria-hidden="true" />
        Take it down
      </button>

      <FormDialog
        busy={busy}
        commit="Take it down"
        hint="Its editions, pictures and dates stay. You can put it back at the same address."
        onClose={() => setAsking(false)}
        onCommit={() => void commit()}
        open={asking}
        ready={reason.trim() !== ""}
        title="Take this post out of public view?"
      >
        <p className={styles.confirm}>
          {post.title}
          <span>illarin.xyz/blog/{post.slug}</span>
        </p>
        <Field
          hint="Illarin keeps this with the post. Readers never see it."
          htmlFor="withdrawal-reason"
          label="Why it is coming down"
        >
          <textarea
            id="withdrawal-reason"
            maxLength={SAID_LIMIT}
            onChange={(event) => setReason(event.target.value)}
            rows={3}
            value={reason}
          />
        </Field>
        <Field
          hint="Shown on the post's address. Leave it empty and readers get the general message alone."
          htmlFor="withdrawal-explanation"
          label="What readers see"
        >
          <textarea
            id="withdrawal-explanation"
            maxLength={SAID_LIMIT}
            onChange={(event) => setExplanation(event.target.value)}
            rows={3}
            value={explanation}
          />
        </Field>
        <AnnouncementChoice
          chosen={sending}
          note={note}
          onChosen={setSending}
          onNote={setNote}
          postId={post.id}
          transition="withdraw"
        />
        {refusal ? (
          <p className={styles.refusal} role="alert">
            {refusal}
          </p>
        ) : null}
      </FormDialog>
    </>
  );
}
