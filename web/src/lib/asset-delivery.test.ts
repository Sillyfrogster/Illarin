import { expect, test } from "bun:test";
import type { AssetInstance, DownloadTarget } from "@/lib/api/query";
import {
  deliveryDestinations,
  deliveryFailureLine,
  formatChoices,
  instanceStanding,
  sendActionLabel,
} from "@/lib/asset-delivery";

function target(
  format: string,
  label: string,
  recommended: boolean,
  roles: DownloadTarget["roles"] = [],
): DownloadTarget {
  return { format, label, recommended, roles };
}

function role(
  name: string,
  verdict: "carried" | "reduced" | "dropped",
  extra: { reason?: string; destination?: string } = {},
): DownloadTarget["roles"][number] {
  return {
    role: name,
    label: name,
    verdict,
    sample: { count: 3 },
    ...extra,
  };
}

function instance(over: Partial<AssetInstance> = {}): AssetInstance {
  return {
    instanceId: "i1",
    applicationName: "Lumiverse",
    instanceName: "Desk",
    lastSeenAt: null,
    canReceive: true,
    reportsLibrary: true,
    delivery: null,
    installedGeneration: null,
    updateAvailable: false,
    ...over,
  };
}

test("the recommended format leads and every other format follows it", () => {
  const choices = formatChoices({
    downloads: [
      target("json", "JSON", false),
      target("card", "Character Card V3", true),
    ],
    holdsNothing: false,
  });
  expect(choices.map((choice) => choice.format)).toEqual(["card", "json"]);
  expect(choices[0].recommended).toBe(true);
});

test("a format that carries everything says so rather than counting nothing", () => {
  const [choice] = formatChoices({
    downloads: [target("card", "Card", true, [role("greetings", "carried")])],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("Includes everything");
  expect(choice.losses).toHaveLength(0);
});

test("an asset holding nothing does not claim a format includes everything", () => {
  const [choice] = formatChoices({
    downloads: [target("card", "Card", true)],
    holdsNothing: true,
  });
  expect(choice.cost).toBe("There is nothing in it yet");
});

test("a format names what it leaves behind and how much of it there was", () => {
  const [choice] = formatChoices({
    downloads: [
      target("card", "Card", true, [
        role("expressions", "dropped"),
        role("greetings", "reduced", { reason: "their names" }),
        role("gallery", "carried"),
      ]),
    ],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("2 things left out");
  expect(choice.losses.map((loss) => loss.line)).toEqual([
    "Not included.",
    "Included, without their names.",
  ]);
});

test("a carried role that lands somewhere unusual is worth a note, not a loss", () => {
  const [choice] = formatChoices({
    downloads: [
      target("card", "Card", true, [
        role("scenario", "carried", { destination: "the description" }),
      ]),
    ],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("Includes everything");
  expect(choice.losses.map((loss) => loss.line)).toEqual([
    "Included as the description.",
  ]);
});

test("one lost thing is counted in the singular", () => {
  const [choice] = formatChoices({
    downloads: [target("card", "Card", true, [role("gallery", "dropped")])],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("1 thing left out");
});

test("a file is always a destination and every installation that can receive is another", () => {
  const destinations = deliveryDestinations([
    instance({ instanceId: "a", instanceName: "Desk" }),
    instance({ instanceId: "b", instanceName: "Laptop", canReceive: false }),
  ]);
  expect(destinations.map((one) => one.id)).toEqual(["file", "a"]);
  expect(destinations[0].label).toBe("Download a file");
  expect(destinations[1].label).toBe("Lumiverse — Desk");
});

test("no installation leaves the file as the only destination", () => {
  expect(deliveryDestinations([])).toHaveLength(1);
});

test("the send action says what sending would do this time", () => {
  expect(sendActionLabel(instance())).toBe("Send");
  expect(sendActionLabel(instance({ installedGeneration: 2 }))).toBe(
    "Send again",
  );
  expect(
    sendActionLabel(
      instance({ installedGeneration: 2, updateAvailable: true }),
    ),
  ).toBe("Send the update");
  expect(
    sendActionLabel(
      instance({
        delivery: {
          id: "d",
          instanceId: "i1",
          assetId: "a",
          state: "queued",
          queuedAt: "",
          expiresAt: "",
        },
      }),
    ),
  ).toBe("Waiting to be collected");
});

test("an installation that reports nothing does not pretend to know what it holds", () => {
  expect(instanceStanding(instance({ reportsLibrary: false }))).toBe(
    "This installation does not report what it holds, so Illarin cannot say whether you already have it.",
  );
  expect(instanceStanding(instance())).toBe("Not installed here yet.");
  expect(instanceStanding(instance({ installedGeneration: 1 }))).toBe(
    "Installed and up to date.",
  );
  expect(
    instanceStanding(
      instance({ installedGeneration: 1, updateAvailable: true }),
    ),
  ).toBe("Installed, and a newer version exists here.");
});

test("a delivery that failed says why in words a reader can act on", () => {
  expect(deliveryFailureLine("withdrawn")).toBe(
    "This asset was withdrawn before it could be collected.",
  );
  expect(deliveryFailureLine("unsupported")).toBe(
    "This application accepts no format this asset can be written in.",
  );
  expect(deliveryFailureLine("abandoned")).toBe(
    "The application kept taking this delivery without installing it.",
  );
  expect(deliveryFailureLine(null)).toBe("This delivery did not arrive.");
});
