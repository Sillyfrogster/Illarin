import type { Metadata } from "next";
import { DM_Sans, Outfit } from "next/font/google";
import { notFound } from "next/navigation";
import { DirectionPrototype } from "./workspace";
import "./theme.css";

const outfit = Outfit({ variable: "--font-outfit", subsets: ["latin"] });
const body = DM_Sans({ variable: "--font-dm-sans", subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Visual direction study",
  robots: { index: false, follow: false },
};

export default function DirectionPrototypePage() {
  if (process.env.NODE_ENV === "production") notFound();
  return (
    <div className={`${outfit.variable} ${body.variable}`}>
      <DirectionPrototype />
    </div>
  );
}
