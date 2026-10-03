"use client";

import { useCallback, useEffect, useState } from "react";

export function useMeasuredWidth<T extends HTMLElement>() {
  const [node, setNode] = useState<T | null>(null);
  const [width, setWidth] = useState<number>();
  const ref = useCallback((element: T | null) => setNode(element), []);

  useEffect(() => {
    if (!node) return;

    const measure = () => {
      // The layout width, before any CSS zoom on an ancestor shrinks it on screen
      const zoom = (node as { currentCSSZoom?: number }).currentCSSZoom ?? 1;
      const nextWidth = Math.round(node.getBoundingClientRect().width / zoom);
      setWidth((current) => (current === nextWidth ? current : nextWidth));
    };
    const observer = new ResizeObserver(measure);
    measure();
    observer.observe(node);
    return () => observer.disconnect();
  }, [node]);

  return [ref, width] as const;
}
