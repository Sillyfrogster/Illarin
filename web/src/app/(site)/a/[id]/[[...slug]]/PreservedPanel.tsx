"use client";

import { ChevronRight, RotateCcw, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  deletePreservedNamespace,
  fetchPreservedNamespaces,
  type PreservedNamespace,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { describePreservedNamespace } from "@/lib/preserved";
import { useWorkingCopy } from "@/lib/working-copy";

export function PreservedPanel({ assetId }: { assetId: string }) {
  const candidate = useWorkingCopy();
  const [open, setOpen] = useState(false);
  const [namespaces, setNamespaces] = useState<PreservedNamespace[] | null>(
    null,
  );
  const [deleting, setDeleting] = useState<PreservedNamespace | null>(null);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function openMenu() {
    setOpen(true);
    if (namespaces !== null) return;
    setMessage("");
    const found = await fetchPreservedNamespaces(assetId);
    setNamespaces(found);
  }

  async function remove(namespace: string) {
    if (pending) return;
    setPending(true);
    setMessage("");
    try {
      await deletePreservedNamespace(candidate, assetId, namespace);
      setNamespaces(
        (current) =>
          current?.filter((held) => held.name !== namespace) ?? current,
      );
      setDeleting(null);
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "That data could not be deleted. Try again.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <div className={"mt-3.5 border-rule border-t"}>
      <button
        className={
          "flex min-h-16 w-full items-center justify-between gap-3.5 py-3 text-left text-ink outline-offset-3 hover:text-accent"
        }
        type="button"
        aria-expanded={open}
        aria-controls="preserved-menu"
        onClick={() => {
          if (open) {
            setOpen(false);
          } else {
            void openMenu();
          }
        }}
      >
        <span
          className={
            "grid gap-1 [&>span]:text-meta [&>span]:text-mute [&>strong]:text-ui [&>strong]:font-medium"
          }
        >
          <strong>Manage file extras</strong>
          <span>Review what your upload kept for compatible downloads</span>
        </span>
        <ChevronRight
          className={cn(
            "shrink-0 text-mute transition-transform duration-200 motion-reduce:transition-none",
            open && "rotate-90",
          )}
          size={18}
          aria-hidden="true"
        />
      </button>

      {open ? (
        <div className={"pb-3.5"} id="preserved-menu">
          <p className={"text-meta text-mute"}>
            These details came with the original file but are not part of the
            editable page. Removing one stops it travelling in downloads for
            that format.
          </p>
          {namespaces === null ? (
            <p className={"mt-3 text-meta text-mute italic"}>
              Reading the file extras…
            </p>
          ) : namespaces.length === 0 ? (
            <p className={"mt-3 text-meta text-mute italic"}>
              Nothing extra is being kept with this upload.
            </p>
          ) : (
            <ul
              className={
                "mt-3 list-none border-rule border-t [&>li]:flex [&>li]:min-h-13 [&>li]:items-center [&>li]:justify-between [&>li]:gap-3 [&>li]:border-rule [&>li]:border-b [&>li]:text-ui"
              }
            >
              {namespaces.map((namespace) => (
                <li key={namespace.name}>
                  <span>{describePreservedNamespace(namespace.name)}</span>
                  <button
                    className={
                      "flex size-11 shrink-0 items-center justify-center rounded-control text-mute outline-offset-3 hover:bg-deep hover:text-stop"
                    }
                    type="button"
                    onClick={() => {
                      setMessage("");
                      setDeleting(namespace);
                    }}
                  >
                    <Trash2 size={15} aria-hidden="true" />
                    <span className="sr-only">
                      Remove {describePreservedNamespace(namespace.name)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
          {message ? (
            <p className={"mt-3 text-meta text-stop"} role="alert">
              {message}
            </p>
          ) : null}
          {namespaces === null ? (
            <button
              className={
                "mt-3 inline-flex min-h-11 items-center gap-2 rounded-control bg-deep px-3 text-meta font-medium text-ink outline-offset-3"
              }
              type="button"
              onClick={() => {
                setNamespaces(null);
                void openMenu();
              }}
            >
              <RotateCcw size={14} aria-hidden="true" />
              Try again
            </button>
          ) : null}
        </div>
      ) : null}

      {deleting ? (
        <DeleteNamespaceDialog
          namespace={deleting}
          pending={pending}
          onCancel={() => setDeleting(null)}
          onDelete={() => void remove(deleting.name)}
        />
      ) : null}
    </div>
  );
}

function DeleteNamespaceDialog({
  namespace,
  pending,
  onCancel,
  onDelete,
}: {
  namespace: PreservedNamespace;
  pending: boolean;
  onCancel: () => void;
  onDelete: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const description = describePreservedNamespace(namespace.name);

  useEffect(() => dialog.current?.showModal(), []);

  return (
    <dialog
      ref={dialog}
      className={
        "m-auto w-[min(30rem,calc(100vw-2rem))] rounded-plate bg-plane p-0 text-ink backdrop:bg-black/60"
      }
      onCancel={(event) => {
        event.preventDefault();
        onCancel();
      }}
    >
      <div
        className={
          "p-5 [&_h2]:mt-1 [&_h2]:font-display [&_h2]:text-section [&_h2]:font-medium [&_h2]:text-ink [&_p]:mt-2 [&_p]:text-ui [&_p]:text-mute"
        }
      >
        <p className={"text-meta text-mute"}>Remove file extras</p>
        <h2>Remove {description}?</h2>
        <p>
          This detail came with your original file. Removing it means it will
          stop travelling in downloads made for that format.
        </p>
        <p className={"mt-2 text-meta text-mute"}>
          This cannot be undone here. Re-upload the original file if you need it
          back.
        </p>
      </div>
      <footer
        className={
          "flex flex-wrap justify-end gap-2 border-rule border-t p-4 [&>button]:min-h-11 [&>button]:rounded-control [&>button]:px-4 [&>button]:text-ui [&>button]:font-medium [&>button]:outline-offset-3 [&>button]:disabled:opacity-45"
        }
      >
        <button type="button" onClick={onCancel} disabled={pending}>
          Keep it
        </button>
        <button
          type="button"
          className={
            "inline-flex min-h-11 items-center gap-2 rounded-control bg-stop px-4 text-ui font-medium text-on-stop outline-offset-3 disabled:opacity-45"
          }
          onClick={onDelete}
          disabled={pending}
        >
          <Trash2 size={17} aria-hidden="true" />
          {pending ? "Removing…" : "Remove permanently"}
        </button>
      </footer>
    </dialog>
  );
}
