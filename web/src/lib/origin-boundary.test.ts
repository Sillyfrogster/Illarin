import { expect, test } from "bun:test";
import { isClientModule, reachableFrom, sourceFiles } from "./source-graph";

const ORIGIN_SETTINGS = "src/lib/site-metadata.ts";

test("no browser code learns which origin it is on from the server's settings", () => {
  const leaking: string[] = [];
  for (const path of sourceFiles()) {
    if (path.endsWith(".test.ts") || path.endsWith(".test.tsx")) continue;
    if (!isClientModule(path)) continue;
    if (reachableFrom(path).includes(ORIGIN_SETTINGS)) leaking.push(path);
  }
  expect(leaking).toEqual([]);
});
