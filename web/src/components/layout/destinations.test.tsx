import { expect, test } from "bun:test";
import type { SignedInAccount } from "@/lib/auth";
import { accountDestinations } from "./destinations";

const account: SignedInAccount = {
  id: "00000000-0000-4000-8000-000000000022",
  handle: "copy_fixture",
  emailVerified: true,
  email: "copy@example.test",
  discordLinked: false,
  hasPassword: true,
  role: "user",
};

test("account navigation groups the destinations allowed by each role", () => {
  const groups = (
    role: SignedInAccount["role"],
    writer = false,
    emailVerified = true,
  ) =>
    accountDestinations({ ...account, role, emailVerified }, writer).map(
      (group) => group.map(({ id }) => id),
    );
  expect(groups("user")).toEqual([["work"], ["settings"]]);
  expect(groups("user", true, false)).toEqual([
    ["work", "posts"],
    ["settings", "verify"],
  ]);
  expect(groups("moderator")).toEqual([["work"], ["settings"], ["staff"]]);
  expect(groups("admin", true)).toEqual([
    ["work", "posts"],
    ["settings"],
    ["blog-admin", "staff"],
  ]);
  expect(
    accountDestinations(null, false).map((group) => group.map(({ id }) => id)),
  ).toEqual([["sign-in", "sign-up"]]);
});
