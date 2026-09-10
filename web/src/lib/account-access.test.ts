import { expect, test } from "bun:test";
import { waysIn } from "./account-access";
import type { SignedInAccount } from "./auth";

const account = (rest: Partial<SignedInAccount> = {}): SignedInAccount =>
  ({
    discordLinked: false,
    email: null,
    emailVerified: false,
    handle: "someone",
    hasPassword: false,
    ...rest,
  }) as SignedInAccount;

function find(ways: ReturnType<typeof waysIn>, id: string) {
  const way = ways.find((one) => one.id === id);
  if (!way) throw new Error(`no way in called ${id}`);
  return way;
}

test("lists the three ways into an account, in the order they are set up", () => {
  expect(waysIn(account()).map((way) => way.id)).toEqual([
    "email",
    "discord",
    "password",
  ]);
});

test("a new account has no way in that is settled yet", () => {
  for (const way of waysIn(account())) expect(way.settled).toBe(false);
});

test("an unverified address is not a way back in", () => {
  const way = find(waysIn(account({ email: "one@example.com" })), "email");

  expect(way.settled).toBe(false);
  expect(way.standing).toBe("one@example.com");
});

test("a verified address says so and offers nothing further to do", () => {
  const way = find(
    waysIn(account({ email: "one@example.com", emailVerified: true })),
    "email",
  );

  expect(way.settled).toBe(true);
  expect(way.standing).toBe("one@example.com");
});

test("an account with no address at all says that rather than showing nothing", () => {
  expect(find(waysIn(account()), "email").standing).toBe(
    "No verified address yet",
  );
});

test("Discord can only be detached once another way in is settled", () => {
  const only = find(waysIn(account({ discordLinked: true })), "discord");

  expect(only.settled).toBe(true);
  expect(only.canDetach).toBe(false);
});

test("Discord detaches once an address is verified and a password is set", () => {
  const way = find(
    waysIn(
      account({
        discordLinked: true,
        email: "one@example.com",
        emailVerified: true,
        hasPassword: true,
      }),
    ),
    "discord",
  );

  expect(way.canDetach).toBe(true);
});

test("a verified address alone does not release Discord", () => {
  const way = find(
    waysIn(
      account({
        discordLinked: true,
        email: "one@example.com",
        emailVerified: true,
      }),
    ),
    "discord",
  );

  expect(way.canDetach).toBe(false);
});

test("Discord cannot be attached until an address is verified", () => {
  expect(find(waysIn(account()), "discord").canAttach).toBe(false);
  expect(
    find(
      waysIn(account({ email: "one@example.com", emailVerified: true })),
      "discord",
    ).canAttach,
  ).toBe(true);
});

test("a password that is set is settled and is replaced rather than added", () => {
  const way = find(waysIn(account({ hasPassword: true })), "password");

  expect(way.settled).toBe(true);
  expect(way.standing).toBe("Set");
});

test("counts how many ways in are settled, so the page can say how safe it is", () => {
  const bare = waysIn(account()).filter((way) => way.settled);
  const full = waysIn(
    account({
      discordLinked: true,
      email: "one@example.com",
      emailVerified: true,
      hasPassword: true,
    }),
  ).filter((way) => way.settled);

  expect(bare).toHaveLength(0);
  expect(full).toHaveLength(3);
});
