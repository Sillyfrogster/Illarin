"use client";

import {
  CircleCheck,
  Info,
  LoaderCircle,
  OctagonX,
  TriangleAlert,
} from "lucide-react";
import { type CSSProperties, useEffect, useState } from "react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

type Theme = NonNullable<ToasterProps["theme"]>;

/** useSiteTheme follows the appearance setting, which the site keeps on the root element. */
function useSiteTheme(): Theme {
  const [theme, setTheme] = useState<Theme>("system");
  useEffect(() => {
    const root = document.documentElement;
    const read = () => {
      const chosen = root.dataset.theme;
      setTheme(chosen === "light" || chosen === "dark" ? chosen : "system");
    };
    read();
    const observer = new MutationObserver(read);
    observer.observe(root, {
      attributes: true,
      attributeFilter: ["data-theme"],
    });
    return () => observer.disconnect();
  }, []);
  return theme;
}

/** Toaster shows results with Undo on the menu plane, sitting above the bars pinned to the bottom of a page. */
export function Toaster(props: ToasterProps) {
  return (
    <Sonner
      className="toaster group"
      icons={{
        success: <CircleCheck className="size-4" />,
        info: <Info className="size-4" />,
        warning: <TriangleAlert className="size-4" />,
        error: <OctagonX className="size-4" />,
        loading: <LoaderCircle className="size-4 animate-spin" />,
      }}
      mobileOffset={{ bottom: 112 }}
      offset={{ bottom: 128 }}
      position="bottom-center"
      style={
        {
          "--normal-bg": "var(--v-plane)",
          "--normal-text": "var(--v-ink)",
          "--normal-border": "transparent",
          "--error-bg": "var(--v-plane)",
          "--error-text": "var(--v-stop)",
          "--error-border": "transparent",
          "--border-radius": "12px",
        } as CSSProperties
      }
      theme={useSiteTheme()}
      toastOptions={{
        classNames: {
          toast: "font-ui text-ui shadow-popover ring-1 ring-ink/8",
          actionButton:
            "!h-control-compact !rounded-control !bg-action !px-3 !font-ui !text-meta !font-medium !text-on-accent hover:!bg-action-hover",
        },
      }}
      {...props}
    />
  );
}
