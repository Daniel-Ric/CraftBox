import { jsxDEV as reactJSXDEV } from "react/jsx-dev-runtime";
import { localizedType } from "./localized-element";

export { Fragment, type JSX } from "react/jsx-dev-runtime";

export const jsxDEV: typeof reactJSXDEV = (type, props, key, staticChildren, source, self) =>
  reactJSXDEV(localizedType(type, props), props, key, staticChildren, source, self);
