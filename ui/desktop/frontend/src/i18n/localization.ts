export const languages = {
  en: "English",
  "zh-Hans": "简体中文",
  es: "Español",
  de: "Deutsch",
  fr: "Français",
  pt: "Português",
  ja: "日本語",
} as const;

export type Locale = keyof typeof languages;
export type Catalog = Record<string, string>;

export function normalizeLocale(value: unknown): Locale {
  if (typeof value !== "string") return "en";
  const locale = value.trim().toLowerCase().replaceAll("_", "-");
  if (["zh", "zh-hans", "zh-cn", "zh-sg", "zh-hans-cn"].includes(locale))
    return "zh-Hans";
  const base = locale.split("-")[0];
  return base !== "zh" && Object.hasOwn(languages, base) ? base as Locale : "en";
}

const normalizeText = (text: string) => text.replace(/\s+/g, " ").trim();
const placeholder = /\{(\d+)\}/g;
const goVerb = /%(?:\[(\d+)\])?[-+0#.\d]*([bcdeEfFgGopqstTuvwxX%])/g;
const escape = (text: string) => text.replace(/[.*+?^{}()|[\]\\$]/g, "\\$&").replaceAll(" ", "\\s+");

type Pattern = { expression: RegExp; replacement: string; indices: number[]; weight: number };

function goTemplate(value: string) {
  let index = 0;
  return value.replace(goVerb, (_token, offset: string | undefined, verb: string) => {
    if (verb === "%") return "%";
    if (offset) index = Number(offset) - 1;
    return "{" + index++ + "}";
  });
}

function compileCatalog(catalog: Catalog) {
  const exact = new Map<string, string>();
  const prefixes = new Map<string, Pattern[]>();
  const suffixes = new Map<string, Pattern[]>();
  for (const [source, translation] of Object.entries(catalog)) {
    if (!translation || source === translation) continue;
    const english = normalizeText(source);
    exact.set(english, translation);
    const text = goTemplate(english);
    const matches = Array.from(text.matchAll(placeholder));
    if (!matches.length) continue;
    const prefix = text.slice(0, matches[0].index);
    const suffix = text.slice(matches.at(-1)!.index! + matches.at(-1)![0].length);
    if (!prefix && !suffix) continue;
    let end = 0;
    let pattern = "^";
    for (const match of matches) {
      pattern += escape(text.slice(end, match.index)) + "(.+?)";
      end = match.index! + match[0].length;
    }
    pattern += escape(text.slice(end)) + "$";
    const table = prefix ? prefixes : suffixes;
    const key = prefix ? prefix[0] : suffix.at(-1)!;
    const patterns = table.get(key) ?? [];
    patterns.push({
      expression: new RegExp(pattern),
      replacement: text === english ? translation : goTemplate(translation),
      indices: matches.map((match) => Number(match[1])),
      weight: text.length - matches.length * 3,
    });
    table.set(key, patterns);
  }
  for (const table of [prefixes, suffixes])
    for (const patterns of table.values()) patterns.sort((a, b) => b.weight - a.weight);
  return { exact, prefixes, suffixes };
}

export function createLocalization(load: (locale: Exclude<Locale, "en">) => Promise<Catalog>) {
  const listeners = new Set<() => void>();
  let current: Locale = "en";
  let request = 0;
  let compiled = compileCatalog({});
  let resolved = new Map<string, string>();
  return {
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
    language: () => current,
    async setLocale(locale: Locale) {
      const revision = ++request;
      if (locale === current) return;
      const catalog = locale === "en" ? {} : await load(locale);
      if (revision !== request) return;
      compiled = compileCatalog(catalog);
      resolved = new Map();
      current = locale;
      listeners.forEach((listener) => listener());
    },
    text(value: string) {
      if (current === "en" || value.length > 4096 || !value.trim()) return value;
      const content = value.trim();
      const key = normalizeText(value);
      const cached = resolved.get(content);
      let translated = cached ?? compiled.exact.get(key);
      if (translated === undefined) {
        const lists = [compiled.prefixes.get(key[0]), compiled.suffixes.get(key.at(-1)!)];
        for (const patterns of lists) {
          if (!patterns) continue;
          for (const { expression, replacement, indices } of patterns) {
            const matched = expression.exec(content);
            if (!matched) continue;
            translated = replacement.replace(placeholder, (token, index: string) => {
              const position = indices.indexOf(Number(index));
              return position < 0 ? token : matched[position + 1];
            });
            break;
          }
          if (translated !== undefined) break;
        }
      }
      if (resolved.size >= 4096) resolved.clear();
      if (translated === undefined) {
        resolved.set(content, content);
        return value;
      }
      resolved.set(content, translated);
      return value.replace(/\S[\s\S]*\S|\S/, () => translated!);
    },
  };
}
