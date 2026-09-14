import { expect, test } from "bun:test";
import { groupAdditions } from "./extension-additions";

const found = [
  { name: "Commands", value: "/roll" },
  { name: "Commands", value: "/rollquiet" },
  { name: "Macros", value: "{{roll_total}}" },
  { name: "Tools", value: "RollTheDice" },
  { name: "Generation hooks", value: "MESSAGE_RECEIVED" },
];

test("gathers each addition under its group in the order the code was read", () => {
  expect(groupAdditions(found)).toEqual([
    {
      name: "Commands",
      total: 2,
      shown: [
        { key: "0", name: "/roll" },
        { key: "1", name: "/rollquiet" },
      ],
    },
    { name: "Macros", total: 1, shown: [{ key: "2", name: "{{roll_total}}" }] },
    { name: "Tools", total: 1, shown: [{ key: "3", name: "RollTheDice" }] },
    {
      name: "Generation hooks",
      total: 1,
      shown: [{ key: "4", name: "MESSAGE_RECEIVED" }],
    },
  ]);
});

test("shows the first few additions while counting every one in their groups", () => {
  const mixed = [
    { name: "Tools", value: "search_notes" },
    { name: "Macros", value: "{{weather}}" },
    { name: "Tools", value: "tidy_reply" },
  ];

  expect(groupAdditions(mixed, 2)).toEqual([
    { name: "Tools", total: 2, shown: [{ key: "0", name: "search_notes" }] },
    { name: "Macros", total: 1, shown: [{ key: "1", name: "{{weather}}" }] },
  ]);
});
