"use client";

import * as SheetPrimitive from "@radix-ui/react-dialog";
import { cva, type VariantProps } from "class-variance-authority";
import { X } from "lucide-react";
import * as React from "react";

import { cn } from "@/utils/tailwind";

/**
 * Sheet — slide-in panel from any edge (shadcn/ui standard).
 * Built on top of @radix-ui/react-dialog.
 *
 * Two layout modes:
 *   - Default: portals to <body>, uses `position: fixed`, overlays the
 *     entire viewport.
 *   - Contained (`container` prop set): portals to that DOM node, uses
 *     `position: absolute`, constrained to the container's box. Used by
 *     ChatView to keep the conversations slide-in inside the chat pane
 *     rather than overlaying the whole workspace window.
 */

const Sheet = SheetPrimitive.Root;
const SheetTrigger = SheetPrimitive.Trigger;
const SheetClose = SheetPrimitive.Close;

interface SheetPortalProps {
  container?: HTMLElement | null;
  children: React.ReactNode;
}

function SheetPortal({
  container,
  children,
}: SheetPortalProps): React.JSX.Element {
  if (container) {
    return (
      <SheetPrimitive.Portal container={container}>
        {children}
      </SheetPrimitive.Portal>
    );
  }
  return <SheetPrimitive.Portal>{children}</SheetPrimitive.Portal>;
}

interface SheetOverlayProps
  extends React.ComponentPropsWithoutRef<typeof SheetPrimitive.Overlay> {
  container?: HTMLElement | null;
}

const SheetOverlay = React.forwardRef<
  React.ElementRef<typeof SheetPrimitive.Overlay>,
  SheetOverlayProps
>(({ className, ...props }, ref) => {
  const isContained =
    (props as Record<string, unknown>)["data-contained"] === "true";
  return (
    <SheetPrimitive.Overlay
      ref={ref}
      className={cn(
        isContained
          ? "absolute inset-0 z-40 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
          : "fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
        className,
      )}
      {...props}
    />
  );
});
SheetOverlay.displayName = SheetPrimitive.Overlay.displayName;

const sheetVariants = cva(
  "z-50 gap-4 bg-card shadow-lg transition ease-in-out data-[state=closed]:duration-300 data-[state=open]:duration-500 data-[state=open]:animate-in data-[state=closed]:animate-out",
  {
    variants: {
      side: {
        top: "inset-x-0 top-0 border-b data-[state=closed]:slide-out-to-top data-[state=open]:slide-in-from-top",
        bottom:
          "inset-x-0 bottom-0 border-t data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom",
        left: "inset-y-0 left-0 h-full w-3/4 border-r data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left sm:max-w-sm",
        right:
          "inset-y-0 right-0 h-full w-3/4 border-l data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right sm:max-w-sm",
      },
      position: {
        viewport: "fixed",
        contained: "absolute",
      },
    },
    defaultVariants: { side: "right", position: "viewport" },
  },
);

interface SheetContentProps
  extends React.ComponentPropsWithoutRef<typeof SheetPrimitive.Content>,
    VariantProps<typeof sheetVariants> {
  container?: HTMLElement | null;
}

const SheetContent = React.forwardRef<
  React.ElementRef<typeof SheetPrimitive.Content>,
  SheetContentProps
>(
  (
    { side = "right", position, container, className, children, ...props },
    ref,
  ) => (
    <SheetPortal container={container}>
      <SheetOverlay
        data-contained={position === "contained" ? "true" : "false"}
      />
      <SheetPrimitive.Content
        ref={ref}
        className={cn(sheetVariants({ side, position }), className)}
        {...props}
      >
        {children}
        <SheetPrimitive.Close className="absolute right-3 top-3 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none">
          <X className="h-4 w-4" aria-hidden="true" />
          <span className="sr-only">Close</span>
        </SheetPrimitive.Close>
      </SheetPrimitive.Content>
    </SheetPortal>
  ),
);
SheetContent.displayName = SheetPrimitive.Content.displayName;

function SheetHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>): React.JSX.Element {
  return (
    <div
      className={cn(
        "flex flex-col space-y-2 text-center sm:text-left",
        className,
      )}
      {...props}
    />
  );
}
SheetHeader.displayName = "SheetHeader";

function SheetFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>): React.JSX.Element {
  return (
    <div
      className={cn(
        "flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2",
        className,
      )}
      {...props}
    />
  );
}
SheetFooter.displayName = "SheetFooter";

const SheetTitle = React.forwardRef<
  React.ElementRef<typeof SheetPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof SheetPrimitive.Title>
>(({ className, ...props }, ref) => (
  <SheetPrimitive.Title
    ref={ref}
    className={cn("text-lg font-semibold text-foreground", className)}
    {...props}
  />
));
SheetTitle.displayName = SheetPrimitive.Title.displayName;

const SheetDescription = React.forwardRef<
  React.ElementRef<typeof SheetPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof SheetPrimitive.Description>
>(({ className, ...props }, ref) => (
  <SheetPrimitive.Description
    ref={ref}
    className={cn("text-sm text-muted-foreground", className)}
    {...props}
  />
));
SheetDescription.displayName = SheetPrimitive.Description.displayName;

export {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetOverlay,
  SheetPortal,
  SheetTitle,
  SheetTrigger,
};
