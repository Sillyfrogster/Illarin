import { notFound } from "next/navigation";

/** Any deeper path the blog does not know gets the blog's own not-found page. */
export default function NothingBeneath(): never {
  notFound();
}
