"use client";

import { ChevronDown, ChevronUp, Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { ProfileLink } from "@/lib/api/query";
import { cn } from "@/lib/cn";
import {
  addLink,
  LINK_LIMIT,
  moveLink,
  removeLink,
  writeLink,
} from "@/lib/profile-draft";

const CELL =
  "min-h-11 w-full min-w-0 border-0 bg-transparent px-0 font-ui text-ui text-ink outline-offset-2 placeholder:text-mute";

export function ProfileLinks({
  links,
  onChange,
  trouble,
}: {
  links: ProfileLink[];
  onChange: (links: ProfileLink[]) => void;
  trouble?: string;
}) {
  return (
    <fieldset className="m-0 grid min-w-0 gap-3 border-0 p-0">
      <legend className="font-ui text-ui text-ink">Links</legend>
      <p className="font-ui text-meta text-mute">
        Up to {LINK_LIMIT}, in the order you list them. Each needs a label and
        an https address.
      </p>
      {trouble ? (
        <p className="font-ui text-meta text-stop" id="profile-links-trouble">
          {trouble}
        </p>
      ) : null}

      {links.length > 0 ? (
        <ol
          className={cn(
            "m-0 grid list-none gap-px overflow-hidden rounded-plate bg-rule p-0",
            trouble && "inset-ring-2 inset-ring-stop",
          )}
        >
          {links.map((link, index) => (
            <li
              className="flex min-w-0 items-center gap-3 bg-plane pr-2 pl-4"
              // biome-ignore lint/suspicious/noArrayIndexKey: a link's position is its identity here
              key={index}
            >
              <div className="grid min-w-0 flex-1 py-1 sm:grid-cols-[minmax(0,11rem)_minmax(0,1fr)] sm:gap-4">
                <input
                  aria-label={`Link ${index + 1} label`}
                  className={cn(CELL, "font-medium")}
                  maxLength={32}
                  onChange={(event) =>
                    onChange(
                      writeLink(links, index, "label", event.target.value),
                    )
                  }
                  placeholder="Label"
                  type="text"
                  value={link.label}
                />
                <input
                  aria-label={`Link ${index + 1} address`}
                  className={cn(CELL, "text-mute")}
                  maxLength={300}
                  onChange={(event) =>
                    onChange(
                      writeLink(links, index, "address", event.target.value),
                    )
                  }
                  placeholder="https://example.com"
                  type="url"
                  value={link.address}
                />
              </div>
              <div className="flex shrink-0 items-center">
                <Button
                  aria-label={`Move link ${index + 1} up`}
                  disabled={index === 0}
                  onClick={() => onChange(moveLink(links, index, -1))}
                  size="icon"
                  variant="ghost"
                >
                  <ChevronUp aria-hidden="true" />
                </Button>
                <Button
                  aria-label={`Move link ${index + 1} down`}
                  disabled={index === links.length - 1}
                  onClick={() => onChange(moveLink(links, index, 1))}
                  size="icon"
                  variant="ghost"
                >
                  <ChevronDown aria-hidden="true" />
                </Button>
                <Button
                  aria-label={`Remove link ${index + 1}`}
                  onClick={() => onChange(removeLink(links, index))}
                  size="icon"
                  variant="ghost"
                >
                  <X aria-hidden="true" />
                </Button>
              </div>
            </li>
          ))}
        </ol>
      ) : null}

      {links.length < LINK_LIMIT ? (
        <div>
          <Button onClick={() => onChange(addLink(links))} variant="ghost">
            <Plus aria-hidden="true" />
            Add link
          </Button>
        </div>
      ) : null}
    </fieldset>
  );
}
