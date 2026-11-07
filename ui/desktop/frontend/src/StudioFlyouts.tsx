import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Toaster, toast } from "sonner";
import {
  CircleCheck,
  Info,
  TriangleAlert,
  CircleAlert,
  X,
  FolderOpen,
  ArrowUpRight,
  LoaderCircle,
} from "lucide-react";
import { api } from "./bridge";
import "./studio-flyouts.css";
import {notificationHistory,type NotificationValue} from "./notification-history";

type Flyout = NotificationValue;
export function notify(value: Flyout, options: {history?: boolean} = {}) {
  if(options.history !== false) notificationHistory.record(value);
  return toast.custom(
    (id) => <FlyoutCard value={value} dismiss={() => toast.dismiss(id)} />,
    {
      id: value.id,
      duration:
        value.duration ??
        (value.tone === "error"
          ? Infinity
          : value.action
            ? 9000
            : value.tone === "warning"
              ? 10000
              : 5500),
    },
  );
}
export function notifySaved(
  path: string,
  open: (path: string) => void | Promise<unknown> = (value) =>
    api.OpenResultFolder(value),
) {
  if (path)
    notify({
      tone: "success",
      title: "File saved",
      description: path,
      action: { label: "Open folder", onClick: () => open(path) },
    });
}
export function notifyFailure(reason: unknown, title = "Action failed",id?:string) {
  notify({
    id,
    tone: "error",
    title,
    description: reason instanceof Error ? reason.message : String(reason),
  });
}
export function notifyAction(message: string, context = "CraftBox") {
  if (!message) return;
  if (/^Saved: /i.test(message)) {
    notifySaved(message.replace(/^Saved: /i, ""));
    return;
  }
  const tone = /error|failed|could not|unavailable|unsupported|denied/i.test(
    message,
  )
    ? "error"
    : /cancel/i.test(message)
      ? "info"
      : /limit|select .*first|use \/|purposes:|open: /i.test(message)
        ? "warning"
        : /copied|saved|applied|stopped|cleared/i.test(message)
          ? "success"
          : "info";
  notify({ tone, title: context, description: message });
}
function FlyoutCard({
  value,
  dismiss,
}: {
  value: Flyout;
  dismiss: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const Icon = {
    success: CircleCheck,
    info: Info,
    warning: TriangleAlert,
    error: CircleAlert,
  }[value.tone];
  return (
    <div
      className="studio-flyout"
      data-tone={value.tone}
      role={value.tone === "error" ? "alert" : "status"}
    >
      <div className="flyout-symbol">
        <Icon size={18} />
      </div>
      <div className="flyout-content">
        <strong>{value.title}</strong>
        {value.description && <p>{value.description}</p>}
        {value.action && (
          <button
            className="flyout-action"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await value.action!.onClick();
                dismiss();
              } catch (e) {
                notify({
                  tone: "error",
                  title: "Action failed",
                  description: e instanceof Error ? e.message : String(e),
                });
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? (
              <LoaderCircle size={13} className="spin" />
            ) : value.action.label === "Open folder" ? (
              <FolderOpen size={13} />
            ) : (
              <ArrowUpRight size={13} />
            )}
            <span>{value.action.label}</span>
          </button>
        )}
      </div>
      <button
        className="flyout-dismiss"
        aria-label="Dismiss notification"
        onClick={dismiss}
      >
        <X size={14} />
      </button>
    </div>
  );
}
export default function StudioFlyouts() {
  const [host] = useState(() => document.createElement("div"));
  useEffect(() => {
    host.className = "studio-flyout-layer";
    if (typeof host.showPopover === "function") host.setAttribute("popover", "manual");
    const place = () => {
      // Keep recovery controls inside the active modal's DOM focus boundary.
      // A manual popover puts their fixed-position layer above clipping and
      // transformed dialogs, while keeping the same Toaster mounted throughout.
      const dialogs = document.querySelectorAll<HTMLElement>('[data-slot="dialog-content"][data-open]');
      const parent = dialogs[dialogs.length - 1] ?? document.body;
      if (host.parentElement !== parent) parent.appendChild(host);
      if (typeof host.showPopover === "function" && !host.matches(":popover-open")) host.showPopover();
    };
    place();
    const observer = new MutationObserver(place);
    observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ["data-open", "data-closed"] });
    return () => { observer.disconnect(); host.remove(); };
  }, [host]);
  return createPortal(
    <Toaster
      className="studio-flyouts"
      position="bottom-right"
      offset={{ bottom: 52, right: 20 }}
      visibleToasts={3}
      gap={10}
      toastOptions={{ unstyled: true }}
      containerAriaLabel="Notifications"
    />, host
  );
}
