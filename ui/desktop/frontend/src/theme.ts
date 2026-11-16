export type Theme = "dark" | "light";

export function normalizeTheme(value: unknown): Theme {
  if (value === "system" && typeof window !== "undefined") return window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  return value === "light" ? "light" : "dark";
}

export function applyTheme(value: unknown): Theme {
  const theme = normalizeTheme(value);
  document.documentElement.classList.toggle("dark", theme === "dark");
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
  if (theme === "dark") {
    window.runtime?.WindowSetDarkTheme?.();
    window.runtime?.WindowSetBackgroundColour?.(29, 33, 40, 255);
  } else {
    window.runtime?.WindowSetLightTheme?.();
    window.runtime?.WindowSetBackgroundColour?.(233, 237, 242, 255);
  }
  try {
    localStorage.setItem("CraftBox-theme", theme);
  } catch {}
  return theme;
}

export function initializeTheme() {
  let saved: unknown;
  try {
    saved = localStorage.getItem("CraftBox-theme");
  } catch {}
  return applyTheme(saved);
}
