import { useSyncExternalStore } from "react";
import { createLocalization, normalizeLocale } from "./localization";

export { languages, normalizeLocale, type Locale } from "./localization";

const preferenceKey = "craftbox-interface-language";
const loaders = {
  "zh-Hans": () => import("../../../../../locale/messages/zh-Hans.json"),
  es: () => import("../../../../../locale/messages/es.json"),
  de: () => import("../../../../../locale/messages/de.json"),
  fr: () => import("../../../../../locale/messages/fr.json"),
  pt: () => import("../../../../../locale/messages/pt.json"),
  ja: () => import("../../../../../locale/messages/ja.json"),
};

const storeKey = Symbol.for("craftbox.localization");
const stores = globalThis as typeof globalThis & Record<symbol, ReturnType<typeof createLocalization> | undefined>;
export const localization = stores[storeKey] ??= createLocalization(async (locale) => (await loaders[locale]()).default);
export const text = localization.text;

export function formatLocale(english = "en-US") {
  const locale = localization.language();
  return locale === "en" ? english : locale;
}

export function useLocale() {
  return useSyncExternalStore(localization.subscribe, localization.language, localization.language);
}

export async function initializeLanguage(syncNative: (language: string) => Promise<unknown>) {
  const update = () => {
    const language = localization.language();
    document.documentElement.lang = language;
    return syncNative(language).catch(() => {});
  };
  try {
    await localization.setLocale(normalizeLocale(localStorage.getItem(preferenceKey)));
  } catch {}
  await update();
  localization.subscribe(() => { void update(); });
  window.addEventListener("storage", (event) => {
    if (event.key === preferenceKey)
      void localization.setLocale(normalizeLocale(event.newValue)).catch(() => {});
  });
}

export async function saveLanguage(value: string) {
  const locale = normalizeLocale(value);
  await localization.setLocale(locale);
  if (localization.language() === locale)
    localStorage.setItem(preferenceKey, locale);
}
