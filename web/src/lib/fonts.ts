import { DM_Sans, Outfit } from "next/font/google";

const outfit = Outfit({
  variable: "--font-outfit",
  subsets: ["latin"],
});

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  style: ["normal", "italic"],
});

/** The two typefaces, named on the document so every token that reads them resolves. */
export const FONT_VARIABLES = `${outfit.variable} ${dmSans.variable}`;
