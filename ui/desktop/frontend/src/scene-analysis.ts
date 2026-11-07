import type { BufferGeometry } from "three";
import {
  analyseModel,
  type ModelAnalysis,
  type SurfaceSample,
} from "./auto-direction";

export type AnalysisProgress = {
  stage:
    | "prepare"
    | "surfaces"
    | "features"
    | "composition"
    | "context"
    | "matching"
    | "done";
  label: string;
  detail?: string;
  completed?: number;
  total?: number;
  cached?: boolean;
};
export type AnalysisProgressCallback = (
  value: number,
  status?: AnalysisProgress,
) => void;
export type AnalysisMesh = {
  geometry: BufferGeometry;
  mode: string;
  offset: readonly number[];
};
export type AnalysisEvidence = Parameters<typeof analyseModel>[1];
export type PackedSamples = {
  data: Float32Array;
  modes: string[];
  count: number;
  triangles: number;
};
const fields = 11;
export function checkAnalysisAbort(signal?: AbortSignal) {
  if (signal?.aborted)
    throw new DOMException("Analysis cancelled", "AbortError");
}
export const analysisYield = () =>
  new Promise<void>((resolve) => setTimeout(resolve, 0));

/** A bounded, deterministic stratum per mesh/material, weighted by the exact
 * number of triangles represented. No full geometry copies or GPU readbacks. */
export async function sampleScene(
  meshes: readonly AnalysisMesh[],
  atlas: { pixels: Uint8Array; width: number; height: number },
  progress?: AnalysisProgressCallback,
  signal?: AbortSignal,
  budget = 48000,
): Promise<PackedSamples> {
  checkAnalysisAbort(signal);
  progress?.(0.02, {
    stage: "prepare",
    label: "Indexing loaded chunks and materials",
  });
  await analysisYield();
  checkAnalysisAbort(signal);
  const limit = Math.max(1, Math.min(96000, Math.floor(budget)));
  const available = meshes
    .map((mesh, order) => ({
      ...mesh,
      order,
      triangles: Math.floor(
        (mesh.geometry.index?.count ??
          mesh.geometry.attributes.position?.count ??
          0) / 3,
      ),
      quota: 1,
      fraction: 0,
    }))
    .filter(
      (m) =>
        m.triangles > 0 &&
        m.geometry.attributes.uv &&
        m.geometry.attributes.normal,
    );
  const triangles = available.reduce((sum, m) => sum + m.triangles, 0);
  // Pathological scenes with more materials than probes are distributed over
  // the entire sorted scene instead of truncating its tail.
  const jobs =
    available.length > limit
      ? Array.from(
          { length: limit },
          (_, i) =>
            available[Math.floor(((i + 0.5) * available.length) / limit)],
        )
      : available;
  const total = jobs.reduce((sum, m) => sum + m.triangles, 0);
  const target = Math.min(limit, total),
    extra = target - jobs.length,
    weight = total - jobs.length;
  for (const job of jobs) {
    const share = weight ? (extra * (job.triangles - 1)) / weight : 0;
    job.quota += Math.floor(share);
    job.fraction = share % 1;
  }
  let remainder = target - jobs.reduce((sum, m) => sum + m.quota, 0);
  for (const job of [...jobs].sort(
    (a, b) => b.fraction - a.fraction || a.order - b.order,
  )) {
    if (remainder > 0 && job.quota < job.triangles) {
      job.quota++;
      remainder--;
    }
  }
  const data = new Float32Array(target * fields),
    modes: string[] = [];
  let count = 0,
    visited = 0,
    lastYield = performance.now(),
    lastReport = -Infinity;
  const report = (finished = false) => {
    const now = performance.now();
    if (!finished && now - lastReport < 50) return;
    lastReport = now;
    progress?.(0.05 + (0.65 * visited) / Math.max(1, target), {
      stage: "surfaces",
      label: "Sampling geometry, textures and material families",
      detail: `${visited.toLocaleString()} / ${target.toLocaleString()} surface probes · ${jobs.length.toLocaleString()} meshes`,
      completed: visited,
      total: target,
    });
  };
  for (const job of jobs) {
    checkAnalysisAbort(signal);
    const { geometry, offset } = job,
      p = geometry.attributes.position,
      n = geometry.attributes.normal,
      uv = geometry.attributes.uv,
      index = geometry.index;
    let mode = modes.indexOf(job.mode);
    if (mode < 0) {
      mode = modes.length;
      modes.push(job.mode);
    }
    for (let slot = 0; slot < job.quota; slot++) {
      const start = Math.floor((slot * job.triangles) / job.quota),
        end = Math.floor(((slot + 1) * job.triangles) / job.quota);
      const j =
        start + ((Math.imul(slot + 17, job.order + 31) >>> 0) % (end - start));
      const a = index ? index.getX(j * 3) : j * 3,
        b = index ? index.getX(j * 3 + 1) : j * 3 + 1,
        c = index ? index.getX(j * 3 + 2) : j * 3 + 2;
      const bx = p.getX(b) - p.getX(a),
        by = p.getY(b) - p.getY(a),
        bz = p.getZ(b) - p.getZ(a),
        cx = p.getX(c) - p.getX(a),
        cy = p.getY(c) - p.getY(a),
        cz = p.getZ(c) - p.getZ(a);
      let area =
        Math.hypot(by * cz - bz * cy, bz * cx - bx * cz, bx * cy - by * cx) *
        0.5 *
        (end - start);
      // If strata had to be omitted, each selected mesh represents its interval.
      if (available.length > jobs.length)
        area *= available.length / jobs.length;
      const u = uv.getX(a) * 0.23 + uv.getX(b) * 0.31 + uv.getX(c) * 0.46,
        v = uv.getY(a) * 0.23 + uv.getY(b) * 0.31 + uv.getY(c) * 0.46;
      const pixel =
        (Math.max(0, Math.min(atlas.height - 1, Math.floor(v * atlas.height))) *
          atlas.width +
          Math.max(0, Math.min(atlas.width - 1, Math.floor(u * atlas.width)))) *
        4;
      if (
        area > 1e-9 &&
        Number.isFinite(area) &&
        atlas.pixels[pixel + 3] >= 64
      ) {
        const at = count * fields;
        data[at] =
          p.getX(a) * 0.23 +
          p.getX(b) * 0.31 +
          p.getX(c) * 0.46 +
          (offset[0] ?? 0);
        data[at + 1] =
          p.getY(a) * 0.23 +
          p.getY(b) * 0.31 +
          p.getY(c) * 0.46 +
          (offset[1] ?? 0);
        data[at + 2] =
          p.getZ(a) * 0.23 +
          p.getZ(b) * 0.31 +
          p.getZ(c) * 0.46 +
          (offset[2] ?? 0);
        data[at + 3] = n.getX(a) * 0.23 + n.getX(b) * 0.31 + n.getX(c) * 0.46;
        data[at + 4] = n.getY(a) * 0.23 + n.getY(b) * 0.31 + n.getY(c) * 0.46;
        data[at + 5] = n.getZ(a) * 0.23 + n.getZ(b) * 0.31 + n.getZ(c) * 0.46;
        data[at + 6] = atlas.pixels[pixel] / 255;
        data[at + 7] = atlas.pixels[pixel + 1] / 255;
        data[at + 8] = atlas.pixels[pixel + 2] / 255;
        data[at + 9] = area;
        data[at + 10] = mode;
        count++;
      }
      visited++;
      if (visited % 256 === 0) {
        report();
        checkAnalysisAbort(signal);
        if (performance.now() - lastYield >= 6) {
          await analysisYield();
          checkAnalysisAbort(signal);
          lastYield = performance.now();
        }
      }
    }
  }
  report(true);
  checkAnalysisAbort(signal);
  return { data: data.slice(0, count * fields), modes, count, triangles };
}

export function unpackSamples(packed: PackedSamples): SurfaceSample[] {
  const out: SurfaceSample[] = [];
  for (let i = 0; i < packed.count; i++) {
    const a = i * fields,
      d = packed.data;
    out.push({
      position: [d[a], d[a + 1], d[a + 2]],
      normal: [d[a + 3], d[a + 4], d[a + 5]],
      color: [d[a + 6], d[a + 7], d[a + 8]],
      area: d[a + 9],
      mode: packed.modes[d[a + 10]] ?? "solid",
    });
  }
  return out;
}

export class AnalysisCache {
  private entry?: { revision: number; source: string; value: ModelAnalysis };
  get(revision: number, source = "") {
    return this.entry?.revision === revision && this.entry.source === source
      ? structuredClone(this.entry.value)
      : undefined;
  }
  set(revision: number, source: string | undefined, value: ModelAnalysis) {
    this.entry = {
      revision,
      source: source ?? "",
      value: structuredClone(value),
    };
  }
}

/** Transfer only bounded samples; millions of terrain vertices stay in place. */
export async function analysePacked(
  packed: PackedSamples,
  evidence: AnalysisEvidence,
  progress?: AnalysisProgressCallback,
  signal?: AbortSignal,
): Promise<ModelAnalysis> {
  checkAnalysisAbort(signal);
  progress?.(0.73, {
    stage: "features",
    label: "Measuring palette, surface normals and vegetation",
    detail: `${packed.count.toLocaleString()} sampled surfaces`,
  });
  await analysisYield();
  checkAnalysisAbort(signal);
  if (typeof Worker === "undefined")
    return analyseModel(unpackSamples(packed), evidence, (value, status) =>
      progress?.(value, status),
    );
  return new Promise((resolve, reject) => {
    let worker: Worker;
    const cleanup = () => {
      worker?.terminate();
      signal?.removeEventListener("abort", abort);
    };
    const abort = () => {
      cleanup();
      reject(new DOMException("Analysis cancelled", "AbortError"));
    };
    try {
      worker = new Worker(
        new URL("./scene-analysis.worker.ts", import.meta.url),
        { type: "module" },
      );
      signal?.addEventListener("abort", abort, { once: true });
      worker.onmessage = ({ data }) => {
        if (signal?.aborted) {
          abort();
          return;
        }
        if (data.progress) {
          try {
            progress?.(data.progress.value, data.progress.status);
          } catch (error) {
            cleanup();
            reject(error);
          }
          return;
        }
        cleanup();
        if (data.error) reject(new Error(data.error));
        else resolve(data.result);
      };
      worker.onerror = () => {
        cleanup();
        reject(
          new Error(
            "Could not run background analysis. Reopen the editor and try again.",
          ),
        );
      };
      worker.postMessage({ packed, evidence }, [packed.data.buffer]);
    } catch (error) {
      cleanup();
      reject(error);
    }
  });
}
