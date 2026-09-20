"use client";

import { motion, useReducedMotion } from "framer-motion";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/field";
import { Select } from "@/components/ui/select";
import { startPost } from "@/lib/api/posts";
import type { BlogWorkspace } from "@/lib/api/query";

export function StartPost({
  onFailure,
  workspace,
}: {
  onFailure: (message: string) => void;
  workspace: BlogWorkspace;
}) {
  const router = useRouter();
  const reduced = useReducedMotion();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [title, setTitle] = useState("");
  const [categoryId, setCategoryId] = useState(
    workspace.categories[0]?.id ?? "",
  );
  const naming = useRef<HTMLInputElement>(null);
  const ready = title.trim().length > 0 && categoryId !== "";

  useEffect(() => {
    if (open) naming.current?.focus();
  }, [open]);

  async function start(event: FormEvent) {
    event.preventDefault();
    if (!ready || busy) return;
    setBusy(true);
    const answer = await startPost({ categoryId, title: title.trim() });
    setBusy(false);
    if (answer.error || !answer.value) {
      onFailure(answer.error ?? "The post could not be started.");
      return;
    }
    router.push(`/posts/${answer.value.id}`);
  }

  if (!open) {
    return (
      <Button onClick={() => setOpen(true)} variant="primary">
        <Plus aria-hidden="true" />
        New post
      </Button>
    );
  }

  return (
    <motion.form
      animate={{ opacity: 1, y: 0 }}
      className="w-full rounded-plate bg-deep p-5 sm:p-6"
      initial={reduced ? { opacity: 0 } : { opacity: 0, y: -8 }}
      onSubmit={start}
      transition={{ duration: reduced ? 0 : 0.28, ease: [0.22, 1, 0.36, 1] }}
    >
      <h2 className="font-display text-section font-medium tracking-tight text-ink">
        New post
      </h2>
      <p className="mt-1 font-prose text-meta text-mute">
        You can change all of this while you write.
      </p>
      <div className="mt-5 grid gap-5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start">
        <Field htmlFor="new-post-title" label="Title">
          <TextInput
            className="bg-field"
            id="new-post-title"
            maxLength={160}
            onChange={(event) => setTitle(event.target.value)}
            ref={naming}
            value={title}
          />
        </Field>
        <Field className="sm:w-56" htmlFor="new-post-category" label="Category">
          <Select
            className="bg-field"
            id="new-post-category"
            onChange={(event) => setCategoryId(event.target.value)}
            value={categoryId}
          >
            {workspace.categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.label}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <div className="mt-6 flex flex-wrap items-center gap-2">
        <Button
          disabled={!ready}
          loading={busy}
          type="submit"
          variant="primary"
        >
          Start writing
        </Button>
        <Button disabled={busy} onClick={() => setOpen(false)} variant="ghost">
          Cancel
        </Button>
      </div>
    </motion.form>
  );
}
