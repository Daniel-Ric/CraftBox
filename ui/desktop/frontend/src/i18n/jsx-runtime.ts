import { jsx as reactJSX, jsxs as reactJSXs } from "react/jsx-runtime";
import { localizedType } from "./localized-element";

export { Fragment, type JSX } from "react/jsx-runtime";

export const jsx: typeof reactJSX = (type, props, key) => reactJSX(localizedType(type, props), props, key);
export const jsxs: typeof reactJSXs = (type, props, key) => reactJSXs(localizedType(type, props), props, key);
