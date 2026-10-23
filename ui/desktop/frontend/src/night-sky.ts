import * as THREE from "three";
import type { RenderSky } from "./render-skies";



export function createNightSky(): RenderSky {
  const width = 1024,
    height = 512,
    data = new Float32Array(width * height * 4),
    fill = new Float32Array(data.length);
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++) {
      const i = (x + y * width) * 4,
        h = Math.max(0, Math.sin((y / height - 0.5) * Math.PI));
      data[i] = 0.0006 + 0.0012 * (1 - h);
      data[i + 1] = 0.001 + 0.002 * (1 - h);
      data[i + 2] = 0.0025 + 0.003 * (1 - h);
      fill[i] = 0.006;
      fill[i + 1] = 0.009;
      fill[i + 2] = 0.018;
      data[i + 3] = fill[i + 3] = 1;
    }
  let seed = 42839;
  const random = () => {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
    return seed / 4294967296;
  };
  for (let star = 0; star < 2200; star++) {
    const x = Math.floor(random() * width),
      y = Math.floor(
        (0.5 + Math.asin(random() * 1.96 - 0.98) / Math.PI) * height,
      );
    const brightness = 0.35 + Math.pow(random(), 5) * 9,
      warm = random();
    const at = (x + Math.min(height - 1, y) * width) * 4;
    data[at] += brightness * (0.7 + warm * 0.3);
    data[at + 1] += brightness * 0.85;
    data[at + 2] += brightness;
  }
  const texture = (pixels: Float32Array) => {
    const t = new THREE.DataTexture(
      pixels,
      width,
      height,
      THREE.RGBAFormat,
      THREE.FloatType,
    );
    t.mapping = THREE.EquirectangularReflectionMapping;
    t.colorSpace = THREE.LinearSRGBColorSpace;
    t.magFilter = t.minFilter = THREE.LinearFilter;
    t.wrapS = THREE.RepeatWrapping;
    t.needsUpdate = true;
    return t;
  };
  return {
    background: texture(data),
    lighting: texture(fill),
    sun: new THREE.Vector3(0.58, 0.74, 0.34).normalize(),
  };
}
