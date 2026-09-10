import type { ComponentPropsWithoutRef, ElementType, ReactNode } from "react";
import styles from "./Shell.module.css";

type ShellProps = ComponentPropsWithoutRef<"div"> & {
  children: ReactNode;
  as?: ElementType;
};

/** Holds page content to a fixed width */
export function Shell({
  children,
  as: Tag = "div",
  className,
  ...rest
}: ShellProps) {
  return (
    <Tag
      className={className ? `${styles.shell} ${className}` : styles.shell}
      {...rest}
    >
      {children}
    </Tag>
  );
}
