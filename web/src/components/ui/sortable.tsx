"use client";

import {
  type Announcements,
  closestCenter,
  DndContext,
  type DragEndEvent,
  type DraggableAttributes,
  type DraggableSyntheticListeners,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  type UniqueIdentifier,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  restrictToParentElement,
  restrictToVerticalAxis,
} from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Slot } from "@radix-ui/react-slot";
import { GripVertical } from "lucide-react";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
  useId,
  useMemo,
} from "react";
import { cn } from "@/lib/cn";

const MODIFIERS = [restrictToVerticalAxis, restrictToParentElement];

type Item = {
  attributes: DraggableAttributes;
  dragging: boolean;
  id: string;
  listeners: DraggableSyntheticListeners;
  setActivatorNodeRef: (node: HTMLElement | null) => void;
};

const ItemContext = createContext<Item | null>(null);

function useItem(): Item {
  const item = useContext(ItemContext);
  if (!item) throw new Error("A sortable handle belongs inside SortableItem.");
  return item;
}

/** Sortable lets a person drag its items into a new order, or move them with the keyboard from a handle. */
export function Sortable({
  children,
  disabled = false,
  ids,
  labels,
  onMove,
}: {
  children: ReactNode;
  disabled?: boolean;
  ids: string[];
  labels: (id: UniqueIdentifier) => string;
  onMove: (from: number, to: number) => void;
}) {
  const id = useId();
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 4 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 180, tolerance: 6 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const announcements = useMemo<Announcements>(() => {
    const position = (of: UniqueIdentifier) => ids.indexOf(String(of)) + 1;
    return {
      onDragStart: ({ active }) =>
        `Picked up ${labels(active.id)}, ${position(active.id)} of ${ids.length}. Arrow keys move it, space drops it, escape puts it back.`,
      onDragOver: ({ active, over }) =>
        over
          ? `${labels(active.id)} is over position ${position(over.id)} of ${ids.length}.`
          : `${labels(active.id)} is outside the list.`,
      onDragEnd: ({ active, over }) =>
        over
          ? `${labels(active.id)} is now ${position(over.id)} of ${ids.length}.`
          : `${labels(active.id)} went back to where it was.`,
      onDragCancel: ({ active }) =>
        `${labels(active.id)} went back to where it was.`,
    };
  }, [ids, labels]);

  function onDragEnd({ active, over }: DragEndEvent) {
    if (!over || active.id === over.id) return;
    const from = ids.indexOf(String(active.id));
    const to = ids.indexOf(String(over.id));
    if (from !== -1 && to !== -1) onMove(from, to);
  }

  return (
    <DndContext
      accessibility={{
        announcements,
        screenReaderInstructions: {
          draggable:
            "Press space to pick this up. While holding it, the up and down arrows move it. Press space again to drop it, or escape to put it back.",
        },
      }}
      collisionDetection={closestCenter}
      id={id}
      modifiers={MODIFIERS}
      onDragEnd={onDragEnd}
      sensors={sensors}
    >
      <SortableContext
        disabled={disabled}
        items={ids}
        strategy={verticalListSortingStrategy}
      >
        {children}
      </SortableContext>
    </DndContext>
  );
}

/** SortableItem is one row. It moves with the drag, and its handle picks it up. */
export function SortableItem({
  asChild = false,
  className,
  disabled = false,
  id,
  style,
  ...props
}: ComponentProps<"li"> & {
  asChild?: boolean;
  disabled?: boolean;
  id: string;
}) {
  const {
    attributes,
    isDragging,
    listeners,
    setActivatorNodeRef,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ disabled, id });
  const Row = asChild ? Slot : "li";

  return (
    <ItemContext.Provider
      value={{
        attributes,
        dragging: isDragging,
        id,
        listeners,
        setActivatorNodeRef,
      }}
    >
      <Row
        className={cn("relative", isDragging && "z-10 opacity-85", className)}
        data-dragging={isDragging ? "" : undefined}
        ref={setNodeRef}
        style={{
          transform: CSS.Translate.toString(transform),
          transition,
          ...style,
        }}
        {...props}
      />
    </ItemContext.Provider>
  );
}

/** SortableItemHandle is the grip a person drags, or focuses to move the row with the keyboard. */
export function SortableItemHandle({
  className,
  disabled = false,
  label,
}: {
  className?: string;
  disabled?: boolean;
  label: string;
}) {
  const { attributes, dragging, listeners, setActivatorNodeRef } = useItem();

  return (
    <button
      aria-label={label}
      className={cn(
        "inline-flex size-11 shrink-0 cursor-grab touch-none items-center justify-center rounded-control text-mute outline-offset-3 select-none hover:bg-deep hover:text-ink disabled:cursor-default disabled:opacity-40 disabled:hover:bg-transparent",
        dragging && "cursor-grabbing",
        className,
      )}
      disabled={disabled}
      ref={setActivatorNodeRef}
      type="button"
      {...attributes}
      {...(disabled ? {} : listeners)}
    >
      <GripVertical aria-hidden="true" className="size-4" />
    </button>
  );
}

export { arrayMove };
