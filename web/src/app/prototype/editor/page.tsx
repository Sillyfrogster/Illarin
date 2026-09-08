import type { Metadata } from "next";
import { Newsreader } from "next/font/google";
import { notFound } from "next/navigation";
import { WorkspacePrototype } from "./workspace";
import "./theme.css";

const newsreader = Newsreader({
  variable: "--font-newsreader",
  subsets: ["latin"],
  style: ["normal", "italic"],
});

export const metadata: Metadata = {
  title: "Asset workspace prototype",
  robots: { index: false, follow: false },
};

export default function EditorPrototypePage() {
  if (process.env.NODE_ENV === "production") notFound();
  return (
    <div className={newsreader.variable}>
      <WorkspacePrototype />
    </div>
  );
}
