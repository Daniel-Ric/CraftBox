import {
  Children,
  isValidElement,
  useEffect,
  useId,
  useState,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { CircleAlert, TriangleAlert, X } from "lucide-react";
import { toast } from "sonner";
import { notificationHistory } from "./notification-history";
import "./studio-flyouts.css";

function contentText(children: ReactNode): string {
  return Children.toArray(children)
    .map((child) => {
      if (typeof child === "string" || typeof child === "number")
        return String(child);
      if (isValidElement<{ children?: ReactNode }>(child))
        return contentText(child.props.children);
      return "";
    })
    .join(" ")
    .replace(/\s+/g, " ")
    .trim();
}

// A portal preserves live recovery buttons and validation descriptions while the
// notification lives in the common Sonner stack instead of the page layout.
export default function FeedbackFlyout({
  children,
  title = "Action failed",
  tone = "error",
  id,
}: {
  children: ReactNode;
  title?: string;
  tone?: "error" | "warning";
  id?: string;
}) {
  const toastID = useId();
  const [host] = useState(() => document.createElement("div"));
  const text = contentText(children);
  useEffect(() => {
    if (!text) {
      toast.dismiss(toastID);
      return;
    }
    notificationHistory.record({ id: toastID, title, description: text, tone });
    const Icon = tone === "warning" ? TriangleAlert : CircleAlert;
    toast.custom(
      () => (
        <section
          className="studio-flyout"
          data-tone={tone}
          role={tone === "error" ? "alert" : "status"}
        >
          <div className="flyout-symbol">
            <Icon size={18} />
          </div>
          <div className="flyout-content">
            <strong>{title}</strong>
            <div
              className="feedback-flyout-body"
              ref={(node) => {
                if (node && host.parentNode !== node) node.appendChild(host);
              }}
            />
          </div>
          <button
            className="flyout-dismiss"
            aria-label="Dismiss notification"
            onClick={() => toast.dismiss(toastID)}
          >
            <X size={14} />
          </button>
        </section>
      ),
      { id: toastID, duration: Infinity },
    );
  }, [text, title, tone, toastID, host]);
  useEffect(
    () => () => {
      toast.dismiss(toastID);
    },
    [toastID],
  );
  return (
    <>
      {id && (
        <span id={id} className="sr-only">
          {text}
        </span>
      )}
      {createPortal(children, host)}
    </>
  );
}
