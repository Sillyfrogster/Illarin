import { expect, test } from "bun:test";
import { hasNativeShare, shareReport } from "@/lib/article-share";

test("a device that carries a share sheet is offered one", () => {
  expect(hasNativeShare({ share: async () => {} } as Navigator)).toBe(true);
});

test("a browser without one is offered nothing extra", () => {
  expect(hasNativeShare({} as Navigator)).toBe(false);
  expect(hasNativeShare({ share: undefined } as unknown as Navigator)).toBe(
    false,
  );
});

test("a reader is told nothing before they ask for anything", () => {
  expect(shareReport("ready")).toEqual({ said: "", reveal: false });
});

test("a copy that worked says so and reveals nothing", () => {
  expect(shareReport("copied")).toEqual({
    said: "Link copied.",
    reveal: false,
  });
});

test("a copy the browser refused hands the address over instead", () => {
  const report = shareReport("refused");
  expect(report.reveal).toBe(true);
  expect(report.said).not.toBe("");
});
