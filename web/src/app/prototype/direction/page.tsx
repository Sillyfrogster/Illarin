import type { Metadata } from "next";
import { Newsreader } from "next/font/google";
import { notFound } from "next/navigation";
import { DirectionPrototype } from "./workspace";
import "./theme.css";

const newsreader = Newsreader({
  variable: "--font-newsreader",
  subsets: ["latin"],
  style: ["normal", "italic"],
});

export const metadata: Metadata = {
  title: "Visual direction study",
  robots: { index: false, follow: false },
};

export default function DirectionPrototypePage() {
  if (process.env.NODE_ENV === "production") notFound();
  return (
    <div className={newsreader.variable}>
      <DirectionPrototype />
    </div>
  );
}
