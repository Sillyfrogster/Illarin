"use client";

import { Eye, Link2, MoreHorizontal, PencilLine, Trash2 } from "lucide-react";
import { type ReactNode, useState } from "react";
import { Button } from "./button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "./dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./dropdown-menu";

export function OwnerMenu({
  name,
  noun,
  href,
  frozen = false,
  onEdit,
  onDelete,
  visibility,
  visibilityItems,
}: {
  name: string;
  noun: string;
  href: string | null;
  frozen?: boolean;
  onEdit: () => void;
  onDelete: () => Promise<void>;
  visibility?: ReactNode;
  visibilityItems?: ReactNode;
}) {
  const [dialog, setDialog] = useState<"delete" | "visibility" | "copy" | null>(
    null,
  );
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function remove() {
    setPending(true);
    setMessage("");
    try {
      await onDelete();
      setDialog(null);
    } catch {
      setMessage(`Illarin could not delete this ${noun}. Try again.`);
    } finally {
      setPending(false);
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={`Actions for ${name}`}
            variant="outline"
            className="size-11 bg-plane/95 p-0"
          >
            <MoreHorizontal aria-hidden="true" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem disabled={frozen} onSelect={onEdit}>
            <PencilLine aria-hidden="true" />
            Edit
          </DropdownMenuItem>
          {visibility ? (
            <DropdownMenuItem
              disabled={frozen}
              onSelect={() => {
                setMessage("");
                setDialog("visibility");
              }}
            >
              <Eye aria-hidden="true" />
              Visibility
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem
            disabled={!href}
            onSelect={async () => {
              if (!href) return;
              try {
                await navigator.clipboard.writeText(
                  new URL(href, location.origin).href,
                );
                setMessage("Link copied");
              } catch {
                setMessage("Copy this link");
              }
              setDialog("copy");
            }}
          >
            <Link2 aria-hidden="true" />
            Copy link
          </DropdownMenuItem>
          {visibilityItems ? (
            <>
              <DropdownMenuSeparator />
              {visibilityItems}
              <DropdownMenuSeparator />
            </>
          ) : null}
          <DropdownMenuItem
            disabled={frozen}
            className="text-stop"
            onSelect={() => {
              setMessage("");
              setDialog("delete");
            }}
          >
            <Trash2 aria-hidden="true" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Dialog
        open={dialog !== null}
        onOpenChange={(open) => {
          if (!open && !pending) setDialog(null);
        }}
      >
        <DialogContent className="max-w-lg p-6 sm:p-8 overflow-y-auto">
          <DialogTitle className="pr-10 text-section font-medium">
            {dialog === "delete"
              ? `Delete this ${noun}?`
              : dialog === "copy"
                ? message
                : "Visibility"}
          </DialogTitle>
          <DialogDescription className="mt-2 text-ui text-mute wrap-anywhere">
            {dialog === "delete"
              ? `${name}. You can restore it for 30 days. After that it is gone.`
              : name}
          </DialogDescription>
          {dialog === "delete" ? (
            <>
              {message ? (
                <p role="alert" className="mt-4 text-stop">
                  {message}
                </p>
              ) : null}
              <div className="mt-6 flex justify-end gap-3">
                <Button disabled={pending} onClick={() => setDialog(null)}>
                  Cancel
                </Button>
                <Button variant="stop" loading={pending} onClick={remove}>
                  Delete {noun}
                </Button>
              </div>
            </>
          ) : dialog === "visibility" ? (
            <div className="mt-6">{visibility}</div>
          ) : dialog === "copy" && href ? (
            <input
              aria-label="Link"
              className="mt-4 min-h-11 w-full rounded-control bg-deep px-3 text-ui"
              readOnly
              value={new URL(href, location.origin).href}
              onFocus={(event) => event.target.select()}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  );
}
