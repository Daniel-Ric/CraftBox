import { Fragment, jsx, jsxs } from "react/jsx-runtime";
import { createContext, useContext } from "react";
import type { ElementType, ReactNode } from "react";
import { localization, text, useLocale } from "./index";

type Props = Record<string, unknown> & { children?: ReactNode };
const translatedAttributes = ["aria-label", "aria-description", "aria-valuetext", "aria-roledescription", "title", "alt", "placeholder", "data-tooltip"] as const;
const verbatimTags = new Set(["code", "kbd", "samp", "pre", "textarea", "script", "style", "svg"]);
const sourceText = createContext(false);

function translateChildren(value: ReactNode): ReactNode {
  if (typeof value === "string") return text(value);
  if (!Array.isArray(value)) return value;
  if (value.every((child) => typeof child === "string" || typeof child === "number" || child == null || typeof child === "boolean")) {
    const source = value.filter((child) => typeof child === "string" || typeof child === "number").join("");
    const translated = text(source);
    if (source !== translated) return translated;
  }
  return value.map(translateChildren);
}

const wrappers = new Map<ElementType, (props: Props) => ReturnType<typeof jsx>>();
const originalTypes = new WeakMap<object, ElementType>();

function renderElement(type: ElementType, props: Props) {
  return (Array.isArray(props.children) ? jsxs : jsx)(type, props);
}

function LocalizedFragment({ children }: Props) {
  useLocale();
  const verbatim = useContext(sourceText);
  return renderElement(Fragment, {
    children: verbatim || localization.language() === "en" ? children : translateChildren(children),
  });
}
originalTypes.set(LocalizedFragment, Fragment);

export function originalElementType(type: unknown) {
  if (typeof type === "function" || typeof type === "object" && type !== null)
    return originalTypes.get(type) ?? type;
  return type;
}

export function localizedType(type: ElementType, rawProps: unknown): ElementType {
  const props = rawProps as Props | null | undefined;
  if (type === Fragment) return LocalizedFragment;
  if (!props) return type;
  const verbatim = typeof type === "string" && verbatimTags.has(type);
  const includeChildren = Object.hasOwn(props, "children") && !verbatim;
  const optionLabel = (type === "option" || type === "optgroup") && Object.hasOwn(props, "label");
  if (!includeChildren && !optionLabel && !Object.hasOwn(props, "translate") &&
    !translatedAttributes.some((key) => Object.hasOwn(props, key))) return type;
  const cached = wrappers.get(type);
  if (cached) return cached;
  const tag = type;
  function LocalizedElement(original: Props) {
    useLocale();
    const inherited = useContext(sourceText);
    const preserve = original.translate === "no" || (original.translate !== "yes" && inherited);
    const props = { ...original };
    const render = () => {
      const element = renderElement(tag, props);
      return preserve === inherited ? element : jsx(sourceText.Provider, { value: preserve, children: element });
    };
    if (localization.language() === "en" || preserve) return render();
    for (const attr of translatedAttributes) {
      const source = props[attr];
      if (typeof source === "string") props[attr] = text(source);
    }
    if ((tag === "option" || tag === "optgroup") && typeof props.label === "string") {
      props.label = text(props.label);
    }
    if (!verbatim && (tag !== "option" || props.value !== undefined)) {
      props.children = translateChildren(props.children);
    } else if (tag === "option" && typeof props.children === "string") {
      const translated = text(props.children);
      if (props.children !== translated) {
        props.value = props.children.trim().replace(/\s+/g, " ");
        props.children = translated;
      }
    }
    return render();
  }
  wrappers.set(tag, LocalizedElement);
  originalTypes.set(LocalizedElement, tag);
  return LocalizedElement;
}
