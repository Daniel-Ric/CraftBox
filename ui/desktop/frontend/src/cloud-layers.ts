import {
  parseScalarTracks,
  sampleTrack,
  upsertTrackKey,
  type AnimationTrack,
} from "./animation-tracks";
export type CloudVector = [number, number, number];

export function unwrapCloudRotation(
  canonical: CloudVector,
  reference: CloudVector,
): CloudVector {
  const unwrap = (v: number, r: number) => v + 360 * Math.round((r - v) / 360);
  const candidates = [
    canonical,
    [canonical[0] + 180, 180 - canonical[1], canonical[2] + 180] as CloudVector,
  ].map((v) => v.map((n, i) => unwrap(n, reference[i])) as CloudVector);
  const error = (v: CloudVector) =>
    v.reduce((sum, n, i) => sum + (n - reference[i]) ** 2, 0);
  return error(candidates[0]) <= error(candidates[1])
    ? candidates[0]
    : candidates[1];
}
export const cloudChannelLimits = {
  "position.x": [-100, 100],
  "position.y": [-100, 100],
  "position.z": [-100, 100],
  "size.x": [0.01, 100],
  "size.y": [0.01, 100],
  "size.z": [0.01, 100],
  "rotation.x": [-36000, 36000],
  "rotation.y": [-36000, 36000],
  "rotation.z": [-36000, 36000],
  density: [0, 4],
  coverage: [0, 1],
  dissolve: [0, 1],
} satisfies Record<string, [number, number]>;
export type CloudChannel = keyof typeof cloudChannelLimits;
export const cloudChannelLabels: Record<CloudChannel, string> = {
  "position.x": "Cloud position X",
  "position.y": "Cloud position Y",
  "position.z": "Cloud position Z",
  "size.x": "Cloud width",
  "size.y": "Cloud height",
  "size.z": "Cloud depth",
  "rotation.x": "Cloud rotation X",
  "rotation.y": "Cloud rotation Y",
  "rotation.z": "Cloud rotation Z",
  density: "Cloud density",
  coverage: "Cloud cover",
  dissolve: "Cloud breakup",
};
export type CloudLayer = {
  id: string;
  name: string;
  kind: "bank" | "volume";
  enabled: boolean;

  position: CloudVector;
  size: CloudVector;
  rotation: CloudVector;
  color: string;
  density: number;
  coverage: number;
  dissolve?: number;
  detail: number;
  erosion: number;
  turbulence: number;
  seed: number;
  softness: number;
  shadows: number;
  windSpeed: number;
  windDirection: number;
  tracks: AnimationTrack<CloudChannel>[];
};
export type CloudOpening = {
  strength: number;
  center: [number, number];
  radius: [number, number];
  floor: number;
  softness: number;
  irregularity: number;
};
export const cloudOpeningDefaults: CloudOpening = {
  strength: 0,
  center: [0, 0],
  radius: [0.65, 0.65],
  floor: -0.5,
  softness: 0.25,
  irregularity: 0.35,
};
const finite = (v: unknown, min: number, max: number): v is number =>
  typeof v === "number" && Number.isFinite(v) && v >= min && v <= max;
export function parseCloudLayers(raw: unknown): CloudLayer[] {
  if (raw === undefined) return [];
  if (!Array.isArray(raw) || raw.length > 8)
    throw Error("You can add up to eight cloud regions.");
  const ids = new Set<string>();
  return raw.map((o: CloudLayer) => {
    const vector = (v: unknown, min: number, max: number) =>
      Array.isArray(v) && v.length === 3 && v.every((n) => finite(n, min, max));
    if (
      !o ||
      typeof o.id !== "string" ||
      !o.id ||
      o.id.length > 100 ||
      ids.has(o.id) ||
      typeof o.name !== "string" ||
      !o.name.trim() ||
      o.name.length > 200 ||
      !["bank", "volume"].includes(o.kind) ||
      typeof o.enabled !== "boolean" ||
      !vector(o.position, -100, 100) ||
      !vector(o.size, 0.01, 100) ||
      !vector(o.rotation, -36000, 36000) ||
      !/^#[a-f\d]{6}$/i.test(o.color) ||
      !finite(o.density, 0, 4) ||
      !finite(o.coverage, 0, 1) ||
      !finite(o.dissolve ?? 0, 0, 1) ||
      !finite(o.seed, 0, 999999) ||
      ![o.detail, o.erosion, o.turbulence, o.softness, o.shadows].every((n) =>
        finite(n, 0, 1),
      ) ||
      !finite(o.windSpeed, 0, 40) ||
      !finite(o.windDirection, 0, 360)
    )
      throw Error("Invalid cloud region.");
    ids.add(o.id);
    return {
      ...o,
      position: [...o.position],
      size: [...o.size],
      rotation: [...o.rotation],
      tracks: parseScalarTracks(o.tracks ?? [], 3600, cloudChannelLimits),
    };
  });
}
export function parseCloudOpening(raw: unknown): CloudOpening {
  const o = { ...cloudOpeningDefaults, ...(raw as Partial<CloudOpening>) };
  if (
    ![o.strength, o.irregularity].every((n) => finite(n, 0, 1)) ||
    !finite(o.softness, 0.01, 1) ||
    !finite(o.floor, -100, 100) ||
    !Array.isArray(o.center) ||
    o.center.length !== 2 ||
    !o.center.every((n) => finite(n, -100, 100)) ||
    !Array.isArray(o.radius) ||
    o.radius.length !== 2 ||
    !o.radius.every((n) => finite(n, 0.01, 100))
  )
    throw Error("Invalid cloud opening.");
  return { ...o, center: [...o.center], radius: [...o.radius] };
}
export function createCloudLayer(kind: CloudLayer["kind"]): CloudLayer {
  return {
    id: crypto.randomUUID(),
    name: kind === "bank" ? "Cloud layer" : "Cloud volume",
    kind,
    enabled: true,
    position: [0, -0.5, 0],
    size: kind === "bank" ? [3, 0.7, 3] : [0.8, 0.5, 0.8],
    rotation: [0, 0, 0],
    color: "#eef6ff",
    density: 1,
    coverage: 0.7,
    detail: 0.6,
    erosion: 0.35,
    turbulence: 0.4,
    seed: Math.floor(Math.random() * 100000),
    softness: 0.3,
    shadows: 0.8,
    windSpeed: 0,
    windDirection: 90,
    tracks: [],
  };
}
export function cloudChannelValue(
  layer: CloudLayer,
  channel: CloudChannel,
): number {
  const [property, axis] = channel.split(".");
  return axis
    ? layer[property as "position" | "size" | "rotation"][
        ["x", "y", "z"].indexOf(axis)
      ]
    : (layer[channel as "density" | "coverage" | "dissolve"] ?? 0);
}
export function sampleCloudLayer(layer: CloudLayer, time: number): CloudLayer {
  const next = {
    ...layer,
    position: [...layer.position] as CloudVector,
    size: [...layer.size] as CloudVector,
    rotation: [...layer.rotation] as CloudVector,
  };
  for (const track of layer.tracks) {
    const [lo, hi] = cloudChannelLimits[track.channel],
      value = Math.max(
        lo,
        Math.min(
          hi,
          sampleTrack(
            track,
            time,
            track.defaultValue ?? cloudChannelValue(layer, track.channel),
          ),
        ),
      );
    const [property, axis] = track.channel.split(".");
    if (axis)
      next[property as "position" | "size" | "rotation"][
        ["x", "y", "z"].indexOf(axis)
      ] = value;
    else next[property as "density" | "coverage" | "dissolve"] = value;
  }
  return next;
}
export function keyCloudValue(
  layer: CloudLayer,
  channel: CloudChannel,
  time: number,
  value: number,
  fps: number,
): CloudLayer {
  const [lo, hi] = cloudChannelLimits[channel];
  const track = layer.tracks.find((t) => t.channel === channel) ?? {
    channel,
    keys: [],
    defaultValue: cloudChannelValue(layer, channel),
  };
  const keyed = upsertTrackKey(
    track,
    {
      id: crypto.randomUUID(),
      time,
      value: Math.max(lo, Math.min(hi, value)),
      interpolation: "bezier",
      incoming: [0, 0],
      outgoing: [0, 0],
    },
    fps,
  );
  return {
    ...layer,
    tracks: [...layer.tracks.filter((t) => t.channel !== channel), keyed],
  };
}
