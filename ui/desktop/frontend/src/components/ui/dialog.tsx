import { useDocumentScope } from "../../document-scope";
("use client");

import * as React from "react";
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { cn } from "cn";

import { Button } from "@/components/ui/button";
import {
  XIcon,
  CircleAlert,
  TriangleAlert,
  Info,
  CircleHelp,
} from "lucide-react";
import { DialogValidation, useDialogValidation } from "@/form-validation";
import { useDialogDrag } from "@/dialog-drag";
import DialogResizeHandle from "@/DialogResizeHandle";

function Dialog({
  lockWhileBusy = false,
  ...props
}: DialogPrimitive.Root.Props & { lockWhileBusy?: boolean }) {
  const scope = useDocumentScope();
  return (
    <DialogPrimitive.Root
      data-slot="dialog"
      {...props}
      modal={scope.busy ? false : props.modal}
      onOpenChange={(open, details) => {
        if (!open && scope.busy && lockWhileBusy) {
          details.cancel();
          return;
        }
        props.onOpenChange?.(open, details);
      }}
    />
  );
}

function DialogTrigger({ ...props }: DialogPrimitive.Trigger.Props) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

function DialogPortal({ ...props }: DialogPrimitive.Portal.Props) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />;
}

function DialogClose({ ...props }: DialogPrimitive.Close.Props) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

function DialogOverlay({
  className,
  ...props
}: DialogPrimitive.Backdrop.Props) {
  return (
    <DialogPrimitive.Backdrop
      data-slot="dialog-overlay"
      className={cn(
        "fixed inset-0 isolate z-50 bg-black/10 duration-100 supports-backdrop-filter:backdrop-blur-xs data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0",
        className,
      )}
      {...props}
    />
  );
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  resizable = true,
  tone,
  ref: forwardedRef,
  ...props
}: DialogPrimitive.Popup.Props & {
  showCloseButton?: boolean;
  resizable?: boolean;
  tone?: "error" | "warning" | "info" | "confirmation";
}) {
  const drag = useDialogDrag();
  const dialogRef = React.useCallback(
    (node: HTMLDivElement | null) => {
      drag.ref.current = node;
      if (typeof forwardedRef === "function") {
        const cleanup = forwardedRef(node);
        if (typeof cleanup === "function")
          return () => {
            drag.ref.current = null;
            cleanup();
          };
      } else if (forwardedRef) forwardedRef.current = node;
    },
    [drag.ref, forwardedRef],
  );
  const documentScope = useDocumentScope();
  if (!documentScope.active) return null;
  return (
    <DialogValidation>
      <DialogPortal>
        {!documentScope.busy && <DialogOverlay />}
        <DialogPrimitive.Popup
          data-slot="dialog-content"
          data-message-tone={tone}
          className={cn("studio-window", tone && "message-dialog", className)}
          {...props}
          ref={dialogRef}
          onPointerDown={(e) => {
            props.onPointerDown?.(e);
            if (!e.defaultPrevented) drag.onPointerDown(e);
          }}
          onPointerMove={(e) => {
            props.onPointerMove?.(e);
            if (!e.defaultPrevented) drag.onPointerMove(e);
          }}
          onPointerUp={(e) => {
            props.onPointerUp?.(e);
            drag.onPointerUp(e);
          }}
          onPointerCancel={(e) => {
            props.onPointerCancel?.(e);
            drag.onPointerCancel(e);
          }}
          onLostPointerCapture={(e) => {
            props.onLostPointerCapture?.(e);
            drag.onLostPointerCapture(e);
          }}
          onDoubleClick={(e) => {
            props.onDoubleClick?.(e);
            if (!e.defaultPrevented) drag.onDoubleClick(e);
          }}
        >
          {tone && (
            <div className="message-dialog-icon" aria-hidden="true">
              {tone === "error" ? (
                <CircleAlert />
              ) : tone === "warning" ? (
                <TriangleAlert />
              ) : tone === "confirmation" ? (
                <CircleHelp />
              ) : (
                <Info />
              )}
            </div>
          )}
          {children}
          {resizable && <DialogResizeHandle dialog={drag.ref} />}
          {showCloseButton && (
            <DialogPrimitive.Close
              data-slot="dialog-close"
              render={
                <Button
                  variant="ghost"
                  className="absolute top-2 right-2"
                  size="icon-sm"
                />
              }
            >
              <XIcon />
              <span className="sr-only">Close</span>
            </DialogPrimitive.Close>
          )}
        </DialogPrimitive.Popup>
      </DialogPortal>
    </DialogValidation>
  );
}

function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-header"
      className={cn("studio-window-titlebar", className)}
      {...props}
    />
  );
}

function DialogFooter({
  className,
  showCloseButton = false,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  showCloseButton?: boolean;
}) {
  const validation = useDialogValidation();
  const mapActions = (nodes: React.ReactNode): React.ReactNode =>
    React.Children.map(nodes, (child) => {
      if (
        !React.isValidElement<{
          className?: string;
          variant?: string;
          disabled?: boolean;
          type?: string;
          children?: React.ReactNode;
        }>(child)
      )
        return child;
      if (child.type === React.Fragment)
        return React.cloneElement(child, {
          children: mapActions(child.props.children),
        });
      const primary =
        (child.type === Button &&
          (!child.props.variant ||
            child.props.variant === "default" ||
            child.props.variant === "destructive")) ||
        child.props.className?.split(/\s+/).includes("primary") ||
        child.props.type === "submit";
      return primary && validation?.invalid
        ? React.cloneElement(child, { disabled: true })
        : child;
    });
  const actions = mapActions(children);
  return (
    <div
      data-slot="dialog-footer"
      className={cn("studio-window-actions", className)}
      {...props}
    >
      {actions}
      {showCloseButton && (
        <DialogPrimitive.Close render={<Button variant="outline" />}>
          Close
        </DialogPrimitive.Close>
      )}
    </div>
  );
}

function DialogTitle({ className, ...props }: DialogPrimitive.Title.Props) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn(
        "font-heading text-base leading-none font-medium",
        className,
      )}
      {...props}
    />
  );
}

function DialogDescription({
  className,
  ...props
}: DialogPrimitive.Description.Props) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn(
        "text-sm text-muted-foreground *:[a]:underline *:[a]:underline-offset-3 *:[a]:hover:text-foreground",
        className,
      )}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
};
