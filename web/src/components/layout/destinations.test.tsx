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

test("account navigation exposes the destinations allowed by each role", () => {
  const destinations = (role: SignedInAccount["role"], writer = false) =>
    accountDestinations({ ...account, role }, writer).map(({ id }) => id);
  expect(destinations("user")).toEqual(["work", "profile", "settings"]);
  expect(destinations("user", true)).toEqual([
    "work",
    "profile",
    "settings",
    "posts",
  ]);
  expect(destinations("moderator")).toContain("staff");
  expect(destinations("admin", true)).toEqual([
    "work",
    "profile",
    "settings",
    "posts",
    "blog-admin",
    "staff",
  ]);
  expect(accountDestinations(null, false).map(({ id }) => id)).toEqual([
    "sign-in",
    "sign-up",
  ]);
  expect(
    accountDestinations({ ...account, emailVerified: false }, false).map(
      ({ id }) => id,
    ),
  ).toContain("verify");
});
