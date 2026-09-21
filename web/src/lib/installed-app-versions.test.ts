import { expect, test } from "bun:test";
import { installedVersionsLine } from "@/lib/installed-app-versions";

function spindlePage(installedAppVersions: string[]) {
  return {
    appFormats: [
      { id: "lumiverse", label: "Lumiverse", format: "extension_spindle" },
    ],
    installedAppVersions,
  };
}

test("an extension installed on two app versions names the app and both versions, newest first", () => {
  expect(installedVersionsLine(spindlePage(["1.1.6", "1.2.0"]))).toBe(
    "Installed by readers on Lumiverse 1.2.0 and 1.1.6.",
  );
});

test("versions are ordered by their numbers, so 1.10.0 is newer than 1.9.2", () => {
  expect(
    installedVersionsLine(spindlePage(["1.9.2", "1.10.0", "0.12.1"])),
  ).toBe("Installed by readers on Lumiverse 1.10.0, 1.9.2 and 0.12.1.");
});

test("a long run of versions names the newest four and counts the rest", () => {
  expect(
    installedVersionsLine(
      spindlePage(["1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0", "1.5.0"]),
    ),
  ).toBe(
    "Installed by readers on Lumiverse 1.5.0, 1.4.0, 1.3.0, 1.2.0 and 2 other versions.",
  );
});

test("five versions are all named rather than counting one", () => {
  expect(
    installedVersionsLine(
      spindlePage(["1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0"]),
    ),
  ).toBe(
    "Installed by readers on Lumiverse 1.4.0, 1.3.0, 1.2.0, 1.1.0 and 1.0.0.",
  );
});

test("an extension no group of installations reports a version for shows no line", () => {
  expect(installedVersionsLine(spindlePage([]))).toBeNull();
});

test("a page offering no app to name the versions after shows no line", () => {
  expect(
    installedVersionsLine({ appFormats: [], installedAppVersions: ["1.2.0"] }),
  ).toBeNull();
});

test("a work two apps read names no app, since the versions could belong to either", () => {
  expect(
    installedVersionsLine({
      appFormats: [
        {
          id: "sillytavern",
          label: "SillyTavern",
          format: "extension_sillytavern",
        },
        { id: "lumiverse", label: "Lumiverse", format: "extension_spindle" },
      ],
      installedAppVersions: ["1.2.0"],
    }),
  ).toBe("Installed by readers on versions 1.2.0.");
});
