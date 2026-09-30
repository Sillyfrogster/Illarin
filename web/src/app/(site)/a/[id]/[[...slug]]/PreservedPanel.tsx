"use client";

import { RotateCcw, Trash2 } from "lucide-react";
import { useState } from "react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import {
  deletePreservedData,
  fetchPreservedData,
  type PreservedData,
} from "@/lib/api/query";
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
    <>
      <Accordion
        className="mt-3.5 border-rule border-t pt-2"
        collapsible
        onValueChange={(value) => (value ? void openMenu() : setOpen(false))}
        type="single"
        value={open ? "open" : ""}
      >
        <AccordionItem value="open">
          <AccordionTrigger>File extras</AccordionTrigger>
          <p className="pl-6 text-meta text-mute">
            Data from your file that the page doesn't show
          </p>
          <AccordionContent>
            <div className="pt-3 pb-3.5 pl-6">
              <p className={"text-meta text-mute"}>
                These came with your file but aren't on the page. Removing one
                takes it out of downloads in that format.
              </p>
              {namespaces === null ? (
                <p className={"mt-3 text-meta text-mute italic"}>
                  Loading file extras…
                </p>
              ) : namespaces.length === 0 ? (
                <p className={"mt-3 text-meta text-mute italic"}>
                  No file extras
                </p>
              ) : (
                <ul
                  className={
                    "mt-3 list-none border-rule border-t [&>li]:flex [&>li]:min-h-13 [&>li]:items-center [&>li]:justify-between [&>li]:gap-3 [&>li]:border-rule [&>li]:border-b [&>li]:text-ui"
                  }
                >
                  {namespaces.map((namespace) => (
                    <li key={namespace.name}>
                      <span>{namespace.label}</span>
                      <Tooltip content={`Remove ${namespace.label}`}>
                        <Button
                          aria-label={`Remove ${namespace.label}`}
                          className="hover:text-stop"
                          onClick={() => {
                            setMessage("");
                            setDeleting(namespace);
                          }}
                          size="icon"
                          variant="ghost"
                        >
                          <Trash2 aria-hidden="true" />
                        </Button>
                      </Tooltip>
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
                <Button
                  className="mt-3"
                  onClick={() => {
                    setNamespaces(null);
                    void openMenu();
                  }}
                >
                  <RotateCcw aria-hidden="true" />
                  Try again
                </Button>
              ) : null}
            </div>
          </AccordionContent>
        </AccordionItem>
      </Accordion>

      {deleting ? (
        <DeleteNamespaceDialog
          namespace={deleting}
          pending={pending}
          onCancel={() => setDeleting(null)}
          onDelete={() => void remove(deleting.name)}
        />
      ) : null}
    </>
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
