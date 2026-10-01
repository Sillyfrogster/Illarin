import { Onest } from "next/font/google";

const onest = Onest({
  variable: "--font-onest",
  subsets: ["latin"],
});

export const FONT_VARIABLES = onest.variable;
