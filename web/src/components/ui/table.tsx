import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";

/** Table is a plain table in the site's type, ruled between rows, scrolling sideways inside its own box when too wide. */
function Table({ className, ...props }: ComponentProps<"table">) {
  return (
    <div className="w-full overflow-x-auto">
      <table
        className={cn("w-full border-collapse font-ui text-meta", className)}
        {...props}
      />
    </div>
  );
}

function TableHeader(props: ComponentProps<"thead">) {
  return <thead {...props} />;
}

function TableBody(props: ComponentProps<"tbody">) {
  return <tbody {...props} />;
}

function TableRow({ className, ...props }: ComponentProps<"tr">) {
  return (
    <tr
      className={cn("border-b border-rule last:border-b-0", className)}
      {...props}
    />
  );
}

function TableHead({ className, ...props }: ComponentProps<"th">) {
  return (
    <th
      className={cn(
        "px-3 py-2 text-left align-middle font-medium whitespace-nowrap text-mute",
        className,
      )}
      {...props}
    />
  );
}

function TableCell({ className, ...props }: ComponentProps<"td">) {
  return (
    <td
      className={cn(
        "px-3 py-2 align-middle whitespace-nowrap text-ink",
        className,
      )}
      {...props}
    />
  );
}

export { Table, TableBody, TableCell, TableHead, TableHeader, TableRow };
