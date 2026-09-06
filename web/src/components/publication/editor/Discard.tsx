"use client";

import { Trash2 } from "lucide-react";
import { useState } from "react";
import { FormDialog } from "@/components/console/FormDialog";
import { deletePost } from "@/lib/api/posts";
import type { Post } from "@/lib/api/query";
import styles from "./Discard.module.css";

export function Discard({
  post,
  onDeleted,
}: {
  post: Post;
  onDeleted: (post: Post) => void;
}) {
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [refusal, setRefusal] = useState("");

  function open() {
    setRefusal("");
    setAsking(true);
  }

  async function commit() {
    setBusy(true);
    const answer = await deletePost(post.id, post.version);
    setBusy(false);
    if (answer.error || !answer.value) {
      setRefusal(answer.error ?? "The post is still here.");
      return;
    }
    setAsking(false);
    onDeleted(answer.value);
  }

  return (
    <>
      <button className={styles.open} onClick={open} type="button">
        <Trash2 size={15} strokeWidth={1.8} aria-hidden="true" />
        Delete
      </button>

      <FormDialog
        busy={busy}
        commit="Delete it"
        critical
        hint="Illarin holds it for thirty days. After that the writing, the editions and the pictures are gone."
        onClose={() => setAsking(false)}
        onCommit={() => void commit()}
        open={asking}
        ready
        title="Delete this post?"
      >
        <p className={styles.confirm}>
          {post.title}
          <span>
            {post.publishedAt
              ? `Published, then taken down`
              : "Never published"}
          </span>
        </p>
        {refusal ? (
          <p className={styles.refusal} role="alert">
            {refusal}
          </p>
        ) : null}
      </FormDialog>
    </>
  );
}
