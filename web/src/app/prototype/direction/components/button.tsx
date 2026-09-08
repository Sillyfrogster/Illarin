import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../ui";

const buttonVariants = cva(
  "vd:inline-flex vd:items-center vd:justify-center vd:gap-2 vd:whitespace-nowrap vd:rounded-control vd:text-sm vd:font-medium vd:transition-[background-color,color,transform] vd:active:translate-y-px vd:motion-reduce:transform-none vd:focus-visible:outline-none vd:focus-visible:ring-1 vd:focus-visible:ring-ring vd:disabled:pointer-events-none vd:disabled:opacity-50 vd:[&_svg]:pointer-events-none vd:[&_svg]:size-4 vd:[&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "vd:bg-primary vd:text-primary-foreground vd:hover:bg-primary/90",
        destructive:
          "vd:bg-destructive vd:text-destructive-foreground vd:shadow-sm vd:hover:bg-destructive/90",
        outline:
          "vd:border vd:border-input vd:bg-background vd:shadow-sm vd:hover:bg-accent vd:hover:text-accent-foreground",
        secondary:
          "vd:bg-secondary vd:text-secondary-foreground vd:hover:bg-secondary/80",
        ghost: "vd:hover:bg-accent vd:hover:text-accent-foreground",
        link: "vd:text-primary vd:underline-offset-4 vd:hover:underline",
      },
      size: {
        default: "vd:h-11 vd:px-4 vd:py-2",
        sm: "vd:h-11 vd:rounded-control vd:px-3 vd:text-xs",
        lg: "vd:h-12 vd:rounded-control vd:px-8",
        icon: "vd:h-11 vd:w-11",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
