import { expect, test } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import type { SignedInAccount } from "@/lib/auth";
import { DestinationIcon } from "./DestinationIcon";
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

test("account navigation names each task and keeps icons when labels change", () => {
  const destinations = accountDestinations(account, true, true);
  expect(destinations.map(({ label, href }) => [label, href])).toEqual([
    ["Your work", "/@copy_fixture"],
    ["Account settings", "/settings"],
    ["Your posts", "/posts"],
    ["Blog administration", "/admin/blog"],
  ]);
  const icons = ["circle-user-round", "settings", "notebook-pen", "signature"];
  for (const [index, destination] of destinations.entries()) {
    const renamed = { ...destination, label: "A different label" };
    const markup = renderToStaticMarkup(<DestinationIcon id={renamed.id} />);
    expect(markup).toContain(`lucide-${icons[index]}`);
    expect(markup).toContain('aria-hidden="true"');
  }
});

test("sign-in and verification destinations keep their labels and icons", () => {
  for (const visitor of [null, undefined]) {
    const destinations = accountDestinations(visitor, false, false);
    expect(destinations.map(({ label }) => label)).toEqual([
      "Sign in",
      "Create account",
    ]);
    expect(
      renderToStaticMarkup(<DestinationIcon id={destinations[0].id} />),
    ).toContain("lucide-log-in");
    expect(
      renderToStaticMarkup(<DestinationIcon id={destinations[1].id} />),
    ).toContain("lucide-user-plus");
  }
  const destinations = accountDestinations(
    { ...account, emailVerified: false },
    false,
    false,
  );
  expect(destinations.map(({ label }) => label)).toEqual([
    "Your work",
    "Account settings",
    "Verify email",
  ]);
  expect(
    renderToStaticMarkup(<DestinationIcon id={destinations[2].id} />),
  ).toContain("lucide-mail");
});
