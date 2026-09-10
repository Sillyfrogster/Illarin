import { describe, expect, test } from "bun:test";
import type { Distinction, DistinctionAssignment } from "@/lib/api/query";
import {
  heldBy,
  movedAssignments,
  movedDefinitions,
  nothingIn,
  offeredTo,
  RECOGNITION_REGISTERS,
  recognitionCells,
  registerHolds,
  unexplained,
  whatIsHeld,
} from "./recognition-register";

function definition(shape: Partial<Distinction>): Distinction {
  return {
    explanation: "Published a first asset",
    form: "badge",
    id: "one",
    name: "First light",
    position: 0,
    retired: false,
    ...shape,
  };
}

function assignment(
  shape: Partial<DistinctionAssignment>,
): DistinctionAssignment {
  return {
    active: true,
    assignedAt: "2026-08-01T10:00:00Z",
    distinction: definition({}),
    id: "assignment-one",
    position: 0,
    source: "manual",
    ...shape,
  };
}

describe("the registers", () => {
  test("counts what each register still gives out", () => {
    const cells = recognitionCells([
      definition({ id: "a" }),
      definition({ form: "title", id: "b" }),
      definition({ id: "c", retired: true }),
      definition({ form: "position", id: "d" }),
      definition({ form: "position", id: "e", retired: true }),
    ]);

    expect(cells.map((cell) => cell.id)).toEqual(RECOGNITION_REGISTERS);
    expect(cells.map((cell) => cell.count)).toEqual([2, 1, null]);
    expect(cells.every((cell) => cell.attention)).toBe(false);
  });

  test("titles and badges are one register, positions another", () => {
    const badge = definition({ id: "a" });
    const title = definition({ form: "title", id: "b" });
    const job = definition({ form: "position", id: "c" });
    const gone = definition({ form: "title", id: "d", retired: true });
    const held = [title, job, gone, badge];

    expect(registerHolds("titles", held).current).toEqual([badge, title]);
    expect(registerHolds("titles", held).retired).toEqual([gone]);
    expect(registerHolds("positions", held).current).toEqual([job]);
    expect(registerHolds("positions", held).retired).toEqual([]);
  });

  test("an empty register says what to do about it", () => {
    expect(nothingIn("titles")).toContain("mark");
    expect(nothingIn("positions")).toContain("job");
    expect(nothingIn("accounts")).toContain("handle");
  });

  test("a title nobody is told the reason for is worth flagging", () => {
    expect(unexplained(definition({ explanation: "" }))).toBe(true);
    expect(unexplained(definition({ explanation: "Ten uploads" }))).toBe(false);
    expect(unexplained(definition({ explanation: "", form: "position" }))).toBe(
      false,
    );
  });
});

describe("moving a definition", () => {
  const badgeOne = definition({ id: "b1", position: 0 });
  const badgeTwo = definition({ id: "b2", position: 1 });
  const retiredBadge = definition({ id: "b3", position: 2, retired: true });
  const titleOne = definition({ form: "title", id: "t1", position: 0 });
  const held = [badgeOne, badgeTwo, retiredBadge, titleOne];

  test("moves it past its own form, carrying the retired ones along", () => {
    expect(movedDefinitions(held, badgeOne, 1)).toEqual(["b2", "b1", "b3"]);
  });

  test("refuses to move it past either end of its own form", () => {
    expect(movedDefinitions(held, badgeOne, -1)).toBeNull();
    expect(movedDefinitions(held, badgeTwo, 1)).toBeNull();
    expect(movedDefinitions(held, titleOne, 1)).toBeNull();
  });
});

describe("what one account holds", () => {
  const job = assignment({
    distinction: definition({ form: "position", id: "d1", name: "Developer" }),
    id: "a1",
  });
  const badge = assignment({
    distinction: definition({ id: "d2", name: "First light" }),
    id: "a2",
  });
  const title = assignment({
    distinction: definition({ form: "title", id: "d3", name: "Archivist" }),
    id: "a3",
  });
  const takenBack = assignment({
    active: false,
    distinction: definition({ id: "d4", name: "Quiet fix" }),
    id: "a4",
  });
  const holds = [job, badge, title, takenBack];

  test("splits jobs from what was earned, and both from what was taken back", () => {
    expect(heldBy(holds)).toEqual({
      earned: [badge, title],
      past: [takenBack],
      positions: [job],
    });
  });

  test("says what they hold in one line", () => {
    expect(whatIsHeld(holds)).toBe("1 job · 2 earned");
    expect(whatIsHeld([job, assignment({ id: "a5" })])).toBe(
      "1 job · 1 earned",
    );
    expect(whatIsHeld([takenBack])).toBe("Nothing on their profile yet");
  });

  test("offers only what is still given out and not already held", () => {
    const badgeDefinition = definition({ id: "d2", name: "First light" });
    const spare = definition({ id: "d5", name: "Tenfold" });
    const gone = definition({ id: "d6", retired: true });

    expect(offeredTo([badgeDefinition, spare, gone], holds)).toEqual([spare]);
  });

  test("moves an earned one past its neighbour, keeping jobs in the order", () => {
    expect(movedAssignments(holds, badge, 1)).toEqual(["a1", "a3", "a2"]);
    expect(movedAssignments(holds, badge, -1)).toBeNull();
    expect(movedAssignments(holds, title, 1)).toBeNull();
  });
});
