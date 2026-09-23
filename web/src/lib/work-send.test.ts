import { expect, test } from "bun:test";
import type { DownloadFormat, WorkConnectedApp } from "@/lib/api/query";
import {
  canSendWork,
  connectedAppStanding,
  downloadAddress,
  installsInApp,
  orderedFormats,
  readerAppFormat,
  sendActionLabel,
  sendFailureLine,
} from "@/lib/work-send";

function offered(
  format: string,
  label: string,
  recommended: boolean,
): DownloadFormat {
  return { format, label, recommended, roles: [] };
}

function connectedApp(over: Partial<WorkConnectedApp> = {}): WorkConnectedApp {
  return {
    connectedAppId: "i1",
    appName: "Lumiverse",
    name: "Desk",
    lastSeenAt: null,
    canReceive: true,
    reportsLibrary: true,
    send: null,
    installedVersion: null,
    updateAvailable: false,
    ...over,
  };
}

test("only a published work that staff have not taken down can be sent to an installation", () => {
  const takedown = {
    reason: "Copyright report under review",
    actor: "night.staff",
    at: "2026-09-14T09:00:00Z",
  };
  expect(canSendWork({ lifecycle: "published" })).toBe(true);
  expect(canSendWork({ lifecycle: "published", takedown })).toBe(false);
  expect(canSendWork({ lifecycle: "draft" })).toBe(false);
});

test("the reader's app leads the formats, else the recommended one, else the list as given", () => {
  const downloads = [
    offered("ccv2", "Character Card V2", false),
    offered("ccv3", "Character Card V3", true),
    offered("charx", "CharX", false),
  ];
  expect(orderedFormats(downloads).map((one) => one.format)).toEqual([
    "ccv3",
    "ccv2",
    "charx",
  ]);
  expect(orderedFormats(downloads, "charx").map((one) => one.format)).toEqual([
    "charx",
    "ccv2",
    "ccv3",
  ]);
  expect(orderedFormats([downloads[0]], "charx")).toEqual([downloads[0]]);
});

test("a reader with an app set is handed that app's format, and nobody else is", () => {
  const apps = [{ id: "risu", label: "RisuAI", format: "charx" }];
  expect(readerAppFormat(apps, "risu")?.format).toBe("charx");
  expect(readerAppFormat(apps, "sillytavern")).toBeNull();
  expect(readerAppFormat(apps, null)).toBeNull();
});

test("the send action says what sending would do this time", () => {
  expect(sendActionLabel(connectedApp())).toBe("Send to Desk");
  expect(sendActionLabel(connectedApp({ installedVersion: 2 }))).toBe(
    "Send again to Desk",
  );
  expect(
    sendActionLabel(
      connectedApp({ installedVersion: 2, updateAvailable: true }),
    ),
  ).toBe("Send the update to Desk");
  expect(
    sendActionLabel(
      connectedApp({
        send: {
          id: "d",
          connectedAppId: "i1",
          workId: "a",
          state: "queued",
          queuedAt: "",
          settledAt: null,
          expiresAt: "",
          updatesInstall: false,
        },
      }),
    ),
  ).toBe("Sent to Desk");
});

test("a delivered send no longer blocks sending, and an extension is installed rather than sent", () => {
  const delivered = connectedApp({
    send: {
      id: "d",
      connectedAppId: "i1",
      workId: "a",
      state: "delivered",
      queuedAt: "",
      settledAt: "2026-09-13T10:01:00Z",
      expiresAt: "",
      updatesInstall: false,
    },
    installedVersion: 1,
  });
  expect(sendActionLabel(delivered)).toBe("Send again to Desk");
  expect(sendActionLabel(connectedApp(), true)).toBe("Install on Desk");
  expect(sendActionLabel(delivered, true)).toBe("Install again on Desk");
  expect(
    sendActionLabel(
      connectedApp({ installedVersion: 1, updateAvailable: true }),
      true,
    ),
  ).toBe("Update on Desk");
});

test("only an extension is installed in a connected app", () => {
  expect(installsInApp("extension")).toBe(true);
  expect(installsInApp("character")).toBe(false);
});

test("an installation that reports nothing does not pretend to know what it holds", () => {
  expect(connectedAppStanding(connectedApp({ reportsLibrary: false }))).toBe(
    "This app does not report installed works. Installation status is unavailable.",
  );
  expect(connectedAppStanding(connectedApp())).toBe("Not installed here yet.");
  expect(connectedAppStanding(connectedApp({ installedVersion: 1 }))).toBe(
    "Installed and up to date.",
  );
  expect(
    connectedAppStanding(
      connectedApp({ installedVersion: 1, updateAvailable: true }),
    ),
  ).toBe("Installed, and a newer version exists here.");
});

test("a send that failed says why in words a reader can act on", () => {
  expect(sendFailureLine("withdrawn")).toBe(
    "This work was withdrawn before it could be collected.",
  );
  expect(sendFailureLine("unsupported")).toBe(
    "This app accepts no format this work can be written in.",
  );
  expect(sendFailureLine("abandoned")).toBe(
    "The app kept collecting this send without installing it.",
  );
  expect(sendFailureLine(null)).toBe("This send did not arrive.");
});

test("a download names its version only where one is chosen", () => {
  expect(downloadAddress({ workId: "work", format: "charx" })).toBe(
    "/download/work/charx",
  );
  expect(downloadAddress({ workId: "work", format: "charx", version: 3 })).toBe(
    "/download/work/charx?version=3",
  );
});
