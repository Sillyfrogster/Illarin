import { expect, test } from "bun:test";
import type {
  AssetBlock,
  AssetElement,
  AssetImage,
  AssetInstance,
  DownloadTarget,
} from "@/lib/api/query";
import {
  appLabel,
  deliveryDestinations,
  deliveryFailureLine,
  downloadAddress,
  downloadBytes,
  fileSize,
  formatChoices,
  installsOnInstance,
  instanceStanding,
  sendActionLabel,
  travellingGallery,
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
  extra: { reason?: string; destination?: string; shownBy?: string[] } = {},
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
  expect(choice.cost).toBe("No exportable content yet");
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
  expect(choice.cost).toBe("2 content types have limited support");
  expect(choice.losses.map((loss) => loss.line)).toEqual([
    "Not included.",
    "Included, without their names.",
  ]);
});

test("a carried role that lands somewhere unusual is a note, and not everything travels", () => {
  const [choice] = formatChoices({
    downloads: [
      target("card", "Card", true, [
        role("scenario", "carried", {
          destination: "It goes into the description instead.",
        }),
      ]),
    ],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("1 content type unsupported by some apps");
  expect(choice.losses.map((loss) => loss.line)).toEqual([
    "It goes into the description instead.",
  ]);
});

test("one lost thing is counted in the singular", () => {
  const [choice] = formatChoices({
    downloads: [target("card", "Card", true, [role("gallery", "dropped")])],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("1 content type has limited support");
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
          settledAt: null,
          expiresAt: "",
          updatesInstall: false,
        },
      }),
    ),
  ).toBe("Waiting to be collected");
});

test("a delivered delivery no longer blocks sending, and an extension is installed rather than sent", () => {
  const delivered = instance({
    delivery: {
      id: "d",
      instanceId: "i1",
      assetId: "a",
      state: "delivered",
      queuedAt: "",
      settledAt: "2026-09-13T10:01:00Z",
      expiresAt: "",
      updatesInstall: false,
    },
    installedGeneration: 1,
  });
  expect(sendActionLabel(delivered)).toBe("Send again");
  expect(sendActionLabel(instance(), true)).toBe("Install on Desk");
  expect(sendActionLabel(delivered, true)).toBe("Install again on Desk");
  expect(
    sendActionLabel(
      instance({ installedGeneration: 1, updateAvailable: true }),
      true,
    ),
  ).toBe("Update on Desk");
});

test("only an extension is installed on an instance", () => {
  expect(installsOnInstance("extension")).toBe(true);
  expect(installsOnInstance("character")).toBe(false);
});

test("an installation that reports nothing does not pretend to know what it holds", () => {
  expect(instanceStanding(instance({ reportsLibrary: false }))).toBe(
    "This application does not report installed assets. Installation status is unavailable.",
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

function galleryBlock(
  images: { mediaId: string; name?: string; omitFromDownloads?: boolean }[],
): AssetBlock {
  return {
    id: "block-gallery",
    definition: "gallery",
    title: "Gallery",
    position: 1,
    hidden: false,
    layout: "single",
    width: "full",
    required: false,
    hideable: true,
    elements: [
      {
        id: "element-gallery",
        type: "image_set",
        role: "gallery",
        slot: "main",
        isEmpty: images.length === 0,
        content: { images },
      } as AssetElement,
    ],
  } as AssetBlock;
}

function image(
  id: string,
  bytes: number,
  role: AssetImage["role"],
): AssetImage {
  return {
    id,
    role,
    isCover: role === "avatar",
    detailUrl: `/media/${id}`,
    thumbUrl: `/media/${id}/thumb`,
    width: 100,
    height: 100,
    bytes,
  };
}

test("the creator's own choice is what a download carries when nobody changed it", () => {
  const gallery = travellingGallery({
    blocks: [
      galleryBlock([
        { mediaId: "a" },
        { mediaId: "b", omitFromDownloads: true },
      ]),
    ],
    images: [image("a", 10, "gallery"), image("b", 20, "gallery")],
  });

  expect(gallery.map((one) => one.mediaId)).toEqual(["a", "b"]);
  expect(gallery.map((one) => one.chosen)).toEqual([true, false]);
});

test("an archive carries the images as they are and a card base64s them", () => {
  const blocks = [galleryBlock([{ mediaId: "a" }])];
  const images = [
    image("a", 3_000_000, "gallery"),
    image("c", 1_000_000, "avatar"),
  ];

  const archive = downloadBytes({
    format: "charx",
    blocks,
    images,
    chosen: ["a"],
  });
  const card = downloadBytes({
    format: "chara_card_v3",
    blocks,
    images,
    chosen: ["a"],
  });

  expect(archive).toBe(4_000_000);
  expect(card).toBeGreaterThan(archive);
});

test("leaving an image out takes its weight off the download", () => {
  const blocks = [galleryBlock([{ mediaId: "a" }, { mediaId: "b" }])];
  const images = [
    image("a", 1_000_000, "gallery"),
    image("b", 2_000_000, "gallery"),
  ];

  expect(
    downloadBytes({ format: "charx", blocks, images, chosen: ["a", "b"] }),
  ).toBe(3_000_000);
  expect(
    downloadBytes({ format: "charx", blocks, images, chosen: ["a"] }),
  ).toBe(1_000_000);
});

test("a format that drops the gallery counts none of it", () => {
  const blocks = [galleryBlock([{ mediaId: "a" }])];
  const images = [image("a", 1_000_000, "gallery")];

  expect(
    downloadBytes({
      format: "chara_card_v2",
      blocks,
      images,
      chosen: ["a"],
      carries: false,
    }),
  ).toBe(0);
});

test("a file size reads in the unit a person would say it in", () => {
  expect(fileSize(0)).toBe("0 KB");
  expect(fileSize(4096)).toBe("4 KB");
  expect(fileSize(4_600_000)).toBe("4.4 MB");
});

test("a format that lands content somewhere apps ignore does not claim everything travels", () => {
  const [elsewhere, everything] = formatChoices({
    downloads: [
      target("charx", "CharX", false, [role("gallery", "carried")]),
      target("chara_card_v3", "Character Card V3", true, [
        role("gallery", "carried", {
          destination: "In the file, but only some apps show them.",
        }),
      ]),
    ],
    holdsNothing: false,
  });

  expect(everything.cost).toBe("Includes everything");
  expect(elsewhere.cost).toBe("1 content type unsupported by some apps");
});

test("a format that drops the gallery says so, so the chooser can stop offering it", () => {
  const [drops, carries] = formatChoices({
    downloads: [
      target("chara_card_v2", "Character Card V2", true, [
        role("gallery", "dropped"),
      ]),
      target("charx", "CharX", false, [role("gallery", "carried")]),
    ],
    holdsNothing: false,
  });

  expect(drops.carriesGallery).toBe(false);
  expect(carries.carriesGallery).toBe(true);
});

const inlineGallery = role("gallery", "carried", {
  destination: "Written into the card itself.",
  shownBy: ["risu"],
});

test("an app that shows a destination is told the content reaches it", () => {
  const [choice] = formatChoices({
    downloads: [target("ccv3", "Character Card V3", true, [inlineGallery])],
    holdsNothing: false,
    app: "risu",
    apps: [{ id: "risu", label: "RisuAI", format: "ccv3" }],
  });
  expect(choice.cost).toBe("Includes everything");
  expect(choice.carriesGallery).toBe(true);
  expect(choice.gallery?.line).toBe("");
});

test("an app that shows no destination is told the content is left out", () => {
  const [choice] = formatChoices({
    downloads: [target("ccv3", "Character Card V3", true, [inlineGallery])],
    holdsNothing: false,
    app: "sillytavern",
    apps: [{ id: "sillytavern", label: "SillyTavern", format: "ccv3" }],
  });
  expect(choice.cost).toBe("1 content type has limited support");
  expect(choice.carriesGallery).toBe(false);
  expect(choice.gallery?.line).toBe("SillyTavern does not show these.");
});

test("with no app chosen a destination stays a note rather than a loss", () => {
  const [choice] = formatChoices({
    downloads: [target("ccv3", "Character Card V3", true, [inlineGallery])],
    holdsNothing: false,
  });
  expect(choice.cost).toBe("1 content type unsupported by some apps");
  expect(choice.carriesGallery).toBe(true);
});

test("a destination names the apps that show it and the apps that do not", () => {
  const [choice] = formatChoices({
    downloads: [target("ccv3", "Character Card V3", true, [inlineGallery])],
    holdsNothing: false,
    apps: [
      { id: "sillytavern", label: "SillyTavern", format: "ccv3" },
      { id: "risu", label: "RisuAI", format: "ccv3" },
      { id: "lumiverse", label: "Lumiverse", format: "ccv3" },
    ],
  });
  expect(choice.gallery?.line).toBe(
    "Written into the card itself. RisuAI shows these; SillyTavern and Lumiverse do not.",
  );
});

test("a destination no listed app shows says so without naming one", () => {
  const [choice] = formatChoices({
    downloads: [target("ccv3", "Character Card V3", true, [inlineGallery])],
    holdsNothing: false,
    apps: [{ id: "sillytavern", label: "SillyTavern", format: "ccv3" }],
  });
  expect(choice.gallery?.line).toBe(
    "Written into the card itself. SillyTavern does not show these.",
  );
});

test("the format an app is offered leads the list while that app is chosen", () => {
  const choices = formatChoices({
    downloads: [
      target("ccv3", "Character Card V3", true, [inlineGallery]),
      target("charx", "CharX", false, [role("gallery", "carried")]),
    ],
    holdsNothing: false,
    app: "sillytavern",
    apps: [{ id: "sillytavern", label: "SillyTavern", format: "charx" }],
  });
  expect(choices.map((choice) => choice.format)).toEqual(["charx", "ccv3"]);
  expect(choices[0].recommended).toBe(true);
});

test("an app is named by its own label rather than its id", () => {
  expect(
    appLabel([{ id: "risu", label: "RisuAI", format: "charx" }], "risu"),
  ).toBe("RisuAI");
  expect(appLabel([], "risu")).toBe("");
});

test("a download names its version, and the reader's images only where they differ", () => {
  const gallery = [
    { mediaId: "a", name: "A", chosen: true, bytes: 1, thumbUrl: "" },
    { mediaId: "b", name: "B", chosen: false, bytes: 1, thumbUrl: "" },
  ];
  const address = (included: string[], version?: number) =>
    downloadAddress({
      assetId: "asset",
      format: "charx",
      carried: true,
      gallery,
      included,
      version,
    });

  expect(address(["a"])).toBe("/download/asset/charx");
  expect(address(["a", "b"])).toBe("/download/asset/charx?images=a%2Cb");
  expect(address(["a"], 3)).toBe("/download/asset/charx?version=3");
  expect(address([], 3)).toBe("/download/asset/charx?version=3&images=");
  expect(
    downloadAddress({
      assetId: "asset",
      format: "chara_card_v2",
      carried: false,
      gallery,
      included: [],
      version: 2,
    }),
  ).toBe("/download/asset/chara_card_v2?version=2");
});
