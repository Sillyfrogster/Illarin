import { describe, expect, test } from "bun:test";
import type {
  PostDelivery,
  PublicationApp,
  PublicationCategory,
  PublicationDestination,
  PublicationGrant,
  PublicationToken,
} from "@/lib/api/query";
import {
  canReplay,
  destinationActions,
  destinationStanding,
  destinationTakes,
  grantAllowance,
  nothingDelivered,
  nothingIn,
  REGISTERS,
  registerName,
  registerStandings,
  tokenEnded,
  tokenStanding,
} from "./publication-register";

function category(shape: Partial<PublicationCategory>): PublicationCategory {
  return {
    id: "cat-1",
    slug: "news",
    label: "News",
    position: 0,
    retired: false,
    ...shape,
  };
}

function app(shape: Partial<PublicationApp>): PublicationApp {
  return {
    id: "app-1",
    slug: "beacon",
    name: "Beacon",
    home: "https://beacon.example.test",
    position: 0,
    retired: false,
    destinations: [],
    ...shape,
  };
}

function grant(shape: Partial<PublicationGrant>): PublicationGrant {
  return {
    id: "grant-1",
    holder: {
      handle: "writer",
      displayName: "",
      avatar: null,
      restricted: false,
    },
    app: app({}),
    categories: [category({})],
    defaultCategory: category({}),
    destinations: [],
    destinationsInherited: true,
    grantedAt: "2026-08-01T09:00:00Z",
    active: true,
    ...shape,
  };
}

function destination(
  shape: Partial<PublicationDestination>,
): PublicationDestination {
  return {
    id: "dest-1",
    kind: "webhook",
    name: "Relay",
    host: "hooks.example.test",
    address: "https://hooks.example.test/…",
    state: "active",
    events: ["publication.post.published.v1"],
    secretSetAt: "2026-08-01T09:00:00Z",
    verifiedAt: "2026-08-02T09:00:00Z",
    createdAt: "2026-08-01T09:00:00Z",
    ...shape,
  };
}

function delivery(shape: Partial<PostDelivery>): PostDelivery {
  return {
    id: "del-1",
    eventId: "event-1",
    eventType: "publication.post.published.v1",
    postId: "post-1",
    postTitle: "A title",
    revisionId: "rev-1",
    destination: "Relay",
    kind: "webhook",
    messageId: "",
    removed: false,
    state: "delivered",
    run: 1,
    attempts: 1,
    occurredAt: "2026-09-01T09:00:00Z",
    dueAt: "2026-09-01T09:00:00Z",
    settledAt: "2026-09-01T09:00:02Z",
    ...shape,
  };
}

function token(shape: Partial<PublicationToken>): PublicationToken {
  return {
    id: "tok-1",
    grantId: "grant-1",
    name: "Release robot",
    prefix: "ilp_9f2a",
    createdAt: "2026-06-20T10:00:00Z",
    active: true,
    ...shape,
  };
}

const EMPTY = {
  grants: [],
  apps: [],
  categories: [],
  destinations: [],
  stopped: 0,
};

describe("the registers", () => {
  test("names every register it lists", () => {
    for (const register of REGISTERS) {
      expect(registerName(register).length).toBeGreaterThan(0);
    }
  });

  test("counts only what is still in use", () => {
    const standings = registerStandings({
      ...EMPTY,
      grants: [grant({}), grant({ id: "grant-2", active: false })],
      apps: [app({}), app({ id: "app-2", retired: true })],
      categories: [
        category({}),
        category({ id: "cat-2", retired: true }),
        category({ id: "cat-3" }),
      ],
      destinations: [destination({})],
    });

    expect(standings.contributors.count).toBe(1);
    expect(standings.apps.count).toBe(1);
    expect(standings.categories.count).toBe(2);
    expect(standings.destinations.count).toBe(1);
  });

  test("says nothing about deliveries until one stops short", () => {
    expect(registerStandings(EMPTY).deliveries).toEqual({
      attention: false,
      count: null,
    });
    expect(registerStandings({ ...EMPTY, stopped: 3 }).deliveries).toEqual({
      attention: true,
      count: 3,
    });
  });

  test("only deliveries ever ask for attention", () => {
    const standings = registerStandings({ ...EMPTY, stopped: 2 });
    const asking = REGISTERS.filter((one) => standings[one].attention);
    expect(asking).toEqual(["deliveries"]);
  });

  test("an empty contributor list says which of the two reasons it is", () => {
    expect(nothingIn("contributors", { apps: [] })).toContain("app");
    expect(nothingIn("contributors", { apps: [app({})] })).not.toContain(
      "Add an app",
    );
  });

  test("every register says something when it is empty", () => {
    for (const register of REGISTERS) {
      expect(nothingIn(register, { apps: [] }).length).toBeGreaterThan(0);
    }
  });

  test("an empty delivery view says what was looked for", () => {
    expect(nothingDelivered("failed")).not.toBe(nothingDelivered("pending"));
    expect(nothingDelivered("").length).toBeGreaterThan(0);
  });
});

describe("what a contributor is allowed", () => {
  test("names the app, the categories and the one they start in", () => {
    const said = grantAllowance(
      grant({
        app: app({ name: "Beacon" }),
        categories: [
          category({ label: "News" }),
          category({ id: "cat-2", label: "Release notes" }),
        ],
        defaultCategory: category({ id: "cat-2", label: "Release notes" }),
      }),
    );
    expect(said).toContain("Beacon");
    expect(said).toContain("News");
    expect(said).toContain("Release notes by default");
  });

  test("marks a retired app so a stale approval is visible", () => {
    expect(grantAllowance(grant({ app: app({ retired: true }) }))).toContain(
      "retired",
    );
  });
});

describe("how a destination stands", () => {
  test("a rotation in its overlap outranks everything else", () => {
    const said = destinationStanding(
      destination({
        previousSecretUntil: new Date(Date.now() + 3600_000).toISOString(),
      }),
    );
    expect(said).toContain("Both signing secrets");
  });

  test("a spent overlap is not mentioned", () => {
    const said = destinationStanding(
      destination({ previousSecretUntil: "2026-01-01T00:00:00Z" }),
    );
    expect(said).not.toContain("Both signing secrets");
  });

  test("a destination waiting to prove itself says nothing is sent", () => {
    const said = destinationStanding(
      destination({ state: "unverified", verifiedAt: null }),
    );
    expect(said).toContain("Nothing is sent");
  });

  test("a switched-off destination says how to start it again", () => {
    expect(destinationStanding(destination({ state: "disabled" }))).toContain(
      "Verify it again",
    );
  });

  test("a channel says the name Discord announces under", () => {
    const said = destinationStanding(
      destination({
        kind: "discord",
        channel: {
          guildId: "1",
          channelId: "2",
          webhookName: "Illarin",
          roleId: "",
          roleName: "",
        },
      }),
    );
    expect(said).toContain("Illarin");
  });

  test("a webhook takes the events it asked for, and a channel takes one", () => {
    expect(
      destinationTakes(
        destination({
          events: [
            "publication.post.published.v1",
            "publication.post.withdrawn.v1",
          ],
        }),
      ),
    ).toBe("Publication · Takedown");
    expect(destinationTakes(destination({ events: [] }))).toContain("nothing");
    expect(
      destinationTakes(
        destination({
          kind: "discord",
          channel: {
            guildId: "1",
            channelId: "2",
            webhookName: "Illarin",
            roleId: "9",
            roleName: "Blog readers",
          },
        }),
      ),
    ).toContain("@Blog readers");
  });

  test("an active destination is switched off rather than verified", () => {
    expect(destinationActions(destination({ state: "active" }))).toEqual({
      rotate: true,
      switchOff: true,
      verify: false,
    });
  });

  test("a waiting destination is verified rather than switched off", () => {
    expect(destinationActions(destination({ state: "unverified" }))).toEqual({
      rotate: true,
      switchOff: false,
      verify: true,
    });
  });

  test("a channel has no signing secret to rotate", () => {
    expect(
      destinationActions(
        destination({
          kind: "discord",
          channel: {
            guildId: "1",
            channelId: "2",
            webhookName: "Illarin",
            roleId: "",
            roleName: "",
          },
        }),
      ).rotate,
    ).toBe(false);
  });
});

describe("sending an announcement again", () => {
  test("one Illarin gave up on can be sent again", () => {
    expect(
      canReplay(
        delivery({ state: "failed", settledReason: "exhausted", attempts: 6 }),
      ),
    ).toBe(true);
  });

  test("one Illarin stopped on purpose cannot", () => {
    expect(
      canReplay(delivery({ state: "failed", settledReason: "disabled" })),
    ).toBe(false);
  });

  test("a removed destination cannot be reached again", () => {
    expect(
      canReplay(
        delivery({
          state: "failed",
          settledReason: "exhausted",
          removed: true,
        }),
      ),
    ).toBe(false);
  });

  test("nothing that arrived or is still going is sent again", () => {
    expect(canReplay(delivery({ state: "delivered" }))).toBe(false);
    expect(canReplay(delivery({ state: "pending" }))).toBe(false);
    expect(canReplay(delivery({ state: "unconfirmed" }))).toBe(false);
  });
});

describe("what a token says about itself", () => {
  test("an unused token says so rather than leaving a gap", () => {
    expect(tokenStanding(token({}))).toContain("never used");
  });

  test("a live token with an end date says when", () => {
    expect(
      tokenStanding(token({ expiresAt: "2026-12-01T23:59:59Z" })),
    ).toContain("expires");
  });

  test("a spent token does not advertise an expiry it never reached", () => {
    expect(
      tokenStanding(
        token({
          active: false,
          expiresAt: "2026-12-01T23:59:59Z",
          revokedAt: "2026-09-01T09:00:00Z",
        }),
      ),
    ).not.toContain("expires");
  });

  test("a spent token says which way it ended", () => {
    expect(
      tokenEnded(token({ active: false, revokedAt: "2026-09-01T09:00:00Z" })),
    ).toContain("Revoked");
    expect(
      tokenEnded(token({ active: false, expiresAt: "2026-09-01T09:00:00Z" })),
    ).toContain("Expired");
    expect(tokenEnded(token({ active: false }))).toBe("Spent");
  });
});
