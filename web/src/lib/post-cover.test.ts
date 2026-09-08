import { expect, test } from "bun:test";
import type { PublicPost } from "@/lib/api/query";
import { postCover } from "./post-cover";

const PLACED = "6f2c1b40-9d38-4a7e-b512-0c8e37a41d59";

const POST = {
  header: { mediaId: PLACED, alt: "A desk with one draft on it" },
  media: [
    {
      id: PLACED,
      postId: "b8d0f1a2-3c45-4e67-89ab-cdef01234567",
      purpose: "header",
      url: `/media/${PLACED}/detail/1`,
      thumbUrl: `/media/${PLACED}/grid/1`,
      width: 1200,
      height: 900,
    },
  ],
} as unknown as PublicPost;

test("a post with a header picture offers it with the words the author wrote for it", () => {
  expect(postCover(POST)).toEqual({
    url: `/media/${PLACED}/detail/1`,
    alt: "A desk with one draft on it",
    width: 1200,
    height: 900,
  });
});

test("a post with no header picture offers nothing to stand in for one", () => {
  expect(postCover({ ...POST, header: null } as PublicPost)).toBeNull();
  expect(postCover(null)).toBeNull();
});

test("a header whose bytes are gone is not a cover", () => {
  const orphaned = {
    ...POST,
    header: { mediaId: "00000000-0000-4000-8000-000000000000", alt: "Gone" },
  } as unknown as PublicPost;
  expect(postCover(orphaned)).toBeNull();
});
