import { Fragment } from "react";
import { Run, RunHeading, RunItem } from "@/components/ui/run";
import { groupAdditions } from "@/lib/extension-additions";
import { ITEM_META, ITEM_NAME } from "./element-runs";

/** ExtensionAdditions lists what an extension's code registers with its app, under each kind of thing it adds. */
export function ExtensionAdditions({
  fields,
  itemLimit,
}: {
  fields: { name?: string; value: string }[];
  itemLimit?: number;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <p className={ITEM_META}>
        Found in the extension’s code. Anything named only while it runs is not
        listed.
      </p>
      <Run>
        {groupAdditions(fields, itemLimit).map((group) => (
          <Fragment key={group.name}>
            <RunHeading count={`${group.total}`}>{group.name}</RunHeading>
            {group.shown.map((addition) => (
              <RunItem itemKey={addition.key} key={addition.key}>
                <p className={ITEM_NAME}>{addition.name}</p>
              </RunItem>
            ))}
          </Fragment>
        ))}
      </Run>
    </div>
  );
}
