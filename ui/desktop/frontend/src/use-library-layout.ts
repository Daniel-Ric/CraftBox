import { useCallback, useState } from "react";
export function useLibraryLayout(kind: string) {
  const key = `craftbox-library-layout:${kind}`;
  const [view, setView] = useState<"grid" | "list">(() => {
    try {
      return localStorage.getItem(key) === "list" ? "list" : "grid";
    } catch {
      return "grid";
    }
  });
  const change = useCallback(
    (value: "grid" | "list") => {
      setView(value);
      try {
        localStorage.setItem(key, value);
      } catch {}
    },
    [key],
  );
  return [view, change] as const;
}
