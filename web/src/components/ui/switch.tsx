"use client";

import * as SwitchPrimitive from "@radix-ui/react-switch";
import { animate, motion, useMotionValue } from "framer-motion";
import {
  type HTMLAttributes,
  type ReactNode,
  type Ref,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { cn, focusRing } from "@/lib/cn";
import { type SizeVariant, useSize } from "@/lib/size-context";
import { spring } from "@/lib/springs";

interface SwitchProps extends HTMLAttributes<HTMLLabelElement> {
  ref?: Ref<HTMLLabelElement>;
  label: ReactNode;
  hint?: ReactNode;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  size?: SizeVariant;
}

const METRICS = {
  default: {
    trackWidth: 34,
    trackHeight: 20,
    thumbSize: 16,
    pillExtend: 2,
    pressExtend: 4,
    pressShrink: 4,
  },
  compact: {
    trackWidth: 28,
    trackHeight: 16,
    thumbSize: 12,
    pillExtend: 2,
    pressExtend: 3,
    pressShrink: 3,
  },
} as const;

const THUMB_OFFSET = 2;
const DRAG_DEAD_ZONE = 2;

/** Switch is Fluid Functionalism's switch: on or off at once when pressed, with a thumb that stretches under the pointer and can be dragged. */
function Switch({
  label,
  hint,
  checked,
  onCheckedChange,
  disabled = false,
  size,
  className,
  ref,
  ...props
}: SwitchProps) {
  const labelId = useId();
  const hintId = useId();
  const controlId = useId();
  const hasMounted = useRef(false);
  const [hovered, setHovered] = useState(false);
  const [pressed, setPressed] = useState(false);
  const sizeClasses = useSize(size);
  const m = METRICS[sizeClasses.variant];
  const thumbTravel = m.trackWidth - m.thumbSize - THUMB_OFFSET * 2;

  const dragging = useRef(false);
  const didDrag = useRef(false);
  const pointerStart = useRef<{
    clientX: number;
    originX: number;
  } | null>(null);

  const motionX = useMotionValue(
    checked ? THUMB_OFFSET + thumbTravel : THUMB_OFFSET,
  );

  useEffect(() => {
    hasMounted.current = true;
  }, []);

  const thumbWidth = pressed
    ? m.thumbSize + m.pressExtend
    : hovered
      ? m.thumbSize + m.pillExtend
      : m.thumbSize;
  const thumbHeight = pressed ? m.thumbSize - m.pressShrink : m.thumbSize;
  const thumbY = pressed ? THUMB_OFFSET + m.pressShrink / 2 : THUMB_OFFSET;
  const extraWidth = thumbWidth - m.thumbSize;
  const thumbX = checked
    ? THUMB_OFFSET + thumbTravel - extraWidth
    : THUMB_OFFSET;

  useEffect(() => {
    if (dragging.current) return;
    if (!hasMounted.current) {
      motionX.set(thumbX);
    } else {
      animate(motionX, thumbX, spring.moderate);
    }
  }, [thumbX, motionX]);

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLLabelElement>) => {
      if (disabled) return;
      if (e.pointerType === "mouse" && e.button !== 0) return;
      setPressed(true);
      dragging.current = false;
      didDrag.current = false;
      pointerStart.current = {
        clientX: e.clientX,
        originX: motionX.get(),
      };
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    },
    [disabled, motionX],
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent<HTMLLabelElement>) => {
      if (!pointerStart.current) return;
      const delta = e.clientX - pointerStart.current.clientX;

      if (!dragging.current) {
        if (Math.abs(delta) < DRAG_DEAD_ZONE) return;
        dragging.current = true;
      }

      const dragMin = THUMB_OFFSET;
      const pressedThumbWidth = m.thumbSize + m.pressExtend;
      const dragMax = m.trackWidth - THUMB_OFFSET - pressedThumbWidth;
      const rawX = pointerStart.current.originX + delta;
      motionX.set(Math.max(dragMin, Math.min(dragMax, rawX)));
    },
    [motionX, m],
  );

  const handlePointerUp = useCallback(() => {
    if (!pointerStart.current) return;
    setPressed(false);

    if (dragging.current) {
      didDrag.current = true;
      dragging.current = false;

      const currentX = motionX.get();
      const dragMin = THUMB_OFFSET;
      const pressedThumbWidth = m.thumbSize + m.pressExtend;
      const dragMax = m.trackWidth - THUMB_OFFSET - pressedThumbWidth;
      const midpoint = (dragMin + dragMax) / 2;

      const shouldBeOn = currentX > midpoint;

      if (shouldBeOn !== checked) {
        onCheckedChange(!checked);
      } else {
        const snapTarget = checked ? THUMB_OFFSET + thumbTravel : THUMB_OFFSET;
        animate(motionX, snapTarget, spring.moderate);
      }

      requestAnimationFrame(() => {
        didDrag.current = false;
      });
    }

    pointerStart.current = null;
  }, [checked, onCheckedChange, motionX, m, thumbTravel]);

  const handlePointerCancel = useCallback(() => {
    if (!pointerStart.current) return;
    setPressed(false);

    if (dragging.current) {
      dragging.current = false;
      const snapTarget = checked ? THUMB_OFFSET + thumbTravel : THUMB_OFFSET;
      animate(motionX, snapTarget, spring.moderate);
    }

    pointerStart.current = null;
  }, [checked, motionX, thumbTravel]);

  return (
    <label
      htmlFor={controlId}
      ref={ref}
      className={cn(
        "relative z-10 flex items-start cursor-pointer select-none touch-none font-ui",
        sizeClasses.gap,
        sizeClasses.variant === "compact" ? "py-1" : "py-2",
        disabled && "opacity-50 pointer-events-none",
        className,
      )}
      onPointerEnter={(e) => {
        if (e.pointerType === "mouse") setHovered(true);
      }}
      onPointerLeave={() => setHovered(false)}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerCancel}
      {...props}
    >
      <SwitchPrimitive.Root
        checked={checked}
        aria-describedby={hint ? hintId : undefined}
        aria-labelledby={labelId}
        id={controlId}
        onCheckedChange={() => {
          if (didDrag.current) return;
          onCheckedChange(!checked);
        }}
        disabled={disabled}
        className={cn(
          "relative mt-px shrink-0 cursor-pointer rounded-full transition-colors duration-80",
          focusRing,
        )}
        style={{
          width: m.trackWidth,
          height: m.trackHeight,
          backgroundColor: checked
            ? hovered
              ? "color-mix(in oklab, var(--v-action) 88%, var(--v-field))"
              : "var(--v-action)"
            : hovered
              ? "color-mix(in oklab, var(--v-ink) 32%, transparent)"
              : "color-mix(in oklab, var(--v-ink) 22%, transparent)",
        }}
      >
        <SwitchPrimitive.Thumb asChild>
          <motion.span
            className="absolute top-0 left-0 block rounded-full bg-[#fff] shadow-[0_1px_2px_rgb(0_0_0/0.2)]"
            initial={false}
            style={{ x: motionX }}
            animate={{
              y: thumbY,
              width: thumbWidth,
              height: thumbHeight,
            }}
            transition={hasMounted.current ? spring.moderate : { duration: 0 }}
          />
        </SwitchPrimitive.Thumb>
      </SwitchPrimitive.Root>

      <span className="min-w-0">
        <span className={cn("block text-ink", sizeClasses.text)} id={labelId}>
          {label}
        </span>
        {hint ? (
          <span className="mt-0.5 block text-meta text-mute" id={hintId}>
            {hint}
          </span>
        ) : null}
      </span>
    </label>
  );
}

export { Switch };
export type { SwitchProps };
