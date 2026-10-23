import { DataUtils } from "three";
type Vector = readonly [number, number, number];
export type PixelImage = {
  width: number;
  height: number;
  data: Uint8ClampedArray | Float32Array;
};
export function decodeHDRFace(encoded: Uint16Array, size: number): PixelImage {
  if (encoded.length !== size * size * 4)
    throw new Error("Invalid HDR buffer.");
  const data = new Float32Array(encoded.length);
  for (let y = 0; y < size; y++)
    for (let x = 0; x < size; x++) {
      const source = ((size - y - 1) * size + x) * 4,
        at = (y * size + x) * 4;
      const alpha = Math.max(
        0,
        Math.min(1, DataUtils.fromHalfFloat(encoded[source + 3])),
      );
      for (let c = 0; c < 3; c++)
        data[at + c] =
          alpha > 0.00001
            ? DataUtils.fromHalfFloat(encoded[source + c]) / alpha
            : 0;
      data[at + 3] = alpha;
    }
  return { width: size, height: size, data };
}
export const cubeFaces: { direction: Vector; up: Vector }[] = [
  { direction: [1, 0, 0], up: [0, 1, 0] },
  { direction: [-1, 0, 0], up: [0, 1, 0] },
  { direction: [0, 1, 0], up: [0, 0, 1] },
  { direction: [0, -1, 0], up: [0, 0, -1] },
  { direction: [0, 0, 1], up: [0, 1, 0] },
  { direction: [0, 0, -1], up: [0, 1, 0] },
];
const dot = (a: Vector, b: Vector) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
const cross = (a: Vector, b: Vector): Vector => [
  a[1] * b[2] - a[2] * b[1],
  a[2] * b[0] - a[0] * b[2],
  a[0] * b[1] - a[1] * b[0],
];
export function equirectangularDirection(u: number, v: number): Vector {
  const longitude = (u - 0.5) * Math.PI * 2,
    latitude = (0.5 - v) * Math.PI;
  return [
    Math.sin(longitude) * Math.cos(latitude),
    Math.sin(latitude),
    -Math.cos(longitude) * Math.cos(latitude),
  ];
}
export function projectCubeDirection(direction: Vector, fov = 92) {
  let face = 0,
    forward = -Infinity;
  cubeFaces.forEach((candidate, i) => {
    const distance = dot(direction, candidate.direction);
    if (distance > forward) {
      face = i;
      forward = distance;
    }
  });
  const camera = cubeFaces[face],
    extent = Math.tan((fov * Math.PI) / 360);
  return {
    face,
    u:
      0.5 +
      dot(direction, cross(camera.direction, camera.up)) /
        (2 * forward * extent),
    v: 0.5 - dot(direction, camera.up) / (2 * forward * extent),
  };
}


export async function stitchPanorama(
  faces: PixelImage[],
  width: number,
  progress?: (value: number) => void,
  cancelled?: () => boolean,
): Promise<PixelImage> {
  if (
    !Number.isInteger(width) ||
    width < 64 ||
    width > 4096 ||
    width % 2 ||
    faces.length !== 6 ||
    faces.some(
      (f) =>
        f.width < 2 ||
        f.width !== f.height ||
        f.data.length !== f.width * f.height * 4,
    )
  )
    throw new Error("Invalid panorama images or output size.");
  const height = width / 2,
    hdr = faces[0].data instanceof Float32Array;
  const data = hdr
    ? new Float32Array(width * height * 4)
    : new Uint8ClampedArray(width * height * 4);
  const alphaScale = hdr ? 1 : 255;
  for (let y = 0; y < height; y++) {
    if (cancelled?.()) throw new Error("Panorama export canceled.");
    for (let x = 0; x < width; x++) {
      const projection = projectCubeDirection(
        equirectangularDirection((x + 0.5) / width, (y + 0.5) / height),
      );
      const face = faces[projection.face];
      const px = Math.max(
          0,
          Math.min(face.width - 1, projection.u * face.width - 0.5),
        ),
        py = Math.max(
          0,
          Math.min(face.height - 1, projection.v * face.height - 0.5),
        );
      const ix = Math.floor(px),
        iy = Math.floor(py),
        fx = px - ix,
        fy = py - iy;
      const color = [0, 0, 0];
      let alpha = 0;
      for (let sy = 0; sy < 2; sy++)
        for (let sx = 0; sx < 2; sx++) {
          const at =
            (Math.min(face.height - 1, iy + sy) * face.width +
              Math.min(face.width - 1, ix + sx)) *
            4;
          const weight = (sx ? fx : 1 - fx) * (sy ? fy : 1 - fy),
            a = face.data[at + 3] / alphaScale;
          alpha += a * weight;
          for (let channel = 0; channel < 3; channel++)
            color[channel] += face.data[at + channel] * a * weight;
        }
      const at = (y * width + x) * 4;
      for (let channel = 0; channel < 3; channel++)
        data[at + channel] = alpha > 0 ? color[channel] / alpha : 0;
      data[at + 3] = alpha * alphaScale;
    }
    if (y % 16 === 0) {
      progress?.(y / height);
      await new Promise<void>((resolve) => setTimeout(resolve, 0));
    }
  }
  progress?.(1);
  return { width, height, data };
}
