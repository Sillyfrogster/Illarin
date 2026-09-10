import { expect, test } from "bun:test";
import { fileWeight } from "./file-weight";

test("writes a small file in bytes, because rounding it hides how small it is", () => {
  expect(fileWeight(0)).toBe("0 bytes");
  expect(fileWeight(1)).toBe("1 byte");
  expect(fileWeight(940)).toBe("940 bytes");
});

test("writes a kilobyte file to one place", () => {
  expect(fileWeight(1024)).toBe("1 KB");
  expect(fileWeight(5000)).toBe("4.9 KB");
});

test("writes a megabyte file to one place", () => {
  expect(fileWeight(1048576)).toBe("1 MB");
  expect(fileWeight(1572864)).toBe("1.5 MB");
  expect(fileWeight(33554432)).toBe("32 MB");
});

test("keeps going past megabytes rather than printing a five-figure count", () => {
  expect(fileWeight(2147483648)).toBe("2 GB");
});
