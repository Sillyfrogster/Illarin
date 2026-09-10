import { expect, test } from "bun:test";
import {
  addLink,
  LINK_LIMIT,
  moveLink,
  removeLink,
  whatIsPublic,
  writeLink,
} from "./profile-draft";

const links = [
  { address: "https://one.example", label: "One" },
  { address: "https://two.example", label: "Two" },
  { address: "https://three.example", label: "Three" },
];

test("adds an empty row for the next address", () => {
  const next = addLink(links);

  expect(next).toHaveLength(4);
  expect(next[3]).toEqual({ address: "", label: "" });
});

test("refuses to add beyond the addresses a profile may carry", () => {
  const full = Array.from({ length: LINK_LIMIT }, () => ({
    address: "",
    label: "",
  }));

  expect(addLink(full)).toBe(full);
});

test("writes one field of one address and leaves the rest alone", () => {
  const next = writeLink(links, 1, "label", "Second");

  expect(next[1]).toEqual({ address: "https://two.example", label: "Second" });
  expect(next[0]).toBe(links[0]);
  expect(links[1].label).toBe("Two");
});

test("removes the address at a position", () => {
  expect(removeLink(links, 1).map((link) => link.label)).toEqual([
    "One",
    "Three",
  ]);
});

test("moves an address past its neighbour", () => {
  expect(moveLink(links, 0, 1).map((link) => link.label)).toEqual([
    "Two",
    "One",
    "Three",
  ]);
  expect(moveLink(links, 2, -1).map((link) => link.label)).toEqual([
    "One",
    "Three",
    "Two",
  ]);
});

test("a move off either end changes nothing", () => {
  expect(moveLink(links, 0, -1)).toBe(links);
  expect(moveLink(links, 2, 1)).toBe(links);
});

test("names what a visitor would see on the profile", () => {
  expect(
    whatIsPublic({
      avatar: true,
      biography: "Writes worlds.",
      contactEmail: "hello@example.com",
      displayName: "Someone",
      links,
    }),
  ).toEqual(["display name", "avatar", "biography", "contact", "3 links"]);
});

test("counts a single address in the singular", () => {
  expect(
    whatIsPublic({
      avatar: false,
      biography: "",
      contactEmail: "",
      displayName: "",
      links: [links[0]],
    }),
  ).toEqual(["1 link"]);
});

test("an untouched profile shows nothing but its handle", () => {
  expect(
    whatIsPublic({
      avatar: false,
      biography: "",
      contactEmail: "",
      displayName: "",
      links: [],
    }),
  ).toEqual([]);
});

test("an address with neither label nor location is not yet public", () => {
  expect(
    whatIsPublic({
      avatar: false,
      biography: "",
      contactEmail: "",
      displayName: "",
      links: [{ address: "", label: "" }],
    }),
  ).toEqual([]);
});
