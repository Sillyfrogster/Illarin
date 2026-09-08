import type { Metadata, Viewport } from "next";
import { DM_Sans, Outfit } from "next/font/google";
import { ArtFilters } from "@/components/art/ArtFilters";
import {
  SITE_DESCRIPTION,
  SITE_NAME,
  siteOpenGraph,
  siteTwitter,
  siteUrl,
} from "@/lib/site-metadata";
import { THEME_BOOTSTRAP_SCRIPT } from "@/lib/theme";
import { Providers } from "./providers";
import "./globals.css";

const outfit = Outfit({
  variable: "--font-outfit",
  subsets: ["latin"],
});

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  style: ["normal", "italic"],
});

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: { default: SITE_NAME, template: `%s \u00b7 ${SITE_NAME}` },
  description: SITE_DESCRIPTION,
  applicationName: SITE_NAME,
  openGraph: {
    ...siteOpenGraph(),
    title: SITE_NAME,
    description: SITE_DESCRIPTION,
    url: "/",
  },
  twitter: {
    ...siteTwitter(),
    title: SITE_NAME,
    description: SITE_DESCRIPTION,
  },
};

/** The browser chrome follows the reader's theme */
export const viewport: Viewport = {
  colorScheme: "light dark",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#080a0c" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${outfit.variable} ${dmSans.variable}`}
      suppressHydrationWarning
    >
      <head>
        <script>{THEME_BOOTSTRAP_SCRIPT}</script>
      </head>
      <body>
        <ArtFilters />
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
