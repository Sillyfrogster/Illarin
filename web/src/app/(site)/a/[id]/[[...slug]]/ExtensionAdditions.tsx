import { Fragment } from "react";
import { Item, ItemGroup, ItemGroupHeading } from "@/components/ui/item";
import { groupAdditions } from "@/lib/extension-additions";
import { ITEM_META, ITEM_NAME } from "./element-runs";

/** ExtensionAdditions lists what an extension's code registers with its app, under each sort of thing it adds. */
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
      <ItemGroup>
        {groupAdditions(fields, itemLimit).map((group) => (
          <Fragment key={group.name}>
            <ItemGroupHeading count={`${group.total}`}>
              {group.name}
            </ItemGroupHeading>
            {group.shown.map((addition) => (
              <Item itemKey={addition.key} key={addition.key}>
                <p className={ITEM_NAME}>{addition.name}</p>
              </Item>
            ))}
          </Fragment>
        ))}
      </ItemGroup>
    </div>
  );
}
