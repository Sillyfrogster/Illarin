import type { Metadata, Viewport } from "next";
import { ArtFilters } from "@/components/art/ArtFilters";
import { AnalyticsScript } from "@/components/layout/AnalyticsScript";
import { StoredTheme } from "@/components/layout/StoredTheme";
import { FONT_VARIABLES } from "@/lib/fonts";
import { siteAddress } from "@/lib/site-address";
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

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: { default: SITE_NAME, template: `%s \u00b7 ${SITE_NAME}` },
  description: SITE_DESCRIPTION,
  applicationName: SITE_NAME,
  manifest: "/site.webmanifest",
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

export const viewport: Viewport = {
  colorScheme: "light dark",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#0a0a0a" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={FONT_VARIABLES} suppressHydrationWarning>
      <head>
        <script>{THEME_BOOTSTRAP_SCRIPT}</script>
        <AnalyticsScript />
      </head>
      <body>
        <StoredTheme />
        <ArtFilters />
        <Providers origins={{ site: siteAddress("/") }}>{children}</Providers>
      </body>
    </html>
  );
}
