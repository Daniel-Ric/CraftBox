import { parseExactJSON, rawBytes } from "./packet-inspector-model";
export type FormObject = Record<string, unknown>;
export type PacketForm = {
  id: unknown;
  type: "form" | "modal" | "custom_form";
  data: FormObject;
};
export const formObject = (v: unknown): FormObject | undefined =>
  v !== null && typeof v === "object" && !Array.isArray(v)
    ? (v as FormObject)
    : undefined;
export const formList = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);
const translations: Record<string, string> = {
  "gui.yes": "Yes",
  "gui.no": "No",
  "gui.ok": "OK",
  "gui.cancel": "Cancel",
  "gui.submit": "Submit",
  "gui.close": "Close",
};
const translateLabel = (value: string) =>
  Object.prototype.hasOwnProperty.call(translations, value)
    ? translations[value]
    : value;
export function formText(value: unknown, depth = 0): string {
  if (depth > 12) return "";
  if (typeof value === "string") return translateLabel(value);
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  const object = formObject(value);
  if (!object) return "";
  if (Array.isArray(object.rawtext))
    return object.rawtext.map((v) => formText(v, depth + 1)).join("");
  if (typeof object.text === "string") return object.text;
  if (typeof object.translate === "string") {
    let n = 0;
    const args = formList(object.with);
    return translateLabel(object.translate).replace(/%s/g, () =>
      formText(args[n++], depth + 1),
    );
  }
  return "";
}
export const serverGlyphs =
  /[\uE000-\uF8FF\u{F0000}-\u{FFFFD}\u{100000}-\u{10FFFD}]/gu;
export function cleanFormText(value: unknown, showGlyphs = false): string {
  const text = formText(value);
  return (showGlyphs ? text : text.replace(serverGlyphs, "")).replace(
    /\n[\t ]*\n(?:[\t ]*\n)+/g,
    "\n\n",
  );
}
export function plainFormText(value: unknown): string {
  return cleanFormText(value).replace(/§[0-9a-u]/gi, "");
}
export function packetForm(value: unknown): PacketForm | undefined {
  const object = formObject(value);
  if (!object) return;
  let raw = object.FormData ?? object.formData;
  if (typeof raw === "string") {
    try {
      raw = parseExactJSON(raw);
    } catch {
      try {
        raw = parseExactJSON(new TextDecoder().decode(rawBytes(raw as string)));
      } catch {
        return;
      }
    }
  }
  const data = formObject(raw);
  if (!data) return;
  if (
    data.type !== "form" &&
    data.type !== "modal" &&
    data.type !== "custom_form"
  )
    return;
  return { id: object.FormID ?? object.formId, type: data.type, data };
}
export const formColors: Record<string, string> = {
  "0": "#000000",
  "1": "#0000aa",
  "2": "#00aa00",
  "3": "#00aaaa",
  "4": "#aa0000",
  "5": "#aa00aa",
  "6": "#ffaa00",
  "7": "#aaaaaa",
  "8": "#555555",
  "9": "#5555ff",
  a: "#55ff55",
  b: "#55ffff",
  c: "#ff5555",
  d: "#ff55ff",
  e: "#ffff55",
  f: "#ffffff",
  g: "#ddd605",
  h: "#e3d4d1",
  i: "#cecaca",
  j: "#443a3b",
  m: "#971607",
  n: "#b4684d",
  p: "#deb12d",
  q: "#47a036",
  s: "#2cbaa8",
  t: "#21497b",
  u: "#9a5cc6",
};
export type FormTextRun = {
  text: string;
  color?: string;
  bold?: boolean;
  italic?: boolean;
  obfuscated?: boolean;
  shortened?: boolean;
};
export function formattedFormText(
  value: unknown,
  showGlyphs = false,
  maxChars = 6000,
  maxRuns = 256,
): FormTextRun[] {
  const source = cleanFormText(value, showGlyphs);
  let text = source.slice(0, Math.max(0, maxChars)),
    shortened = text.length < source.length;
  if (
    shortened &&
    ((text.charCodeAt(text.length - 1) >= 0xd800 &&
      text.charCodeAt(text.length - 1) <= 0xdbff) ||
      text.endsWith("§"))
  )
    text = text.slice(0, -1);
  const runs: FormTextRun[] = [];
  let style: Omit<FormTextRun, "text"> = {},
    buffer = "";
  const flush = () => {
    if (!buffer) return;
    if (runs.length < maxRuns) runs.push({ ...style, text: buffer });
    else shortened = true;
    buffer = "";
  };
  if (maxRuns > 0)
    for (let i = 0; i < text.length; i++) {
      const code = text[i] === "§" ? text[i + 1]?.toLowerCase() : undefined;
      if (code && (formColors[code] || ["k", "l", "o", "r"].includes(code))) {
        flush();
        if (runs.length >= maxRuns) {
          shortened = true;
          break;
        }
        if (formColors[code]) style = { ...style, color: formColors[code] };
        else if (code === "r") style = {};
        else if (code === "l") style = { ...style, bold: true };
        else if (code === "o") style = { ...style, italic: true };
        else style = { ...style, obfuscated: true };
        i++;
      } else buffer += text[i];
    }
  else if (source) shortened = true;
  flush();
  if (shortened)
    runs.push({
      text: " … (full text in JSON)",
      color: "#afb4bf",
      italic: true,
      shortened: true,
    });
  return runs;
}
export function formElements(form: PacketForm): FormObject[] {
  const value =
    form.type === "custom_form"
      ? form.data.content
      : Array.isArray(form.data.elements)
        ? form.data.elements
        : form.data.buttons;
  return formList(value).map(
    (v) => formObject(v) ?? { type: "unsupported", text: String(v) },
  );
}
export function formRange(field: FormObject) {
  const finite = (v: unknown, fallback: number) =>
    typeof v === "number" && Number.isFinite(v) ? v : fallback;
  const min = finite(field.min, 0),
    max = Math.max(min, finite(field.max, 100)),
    span = max - min;
  const step = Math.max(
    Number.EPSILON,
    Math.min(span || 1, finite(field.step, 1) > 0 ? finite(field.step, 1) : 1),
  );
  const initial = Math.min(max, Math.max(min, finite(field.default, min)));
  const value = Math.min(
    max,
    Math.max(min, min + Math.round((initial - min) / step) * step),
  );
  return { min, max, step, value };
}
export function initialFormValues(form: PacketForm): unknown[] {
  return formElements(form).map((field) => {
    switch (field.type) {
      case "input":
        return formText(field.default);
      case "toggle":
        return field.default === true;
      case "dropdown":
      case "step_slider": {
        const options = formList(
            field.type === "dropdown" ? field.options : field.steps,
          ),
          value =
            typeof field.default === "number" ? Math.floor(field.default) : 0;
        return options.length
          ? Math.max(0, Math.min(options.length - 1, value))
          : null;
      }
      case "slider":
        return formRange(field).value;
      default:
        return null;
    }
  });
}
