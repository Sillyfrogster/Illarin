"use client";

import { ChevronRight, RotateCcw, Trash2 } from "lucide-react";
import { useState } from "react";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  deletePreservedData,
  fetchPreservedData,
  type PreservedData,
} from "@/lib/api/query";
import { cn } from "@/lib/cn";
import { useDraftedChanges } from "@/lib/drafted-changes";

export function PreservedPanel({ workId }: { workId: string }) {
  const candidate = useDraftedChanges();
  const [open, setOpen] = useState(false);
  const [namespaces, setNamespaces] = useState<PreservedData[] | null>(null);
  const [deleting, setDeleting] = useState<PreservedData | null>(null);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function openMenu() {
    setOpen(true);
    if (namespaces !== null) return;
    setMessage("");
    const found = await fetchPreservedData(workId);
    setNamespaces(found);
  }

  async function remove(namespace: string) {
    if (pending) return;
    setPending(true);
    setMessage("");
    try {
      await deletePreservedData(candidate, workId, namespace);
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
          <strong>File extras</strong>
          <span>Data from your file that the page doesn't show</span>
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
            These came with your file but aren't on the page. Removing one takes
            it out of downloads in that format.
          </p>
          {namespaces === null ? (
            <p className={"mt-3 text-meta text-mute italic"}>
              Loading file extras…
            </p>
          ) : namespaces.length === 0 ? (
            <p className={"mt-3 text-meta text-mute italic"}>No file extras</p>
          ) : (
            <ul
              className={
                "mt-3 list-none border-rule border-t [&>li]:flex [&>li]:min-h-13 [&>li]:items-center [&>li]:justify-between [&>li]:gap-3 [&>li]:border-rule [&>li]:border-b [&>li]:text-ui"
              }
            >
              {namespaces.map((namespace) => (
                <li key={namespace.name}>
                  <span>{namespace.label}</span>
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
                    <span className="sr-only">Remove {namespace.label}</span>
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
  namespace: PreservedData;
  pending: boolean;
  onCancel: () => void;
  onDelete: () => void;
}) {
  return (
    <AlertDialog
      onOpenChange={(open) => {
        if (!open) onCancel();
      }}
      open
    >
      <AlertDialogContent className="max-w-[30rem]">
        <p className="text-meta text-mute">Remove file extras</p>
        <AlertDialogTitle>Remove {namespace.label}?</AlertDialogTitle>
        <AlertDialogDescription>
          This detail came with your original file. Downloads in that format
          stop carrying it.
        </AlertDialogDescription>
        <p className="text-meta text-mute">
          This cannot be undone here. Re-upload the original file if you need it
          back.
        </p>
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button disabled={pending} variant="ghost">
              Cancel
            </Button>
          </AlertDialogCancel>
          <Button loading={pending} onClick={onDelete} variant="stop">
            <Trash2 aria-hidden="true" />
            {pending ? "Removing…" : "Remove permanently"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
