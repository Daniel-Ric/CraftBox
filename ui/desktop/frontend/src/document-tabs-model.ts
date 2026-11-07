export type DocumentTab = {
  id: string;
  source: string;
  path?: string;
  name: string;
  preview: boolean;
  pinned: boolean;
  dirty: boolean;
  busy: boolean;
};
export type DocumentTabsState = {
  tabs: DocumentTab[];
  active: string;
  closed: DocumentTab[];
};
export const emptyDocuments = (): DocumentTabsState => ({
  tabs: [],
  active: "",
  closed: [],
});
export const documentsLocked = (s: DocumentTabsState) =>
  s.tabs.some((t) => t.busy);
export function activateDocument(s: DocumentTabsState, id: string) {
  return documentsLocked(s) || !s.tabs.some((t) => t.id === id)
    ? s
    : { ...s, active: id };
}
export function openDocument(
  s: DocumentTabsState,
  input: Pick<DocumentTab, "id" | "source" | "name">,
  permanent = false,
): DocumentTabsState {
  if (documentsLocked(s)) return s;
  const existing = s.tabs.find((t) => t.source === input.source);
  if (existing)
    return {
      ...s,
      active: existing.id,
      tabs: s.tabs.map((t) =>
        t.id === existing.id && permanent ? { ...t, preview: false } : t,
      ),
    };
  const preview = permanent
    ? undefined
    : s.tabs.find((t) => t.preview && !t.dirty && !t.pinned);
  const tab: DocumentTab = {
    ...input,
    preview: !permanent,
    pinned: false,
    dirty: false,
    busy: false,
  };
  return {
    ...s,
    active: tab.id,
    closed: s.closed.filter((t) => t.id !== input.id),
    tabs: preview
      ? s.tabs.map((t) => (t.id === preview.id ? tab : t))
      : [...s.tabs, tab],
  };
}
export function markDocument(
  s: DocumentTabsState,
  id: string,
  patch: Partial<
    Pick<DocumentTab, "name" | "dirty" | "busy" | "path" | "source">
  >,
): DocumentTabsState {
  let changed = false;
  const tabs = s.tabs.map((t) => {
    if (t.id !== id) return t;
    const n = {
      ...t,
      ...patch,
      preview: patch.dirty || patch.busy ? false : t.preview,
    };
    if (JSON.stringify(n) === JSON.stringify(t)) return t;
    changed = true;
    return n;
  });
  return changed ? { ...s, tabs } : s;
}
export function keepDocument(
  s: DocumentTabsState,
  id: string,
): DocumentTabsState {
  return {
    ...s,
    tabs: s.tabs.map((t) => (t.id === id ? { ...t, preview: false } : t)),
  };
}
export function pinDocument(
  s: DocumentTabsState,
  id: string,
): DocumentTabsState {
  if (documentsLocked(s)) return s;
  const tabs = s.tabs.map((t) =>
    t.id === id ? { ...t, pinned: !t.pinned, preview: false } : t,
  );
  return {
    ...s,
    tabs: [...tabs.filter((t) => t.pinned), ...tabs.filter((t) => !t.pinned)],
  };
}
export function reorderDocument(
  s: DocumentTabsState,
  id: string,
  before: string,
): DocumentTabsState {
  if (documentsLocked(s)) return s;
  const tab = s.tabs.find((t) => t.id === id),
    target = s.tabs.find((t) => t.id === before);
  if (!tab || !target || tab.pinned !== target.pinned || id === before)
    return s;
  const tabs = s.tabs.filter((t) => t.id !== id);
  tabs.splice(
    tabs.findIndex((t) => t.id === before),
    0,
    tab,
  );
  return { ...s, tabs };
}
// Call only after the owner has resolved dirty documents. Busy areas never close.
export function closeDocuments(
  s: DocumentTabsState,
  ids: string[],
): DocumentTabsState {
  if (documentsLocked(s)) return s;
  const removed = s.tabs.filter((t) => ids.includes(t.id));
  if (!removed.length) return s;
  const tabs = s.tabs.filter((t) => !ids.includes(t.id));
  const oldIndex = s.tabs.findIndex((t) => t.id === s.active);
  return {
    ...s,
    tabs,
    active: tabs.some((t) => t.id === s.active)
      ? s.active
      : tabs[Math.min(Math.max(oldIndex, 0), tabs.length - 1)]?.id || "",
    closed: [...removed.slice().reverse(), ...s.closed].slice(0, 20),
  };
}
