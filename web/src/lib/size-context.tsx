"use client";

import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

type SizeVariant = "default" | "compact";

/** SizeClasses are one step of the control ladder: 36px or 28px for a mouse, 44px for a coarse pointer. */
interface SizeClasses {
  variant: SizeVariant;
  control: string;
  segmentItem: string;
  segmentPad: string;
  text: string;
  px: string;
  itemPx: string;
  gap: string;
  icon: number;
}

const sizeMap: Record<SizeVariant, SizeClasses> = {
  default: {
    variant: "default",
    control: "h-control",
    segmentItem: "h-segment",
    segmentPad: "p-1",
    text: "text-ui",
    px: "px-3",
    itemPx: "px-2",
    gap: "gap-2",
    icon: 16,
  },
  compact: {
    variant: "compact",
    control: "h-control-compact",
    segmentItem: "h-segment-compact",
    segmentPad: "p-0.5",
    text: "text-meta",
    px: "px-2.5",
    itemPx: "px-1.5",
    gap: "gap-1",
    icon: 14,
  },
};

interface SizeContextValue {
  size: SizeVariant;
  setSize: (size: SizeVariant) => void;
  classes: SizeClasses;
}

const SizeContext = createContext<SizeContextValue | null>(null);

/** useSizeVariant resolves the size from the prop, then the nearest provider, then default. */
function useSizeVariant(override?: SizeVariant | null): SizeVariant {
  const ctx = useContext(SizeContext);
  return override ?? ctx?.size ?? "default";
}

function useSize(override?: SizeVariant | null): SizeClasses {
  return sizeMap[useSizeVariant(override)];
}

/** SizeProvider pins every control inside it to one step of the ladder. */
function SizeProvider({
  children,
  size,
  defaultSize = "default",
}: {
  children: ReactNode;
  size?: SizeVariant;
  defaultSize?: SizeVariant;
}) {
  const [internalSize, setInternalSize] = useState<SizeVariant>(defaultSize);
  const isControlled = size !== undefined;
  const resolved = size ?? internalSize;
  const setSize = useCallback(
    (next: SizeVariant) => {
      if (!isControlled) setInternalSize(next);
    },
    [isControlled],
  );
  const value = useMemo(
    () => ({ size: resolved, setSize, classes: sizeMap[resolved] }),
    [resolved, setSize],
  );
  return <SizeContext.Provider value={value}>{children}</SizeContext.Provider>;
}

export { SizeProvider, sizeMap, useSize, useSizeVariant };
export type { SizeClasses, SizeVariant };
