import { useEffect, useId, useRef, useState, type RefObject } from "react";
import { Search, X } from "lucide-react";
import { localization, text, useLocale } from "./i18n";
import { findRenderControls, type RenderSearchEntry } from "./render-search";

const controls =
  "button, input:not([type=hidden]), select, textarea, [role=slider]";
const relatedTerms: Record<string, string[]> = {
  "Depth of field": ["DOF", "Bokeh", "Focus", "Lens", "Aperture"],
  Fog: ["Haze", "Mist", "Volumetric lighting", "Light shafts"],
  Skybox: ["HDRI", "Environment", "Sky"],
  "Motion blur": ["Shutter", "Exposure", "Temporal sampling"],
  "Field of view": ["FOV", "Focal length", "Lens"],
  Sunlight: ["Sun", "Directional light", "Daylight"],
  "Ambient occlusion": ["AO", "SSAO", "Contact shadows"],
};

export default function RenderInspectorSearch({
  panel,
  tabs,
  onTab,
  extra = [],
  attribute = "data-inspector",
}: {
  panel: RefObject<HTMLElement | null>;
  tabs: ReadonlyArray<readonly [string, string]>;
  onTab: (tab: string) => void;
  extra?: RenderSearchEntry[];
  attribute?: "data-inspector" | "data-camera-panel";
}) {
  useLocale();
  const list = useId();
  const field = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const onFind = (event: KeyboardEvent) => {
      if (
        !(event.ctrlKey || event.metaKey) ||
        event.altKey ||
        event.shiftKey ||
        event.key.toLowerCase() !== "f" ||
        !field.current?.getClientRects().length
      )
        return;
      event.preventDefault();
      field.current.focus();
      field.current.select();
    };
    window.addEventListener("keydown", onFind);
    return () => window.removeEventListener("keydown", onFind);
  }, []);
  const tabName = (id: string) =>
    text(tabs.find(([tab]) => tab === id)?.[1] ?? id);
  const jump = (tab: string, node: HTMLElement) => {
    onTab(tab);
    requestAnimationFrame(() => {
      node.scrollIntoView({ block: "center", behavior: "instant" });
      const target = node.matches(controls)
        ? node
        : node.querySelector<HTMLElement>(controls);
      target?.focus({ preventScroll: true });
      if (!matchMedia("(prefers-reduced-motion: reduce)").matches)
        (node.closest("label, fieldset") ?? node).animate(
          [
            { outline: "2px solid var(--primary)" },
            { outline: "2px solid transparent" },
          ],
          { duration: 900 },
        );
    });
  };
  const results =
    open && query.trim()
      ? findRenderControls(query, [
          ...extra.map((entry) => ({
            ...entry,
            label: text(entry.label),
            terms: [
              ...entry.terms,
              entry.label,
              ...entry.terms.map((term) => text(term)),
              tabName(entry.tab),
            ],
          })),
          ...Array.from(
            panel.current?.querySelectorAll<HTMLElement>(`[${attribute}]`) ??
              [],
          ).flatMap((section, index) => {
            const tab = section.getAttribute(attribute)!;
            const legend =
              section.querySelector("legend, h3")?.textContent?.trim() ?? "";
            const nodes = [
              ...section.querySelectorAll<HTMLElement>(controls),
            ].filter((node) => node.closest(`[${attribute}]`) === section);
            return nodes.flatMap((node, i) => {
              if (node.closest("[data-render-search]")) return [];
              const label =
                node.getAttribute("aria-label") ||
                node.getAttribute("data-tooltip") ||
                node.getAttribute("title") ||
                (node.matches("button")
                  ? node.textContent?.trim()
                  : node.closest("label")?.firstChild?.textContent?.trim());
              if (!label || label.length > 100) return [];
              const source = [
                label,
                ...localization.sourceTerms(label),
                ...localization.sourceTerms(legend),
              ];
              const synonyms = Object.entries(relatedTerms).flatMap(
                ([key, aliases]) =>
                  source.some((term) =>
                    term.toLowerCase().includes(key.toLowerCase()),
                  )
                    ? aliases
                    : [],
              );
              return [
                {
                  id: `${index}-${i}`,
                  label,
                  tab,
                  terms: [
                    ...source,
                    ...synonyms,
                    ...synonyms.map((term) => text(term)),
                    legend,
                    tabName(tab),
                    tab,
                  ],
                  open: () => jump(tab, node),
                },
              ];
            });
          }),
        ]).slice(0, 12)
      : [];
  const pick = (entry: RenderSearchEntry) => {
    setOpen(false);
    setQuery("");
    entry.open();
  };
  return (
    <div
      className="render-inspector-search"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
      }}
    >
      <div className="render-inspector-search-field">
        <Search size={15} aria-hidden="true" />
        <input
          ref={field}
          role="combobox"
          aria-label="Search editor controls"
          placeholder="Search editor controls…"
          aria-expanded={open && !!query.trim()}
          aria-controls={list}
          aria-autocomplete="list"
          aria-activedescendant={
            results[active] ? `${list}-${results[active].id}` : undefined
          }
          value={query}
          onFocus={() => setOpen(true)}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
            setOpen(true);
          }}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              setOpen(false);
              event.stopPropagation();
            }
            if (
              (event.key === "ArrowDown" || event.key === "ArrowUp") &&
              results.length
            ) {
              event.preventDefault();
              const next =
                (active +
                  (event.key === "ArrowDown" ? 1 : -1) +
                  results.length) %
                results.length;
              setActive(next);
              document
                .getElementById(`${list}-${results[next].id}`)
                ?.scrollIntoView({ block: "nearest" });
            }
            if (event.key === "Enter" && results[active]) {
              event.preventDefault();
              pick(results[active]);
            }
          }}
        />
        {query && (
          <button
            type="button"
            aria-label="Clear editor search"
            onClick={() => {
              setQuery("");
              field.current?.focus();
            }}
          >
            <X size={14} />
          </button>
        )}
      </div>
      {open && !!query.trim() && (
        <div className="render-inspector-search-results">
          <div id={list} role="listbox" aria-label="Matching editor controls">
            {results.map((entry, index) => (
              <button
                key={entry.id}
                type="button"
                role="option"
                id={`${list}-${entry.id}`}
                aria-selected={index === active}
                tabIndex={-1}
                onPointerMove={() => setActive(index)}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => pick(entry)}
              >
                <span>{entry.label}</span>
                <small>{tabName(entry.tab)}</small>
              </button>
            ))}
          </div>
          {!results.length && <p role="status">No matching controls</p>}
        </div>
      )}
    </div>
  );
}
