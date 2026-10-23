import * as THREE from "three";
import { createNightSky } from "./night-sky";
import { HDRLoader } from "three/addons/loaders/HDRLoader.js";

export const renderSkies = {
  daylight: {
    label: "Partly cloudy day",
    file: "kloofendal_48d_partly_cloudy_puresky_2k.hdr",
    source: "https://polyhaven.com/a/kloofendal_48d_partly_cloudy_puresky",
  },
  dawn: {
    label: "Morning & evening light",
    file: "kloppenheim_06_puresky_2k.hdr",
    source: "https://polyhaven.com/a/kloppenheim_06_puresky",
  },
  sunset: {
    label: "Sunset",
    file: "belfast_sunset_puresky_2k.hdr",
    source: "https://polyhaven.com/a/belfast_sunset_puresky",
  },
  night: {
    label: "Starry sky with moon",
    file: "rogland_clear_night_2k.hdr",
    source: "https://polyhaven.com/a/rogland_clear_night",
  },
};
export type RenderSky = {
  background: THREE.DataTexture;
  lighting: THREE.DataTexture;
  sun: THREE.Vector3;
};

export async function loadRenderSky(
  id: string,
  file?: File,
): Promise<RenderSky> {
  if (id === "night") return createNightSky();
  let url: string;
  if (id === "custom") {
    if (!file) throw new Error("Select an HDR skybox first");
    if (file.size > 64 * 1024 * 1024)
      throw new Error("The HDR skybox must not exceed 64 MiB");
    url = URL.createObjectURL(file);
  } else {
    const sky = renderSkies[id as keyof typeof renderSkies];
    if (!sky) throw new Error("Unknown skybox");
    url = `${import.meta.env.BASE_URL}skies/${sky.file}`;
  }
  let texture: THREE.DataTexture;
  try {
    texture = await new HDRLoader().setDataType(THREE.FloatType).loadAsync(url);
  } finally {
    if (id === "custom") URL.revokeObjectURL(url);
  }
  const { width, height, data } = texture.image as {
    width: number;
    height: number;
    data: Float32Array;
  };
  if (
    width > 8192 ||
    height > 4096 ||
    width * height > 16 * 1024 * 1024 ||
    Math.abs(width / height - 2) > 0.1
  ) {
    texture.dispose();
    throw new Error(
      "The skybox must be a 2:1 HDR panorama of up to 16 megapixels",
    );
  }


  let log = 0,
    weight = 0,
    peak = -1,
    peakX = width / 2,
    peakY = height / 4;
  for (let y = 0; y < height / 2; y += 2) {
    const w = Math.sin(((y + 0.5) / height) * Math.PI);
    for (let x = 0; x < width; x += 2) {
      const i = (y * width + x) * 4;
      const l = 0.2126 * data[i] + 0.7152 * data[i + 1] + 0.0722 * data[i + 2];
      if (!Number.isFinite(l) || l < 0) {
        texture.dispose();
        throw new Error("The HDR skybox contains invalid brightness values");
      }
      log += Math.log(Math.max(1e-5, l)) * w;
      weight += w;
      if (l > peak) {
        peak = l;
        peakX = x;
        peakY = y;
      }
    }
  }
  const gain = Math.min(1000, 0.45 / Math.exp(log / Math.max(1, weight)));
  const lighting = new Float32Array(data.length);
  for (let i = 0; i < data.length; i += 4) {
    for (let c = 0; c < 3; c++) {
      const value = data[i + c];
      if (!Number.isFinite(value) || value < 0) {
        texture.dispose();
        throw new Error("Invalid HDR pixel data");
      }
      data[i + c] = Math.min(65504, value * gain);
      lighting[i + c] = Math.min(8, data[i + c]);
    }
    data[i + 3] = lighting[i + 3] = 1;
  }
  texture.mapping = THREE.EquirectangularReflectionMapping;
  texture.colorSpace = THREE.LinearSRGBColorSpace;
  texture.needsUpdate = true;
  const light = new THREE.DataTexture(
    lighting,
    width,
    height,
    THREE.RGBAFormat,
    THREE.FloatType,
  );
  light.mapping = THREE.EquirectangularReflectionMapping;
  light.colorSpace = THREE.LinearSRGBColorSpace;
  light.flipY = texture.flipY;
  light.needsUpdate = true;
  const theta = ((peakY + 0.5) / height) * Math.PI,
    phi = ((peakX + 0.5) / width - 0.5) * Math.PI * 2;
  return {
    background: texture,
    lighting: light,
    sun: new THREE.Vector3(
      Math.sin(theta) * Math.cos(phi),
      Math.cos(theta),
      Math.sin(theta) * Math.sin(phi),
    ),
  };
}
