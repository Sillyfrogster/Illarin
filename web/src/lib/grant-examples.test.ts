import { expect, test } from "bun:test";
import type { components } from "@/lib/api/schema";
import { grantExamples } from "./grant-examples";

const CATEGORY = {
  id: "b1d6e9f2-3a4c-4d5e-8f70-1a2b3c4d5e6f",
  slug: "announcement",
  label: "Announcement",
  position: 1,
  retired: false,
};

const RELEASE = {
  ...CATEGORY,
  id: "c7e1f3a5-9b2d-4c6e-8f1a-3d5b7c9e1f2a",
  slug: "release",
  label: "Release",
};

const GRANT: components["schemas"]["PublicationGrant"] = {
  id: "2c3d4e5f-6071-4829-b0c1-d2e3f4a5b6c7",
  holder: {
    handle: "paperlantern",
    displayName: "Paper Lantern",
    restricted: false,
  },
  app: {
    id: "5e9a1c3b-7d2f-4e86-b4a0-9c8d7e6f5a4b",
    slug: "paper-lantern",
    name: "Paper Lantern",
    home: "https://paperlantern.example/",
    position: 1,
    retired: false,
    destinations: [],
  },
  categories: [CATEGORY, RELEASE],
  defaultCategory: CATEGORY,
  destinations: [
    {
      id: "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
      name: "Community",
      kind: "discord",
      state: "active",
      events: ["publication.post.published.v1"],
      role: "",
      byDefault: true,
    },
    {
      id: "f6a7b8c9-d0e1-4f2a-b3c4-d5e6f7a8b9c0",
      name: "Site hook",
      kind: "webhook",
      state: "active",
      events: ["publication.post.published.v1"],
      role: "",
      byDefault: false,
    },
  ],
  destinationsInherited: true,
  grantedAt: "2026-08-20T15:30:00Z",
  active: true,
};

test("examples carry the grant's own category, app and default destinations", () => {
  const examples = grantExamples(GRANT);
  expect(examples.map((one) => one.title)).toEqual([
    "Check the token",
    "Create a post",
    "Release fields",
    "Publish now",
  ]);
  expect(examples[1].source).toContain(CATEGORY.id);
  expect(examples[2].source).toContain(GRANT.app.id);
  expect(examples[2].source).toContain(
    "https://paperlantern.example/releases/1.0.0",
  );
  expect(examples[3].source).toContain("e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9");
  expect(examples[3].source).not.toContain(
    "f6a7b8c9-d0e1-4f2a-b3c4-d5e6f7a8b9c0",
  );
  expect(examples[3].note).toContain("Community");
});

test("a grant without the release category gets no release example", () => {
  const examples = grantExamples({ ...GRANT, categories: [CATEGORY] });
  expect(examples.map((one) => one.title)).not.toContain("Release fields");
});

test("no example carries a token value, an address or a secret", () => {
  const joined = grantExamples(GRANT)
    .map((one) => one.source)
    .join("\n");
  expect(joined).toContain("Bearer <token>");
  expect(joined).not.toMatch(/ip1\./);
  expect(joined).not.toMatch(/whsec_/);
  expect(joined).not.toContain("discord.com");
});
