import FeedbackFlyout from "./FeedbackFlyout";
import {
  forwardRef,
  useEffect,
  useId,
  useImperativeHandle,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentProps,
} from "react";
import { useDialogValidation } from "./form-validation";

export type StudioInputProps = ComponentProps<"input"> & {
  "data-tooltip"?: string;
  error?: string;
  validate?: (value: string) => string | undefined;
                                                                                   
  increment?: number;
  pending?: boolean;
};
export function inputError(value: string, p: StudioInputProps) {
  if (p.required && !value.trim()) return "This field is required.";
  if (!value)
    return p.type === "number"
      ? "Enter a number."
      : (p.validate?.(value) ?? "");
  if (p.type === "number") {
    const n = Number(value);
    if (!Number.isFinite(n)) return "Enter a valid number.";
    if (p.min !== undefined && n < Number(p.min))
      return `Enter ${p.min} or more.`;
    if (p.max !== undefined && n > Number(p.max))
      return `Enter ${p.max} or less.`;
    if (p.step !== undefined && p.step !== "any") {
      const step = Number(p.step),
        base = Number(p.min ?? 0),
        q = (n - base) / step;
      if (step > 0 && Math.abs(q - Math.round(q)) > 1e-7)
        return `Use increments of ${p.step}.`;
    }
  }
  if (p.minLength !== undefined && value.length < p.minLength)
    return `Enter at least ${p.minLength} characters.`;
  if (p.maxLength !== undefined && value.length > p.maxLength)
    return `Use ${p.maxLength} characters or fewer.`;
  if (p.type === "email" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value))
    return "Enter a valid email address.";
  if (p.type === "url") {
    try {
      new URL(value);
    } catch {
      return "Enter a full URL, including https://.";
    }
  }
  if (p.pattern) {
    try {
      if (!new RegExp(`^(?:${p.pattern})$`, "u").test(value))
        return "Enter a value in the required format.";
    } catch {
      /* Unsupported platform patterns retain native semantics. */
    }
  }
  return p.validate?.(value) ?? "";
}
const StudioInput = forwardRef<HTMLInputElement, StudioInputProps>(
  function StudioInput(
    { error, validate, title, increment, pending, "data-tooltip": tooltip, ...props },
    ref,
  ) {
    const internal = useRef<HTMLInputElement>(null),
      id = useId();
    useImperativeHandle(ref, () => internal.current!, []);
    const controlled = props.value !== undefined;
    const [draft, setDraft] = useState(
        String(props.value ?? props.defaultValue ?? ""),
      ),
      [touched, setTouched] = useState(false),
      [label, setLabel] = useState<string>();
    const shown =
      controlled && props.type !== "number" && props.type !== "time"
        ? String(props.value ?? "")
        : draft;
    const simple = [
      "checkbox",
      "radio",
      "range",
      "color",
      "file",
      "hidden",
      "button",
      "submit",
      "reset",
    ].includes(props.type ?? "");
    const validation = useDialogValidation(),
      field = validation?.field;
    const invalid =
      !simple &&
      !props.disabled &&
      !props.readOnly &&
      !!(pending || error || inputError(shown, { ...props, validate }));
    const constraintMessage =
      !simple && !props.disabled && !props.readOnly
        ? error || inputError(shown, { ...props, validate }) || (pending ? "Checking…" : "")
        : "";
    useLayoutEffect(() => {
      internal.current?.setCustomValidity(constraintMessage);
    }, [constraintMessage]);
    useEffect(() => {
      field?.(id, invalid);
      return () => field?.(id, false);
    }, [field, id, invalid]);
    useEffect(() => {
      if (controlled) setDraft(String(props.value ?? ""));
    }, [props.value, controlled]);
    useLayoutEffect(() => {
      if (simple || props["aria-label"] || props["aria-labelledby"]) return;
      const original = internal.current?.labels?.[0];
      if (original) {
        const copy = original.cloneNode(true) as HTMLElement;
        copy
          .querySelectorAll("input,button,select,svg,.field-error")
          .forEach((n) => n.remove());
        const text = copy.textContent?.trim();
        if (text) setLabel(text);
      }
    }, [simple, props["aria-label"], props["aria-labelledby"]]);
    if (simple)
      return (
        <input {...props} ref={internal} data-tooltip={tooltip ?? title} />
      );
    const message =
      error || (touched ? inputError(shown, { ...props, validate }) : "");
    const described =
      [props["aria-describedby"], message ? id : undefined]
        .filter(Boolean)
        .join(" ") || undefined;
    return (
      <span className="field-control" data-invalid={message ? "" : undefined}>
        <input
          {...props}
          step={props.step ?? (props.type === "number" ? "any" : undefined)}
          ref={internal}
          defaultValue={undefined}
          value={shown}
          data-tooltip={tooltip ?? title}
          aria-label={props["aria-label"] ?? label}
          aria-invalid={message ? true : props["aria-invalid"]}
          aria-busy={pending || undefined}
          aria-describedby={described}
          onChange={(e) => {
            const raw = e.target.value;
            setDraft(raw);
            setTouched(true);
            if (
              props.type !== "number" ||
              !inputError(raw, { ...props, validate })
            )
              props.onChange?.(e);
          }}
          onBlur={(e) => {
            setTouched(true);
            props.onBlur?.(e);
          }}
          onInvalid={(e) => {
            e.preventDefault();
            setTouched(true);
            props.onInvalid?.(e);
          }}
          onKeyDown={(e) => {
            if (
              props.type === "number" &&
              increment &&
              increment > 0 &&
              !props.disabled &&
              !props.readOnly &&
              (e.key === "ArrowUp" || e.key === "ArrowDown")
            ) {
              e.preventDefault();
              e.stopPropagation();
              let next =
                (Number.isFinite(Number(shown)) ? Number(shown) : 0) +
                (e.key === "ArrowUp" ? increment : -increment);
              if (props.min !== undefined)
                next = Math.max(Number(props.min), next);
              if (props.max !== undefined)
                next = Math.min(Number(props.max), next);
              const value = String(Number(next.toPrecision(12)));
              Object.getOwnPropertyDescriptor(
                HTMLInputElement.prototype,
                "value",
              )?.set?.call(e.currentTarget, value);
              e.currentTarget.dispatchEvent(
                new Event("input", { bubbles: true }),
              );
              return;
            }
            if (
              e.key === "Escape" &&
              controlled &&
              (props.type === "number" || props.type === "time")
            ) {
              setDraft(String(props.value ?? ""));
              setTouched(false);
            }
            if (
              e.key === "Enter" &&
              inputError(shown, { ...props, validate })
            ) {
              e.preventDefault();
              e.stopPropagation();
              setTouched(true);
            } else props.onKeyDown?.(e);
          }}
        />
        {message && (
          <FeedbackFlyout title={props["aria-label"] ? `Check ${props["aria-label"]}` : "Invalid input"} id={id}>
            {message}
          </FeedbackFlyout>
        )}
        {pending && !message && <span className="field-status" role="status">Checking…</span>}
      </span>
    );
  },
);
export default StudioInput;
