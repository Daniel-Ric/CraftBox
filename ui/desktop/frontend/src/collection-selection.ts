export function collectionSelection(
  ids: string[],
  selected: string[],
  anchor: string,
  id: string,
  modifiers: { ctrlKey?: boolean; metaKey?: boolean; shiftKey?: boolean },
  multiple: boolean,
) {
  if (!ids.includes(id)) return { selected, anchor, multiple };
  const add = modifiers.ctrlKey || modifiers.metaKey;
  if (modifiers.shiftKey && ids.includes(anchor)) {
    const a = ids.indexOf(anchor),
      b = ids.indexOf(id);
    const range = ids.slice(Math.min(a, b), Math.max(a, b) + 1);
    return {
      selected: add ? [...new Set([...selected, ...range])] : range,
      anchor,
      multiple: true,
    };
  }
  if (multiple || add)
    return {
      selected: selected.includes(id)
        ? selected.filter((value) => value !== id)
        : [...selected, id],
      anchor: id,
      multiple: true,
    };
  return { selected: [id], anchor: id, multiple: false };
}
export function selectionRoots<T extends { id: string; parentId?: string }>(
  selected: T[],
  all: T[],
): T[] {
  const ids = new Set(selected.map((value) => value.id)),
    parents = new Map(all.map((value) => [value.id, value.parentId]));
  return selected.filter((value) => {
    let parent = value.parentId;
    const visited = new Set<string>();
    while (parent && !visited.has(parent)) {
      if (ids.has(parent)) return false;
      visited.add(parent);
      parent = parents.get(parent);
    }
    return true;
  });
}
