import type { LabChange } from "./bedrock-lab-model";

export type DiffLine = {
  kind: "context" | "added" | "removed";
  text: string;
  oldLine?: number;
  newLine?: number;
};
export function lineDiff(before: string, after: string) {
  const lines = (s: string) => (s ? s.replace(/\r\n/g, "\n").split("\n") : []);
  const allA = lines(before),
    allB = lines(after),
    limited = allA.length > 20000 || allB.length > 20000;
  const a = allA.slice(0, 20000),
    b = allB.slice(0, 20000),
    rows: DiffLine[] = [];
  let coarse = false,
    budget = 1_000_000;
  const emit = (kind: DiffLine["kind"], i: number, j: number) =>
    rows.push({
      kind,
      text: kind === "added" ? b[j] : a[i],
      oldLine: kind === "added" ? undefined : i + 1,
      newLine: kind === "removed" ? undefined : j + 1,
    });
  function range(ai: number, ae: number, bi: number, be: number, depth = 0) {
    while (ai < ae && bi < be && a[ai] === b[bi]) emit("context", ai++, bi++);
    let tail = 0;
    while (
      ai < ae - tail &&
      bi < be - tail &&
      a[ae - tail - 1] === b[be - tail - 1]
    )
      tail++;
    const ax = ae - tail,
      bx = be - tail,
      n = ax - ai,
      m = bx - bi;
    if (!n) for (let j = bi; j < bx; j++) emit("added", ai, j);
    else if (!m) for (let i = ai; i < ax; i++) emit("removed", i, bi);
    else if (n * m <= budget) {
      budget -= n * m;
      const dp = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1));
      for (let i = n - 1; i >= 0; i--)
        for (let j = m - 1; j >= 0; j--)
          dp[i][j] =
            a[ai + i] === b[bi + j]
              ? dp[i + 1][j + 1] + 1
              : Math.max(dp[i + 1][j], dp[i][j + 1]);
      let i = 0,
        j = 0;
      while (i < n || j < m) {
        if (i < n && j < m && a[ai + i] === b[bi + j]) {
          emit("context", ai + i++, bi + j++);
        } else if (i < n && (j === m || dp[i + 1][j] >= dp[i][j + 1]))
          emit("removed", ai + i++, bi + j);
        else emit("added", ai + i, bi + j++);
      }
    } else {
      const unique = (v: string[], start: number, end: number) => {
        const map = new Map<string, number>();
        for (let i = start; i < end; i++) map.set(v[i], map.has(v[i]) ? -1 : i);
        return map;
      };
      const left = unique(a, ai, ax),
        right = unique(b, bi, bx);
      const pairs: [number, number][] = [];
      for (const [s, i] of left) {
        const j = right.get(s);
        if (i >= 0 && j !== undefined && j >= 0) pairs.push([i, j]);
      }
      const tails: number[] = [],
        prev = new Int32Array(pairs.length).fill(-1);
      for (let k = 0; k < pairs.length; k++) {
        let l = 0,
          r = tails.length;
        while (l < r) {
          const mid = (l + r) >> 1;
          if (pairs[tails[mid]][1] < pairs[k][1]) l = mid + 1;
          else r = mid;
        }
        if (l) prev[k] = tails[l - 1];
        tails[l] = k;
      }
      const anchors: [number, number][] = [];
      if (depth < 32)
        for (let k = tails.at(-1) ?? -1; k >= 0; k = prev[k])
          anchors.push(pairs[k]);
      anchors.reverse();
      if (anchors.length) {
        let i = ai,
          j = bi;
        for (const [x, y] of anchors) {
          range(i, x, j, y, depth + 1);
          emit("context", x, y);
          i = x + 1;
          j = y + 1;
        }
        range(i, ax, j, bx, depth + 1);
      } else {
        coarse = true;
        for (let i = ai; i < ax; i++) emit("removed", i, bi);
        for (let j = bi; j < bx; j++) emit("added", ax, j);
      }
    }
    for (let i = 0; i < tail; i++) emit("context", ax + i, bx + i);
  }
  range(0, a.length, 0, b.length);
  return {
    rows,
    coarse,
    limited,
    added: rows.filter((r) => r.kind === "added").length,
    removed: rows.filter((r) => r.kind === "removed").length,
  };
}
export function diffHunks(rows: DiffLine[], context = 3) {
  const ranges: { start: number; end: number }[] = [];
  rows.forEach((row, i) => {
    if (row.kind === "context") return;
    const start = Math.max(0, i - context),
      end = Math.min(rows.length, i + context + 1);
    const last = ranges.at(-1);
    if (last && start <= last.end) last.end = end;
    else ranges.push({ start, end });
  });
  return ranges;
}
export function packPath(path: string) {
  const parts = path.split("/");
  return /^(resource_packs|behavior_packs)$/.test(parts[0])
    ? parts.slice(0, 2).join("/")
    : parts[0];
}
export function changeSummary(changes: Pick<LabChange, "path" | "status">[]) {
  const counts: Record<string, number> = {},
    packs = new Map<string, number>();
  let containers = 0,
    formatting = 0,
    content = 0;
  for (const c of changes) {
    if (c.status === "formatting") {
      formatting++;
      continue;
    }
    if (c.path.toLowerCase().endsWith(".brarchive")) {
      containers++;
      continue;
    }
    counts[c.status] = (counts[c.status] ?? 0) + 1;
    content++;
    const p = packPath(c.path);
    packs.set(p, (packs.get(p) ?? 0) + 1);
  }
  return {
    counts,
    containers,
    formatting,
    content,
    packs: [...packs]
      .map(([path, count]) => ({ path, count }))
      .sort((a, b) => b.count - a.count || a.path.localeCompare(b.path)),
  };
}
