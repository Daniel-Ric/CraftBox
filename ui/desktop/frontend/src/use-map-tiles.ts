import { useEffect, useMemo, useRef, useState } from "react";
import {
  api,
  type Tile,
  type MapPageRequest,
  type MapTilePage,
} from "./bridge";
type DetailTile = Tile & { detailPixels: number };
function sameCoverage(
  a: [number, number][] | undefined,
  b: [number, number][],
) {
  if (!a || a.length !== b.length) return false;
  const keys = new Set(a.map((p) => p.join(",")));
  return b.every((p) => keys.has(p.join(",")));
}
function sameTile(a: Tile | undefined, b: Tile) {
  return !!a && a.version === b.version && a.image === b.image;
}

export function useMapTiles(
  worldId: string | undefined,
  fallback: Map<string, Tile>,
  y: number | undefined,
  detailBounds: [number, number, number, number] | undefined,
  pixels: number,
) {
  const [base, setBase] = useState(new Map<string, Tile>()),
    [details, setDetails] = useState(new Map<string, DetailTile>()),
    [coverage, setCoverage] = useState<[number, number][]>(),
    [loading, setLoading] = useState(false),
    [error, setError] = useState("");
  const sequence = useRef(0),
    requestGate = useRef<Promise<void>>(Promise.resolve());
  useEffect(() => {
    setBase(new Map());
    setDetails(new Map());
    setCoverage(undefined);
    sequence.current = 0;
    setError("");
    if (!worldId) {
      setLoading(false);
      return;
    }
    let active = true,
      timer: ReturnType<typeof setTimeout>;
    async function poll() {
      if (document.hidden) {
        timer = setTimeout(poll, 1500);
        return;
      }
      if (sequence.current === 0) setLoading(true);
      let buffered: Tile[] = [],
        bufferedCoverage: [number, number][] | undefined;
      let lastPublished = performance.now();
      function publish() {
        if (!active || (!buffered.length && !bufferedCoverage)) return;
        const batch = buffered,
          coverage = bufferedCoverage;
        buffered = [];
        bufferedCoverage = undefined;
        lastPublished = performance.now();
        setBase((old) => {
          let next: Map<string, Tile> | undefined;
          const edit = () => (next ??= new Map(old));
          if (coverage) {
            const kept = new Set(coverage.map((p) => p.join(",")));
            for (const k of old.keys()) if (!kept.has(k)) edit().delete(k);
          }
          for (const t of batch) {
            const k = `${t.x},${t.z}`;
            if (!sameTile(old.get(k), t)) edit().set(k, t);
          }
          return next ?? old;
        });
      }
      try {
        let cursor: [number, number] | null = null,
          target = 0;
        do {
          const request: MapPageRequest = {
            after: sequence.current,
            target,
            cursor,
            y,
            pixels: 1,
            limit: 24,
          };
          const page = await api.GetMapPage(worldId!, request);
          if (!active) return;
          if (page.coverage)
            setCoverage((old) =>
              sameCoverage(old, page.coverage!) ? old : page.coverage!,
            );
          target = page.sequence;
          if (page.coverage) bufferedCoverage = page.coverage;
          buffered.push(...page.tiles);
          if (page.complete || performance.now() - lastPublished >= 100)
            publish();
          cursor = page.next;
          if (page.complete) break;
        } while (active);
        sequence.current = target;
        setError("");
      } catch (e) {
        if (active) setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (active) {
          publish();
          setLoading(false);
          timer = setTimeout(poll, 1500);
        }
      }
    }
    void poll();
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [worldId, y]);
  const key = detailBounds?.join(",");
  useEffect(() => {
    if (!worldId || !detailBounds || pixels <= 1) return;
    let active = true,
      timer: ReturnType<typeof setTimeout>,
      after = 0;
    const requestBounds = [...detailBounds] as [number, number, number, number];
    async function update() {
      try {
        if (document.hidden || (after !== 0 && sequence.current <= after))
          return;
        let cursor: [number, number] | null = null,
          target = 0;
        do {
                                                                                     
          const previous = requestGate.current;
          let release!: () => void;
          requestGate.current = new Promise((r) => {
            release = r;
          });
          await previous;
          let page: MapTilePage | undefined;
          try {
            if (!active) return;
            page = await api.GetMapPage(worldId!, {
              after,
              target,
              cursor,
              y,
              pixels,
              bounds: requestBounds,
              limit: 8,
              omitCoverage: true,
            });
          } finally {
            release();
          }
          if (!active || !page) return;
          target = page.sequence;
          setDetails((old) => {
            let next: Map<string, DetailTile> | undefined;
            for (const t of page!.tiles) {
              const k = `${t.x},${t.z}`;
              if (!sameTile(old.get(k), t)) {
                next ??= new Map(old);
                next.delete(k);
                next.set(k, { ...t, detailPixels: pixels });
              }
            }
            if (next) {
              let budget = 0,
                count = 0;
              for (const [k, t] of [...next].reverse()) {
                const cost = (16 * t.detailPixels) ** 2;
                if (budget + cost > 16 * 1024 * 1024 || count >= 512)
                  next.delete(k);
                else {
                  budget += cost;
                  count++;
                }
              }
            }
            return next ?? old;
          });
          cursor = page.next;
          if (page.complete) break;
        } while (active);
        after = target;
      } catch {
        /* Keep existing detail and base images after transient failures. */
      } finally {
        if (active) timer = setTimeout(() => void update(), 900);
      }
    }
    timer = setTimeout(() => void update(), 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [worldId, y, key, pixels]);
  const covered = useMemo(
    () => (coverage ? new Set(coverage.map((p) => p.join(","))) : undefined),
    [coverage],
  );
  const merged = useMemo(() => {
    const out = new Map<string, Tile>();
    if (y === undefined)
      for (const t of fallback.values()) out.set(`${t.x},${t.z}`, t);
    for (const [k, t] of base) out.set(k, t);
    for (const [k, t] of details)
      if (
        (!covered || covered.has(k)) &&
        (!base.get(k)?.version || t.version === base.get(k)?.version)
      )
        out.set(k, t);
    return out;
  }, [fallback, base, details, y, covered]);
  return { tiles: merged, coverage, loading, error };
}
