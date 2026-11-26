import * as THREE from "three";
import type { RenderSettings } from "./render-settings";
import { receiverBlocks, type StudioLight } from "./studio-lights";

export type OfflineOutput = "png" | "exr";
export type OfflineScene = {
  model: string;
  config: string;
};

function encodeBlob(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(",", 2)[1]);
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(blob);
  });
}

async function encodeEnvironment(sky?: THREE.DataTexture) {
  if (!sky) return;
  const { EXRExporter } = await import("three/addons/exporters/EXRExporter.js");
  const source = sky.image as {
    width: number;
    height: number;
    data: Float32Array;
  };
  const width = Math.min(2048, source.width);
  const height = Math.max(
    1,
    Math.round((width * source.height) / source.width),
  );
  if (width === source.width && !sky.flipY)
    return encodeBlob(new Blob([await new EXRExporter().parse(sky)]));
  const data = new Float32Array(width * height * 4);
  for (let y = 0; y < height; y++) {
    const row = Math.floor((y * source.height) / height);
    const sourceRow = sky.flipY ? source.height - 1 - row : row;
    for (let x = 0; x < width; x++) {
      const index =
        (sourceRow * source.width + Math.floor((x * source.width) / width)) * 4;
      data.set(source.data.subarray(index, index + 4), (y * width + x) * 4);
    }
  }
  const texture = new THREE.DataTexture(
    data,
    width,
    height,
    THREE.RGBAFormat,
    THREE.FloatType,
  );
  try {
    return await encodeBlob(new Blob([await new EXRExporter().parse(texture)]));
  } finally {
    texture.dispose();
  }
}

export async function exportOfflineScene(
  terrain: THREE.Group,
  camera: THREE.PerspectiveCamera | THREE.OrthographicCamera,
  sun: THREE.DirectionalLight,
  studio: { source: StudioLight; light: THREE.Light }[],
  settings: RenderSettings,
  width: number,
  height: number,
  samples: number,
  planes: THREE.Plane[],
  blockNames: ReadonlyMap<number, string>,
  sky: THREE.DataTexture | undefined,
  bounds: THREE.Box3,
  signal?: AbortSignal,
): Promise<OfflineScene> {
  signal?.throwIfAborted();
  const { GLTFExporter } =
    await import("three/addons/exporters/GLTFExporter.js");
  const root = new THREE.Scene();
  const materials = new Map<string, THREE.MeshStandardMaterial>();
  const geoms: THREE.BufferGeometry[] = [];
  const receivers = new Set(
    studio.flatMap(({ source }) => receiverBlocks(source.receiver)),
  );
  const overrides = new Map(
    settings.blockMaterials?.map((override) => [override.block, override]),
  );
  const blockMaterial = (
    base: THREE.MeshStandardMaterial,
    name: string,
    emission: number,
  ) => {
    const override = overrides.get(name);
    const key = `${base.uuid}|${override?.block ?? ""}|${emission}`;
    let material = materials.get(key);
    if (!material) {
      material = new THREE.MeshStandardMaterial({
        name: override?.block || base.name || "Minecraft terrain",
        color: settings.clay
          ? settings.clayColor
          : override?.tint
            ? override.color
            : base.color,
        map:
          !settings.clay || base.alphaTest > 0 || base.transparent
            ? base.map
            : null,
        side: base.side,
        transparent: base.transparent,
        opacity: base.opacity,
        alphaTest: base.alphaTest,
        roughness: settings.clay
          ? 0.82
          : (override?.roughness ?? base.roughness ?? 1),
        metalness: settings.clay
          ? 0
          : (override?.metalness ?? base.metalness ?? 0),
      });
      const intensity = settings.clay
        ? 0
        : (override?.emission ?? 0) + emission;
      if (intensity > 0) {
        material.emissive.set(0xffffff);
        material.emissiveMap = base.map;
        material.emissiveIntensity = intensity;
      }
      if (settings.clay) material.userData.clay = settings.clayColor;
      materials.set(key, material);
    }
    return material;
  };
  terrain.updateMatrixWorld(true);
  try {
    for (const group of terrain.children) {
      signal?.throwIfAborted();
      for (const mesh of group.children) {
        if (
          !(mesh instanceof THREE.Mesh) ||
          !mesh.visible ||
          Array.isArray(mesh.material)
        )
          continue;
        const source = mesh.geometry as THREE.BufferGeometry;
        const index = source.getIndex();
        if (!index) continue;
        const base = mesh.material as THREE.MeshStandardMaterial;
        const block = source.getAttribute("studioBlock");
        const glow = source.getAttribute("blockEmission");
        const groups = new Map<
          string,
          { ids: number[]; material: THREE.Material; receiver: string }
        >();
        for (let i = 0; i < index.count; i += 3) {
          const first = index.getX(i);
          const name = blockNames.get(block?.getX(first)) ?? "";
          const receiver = receivers.has(name) ? name : "";
          const emission = settings.clay
            ? 0
            : (Math.round((glow?.getX(first) ?? 0) * 15) / 15) *
              settings.emissionPower;
          const material = blockMaterial(base, name, emission);
          const key = `${material.uuid}|${receiver}`;
          let batch = groups.get(key);
          if (!batch) {
            batch = { ids: [], material, receiver };
            groups.set(key, batch);
          }
          batch.ids.push(first, index.getX(i + 1), index.getX(i + 2));
        }
        for (const batch of groups.values()) {
          const geometry = new THREE.BufferGeometry();
          geoms.push(geometry);
          const positions = source.getAttribute("position");
          const normals = source.getAttribute("normal");
          const uv = source.getAttribute("uv");
          if (groups.size === 1) {
            geometry
              .setAttribute("position", positions)
              .setAttribute("normal", normals)
              .setAttribute("uv", uv);
            geometry.setIndex(index);
          } else {
            const remapped = new Map<number, number>();
            const p: number[] = [],
              n: number[] = [],
              u: number[] = [];
            const faces = batch.ids.map((vertex) => {
              let local = remapped.get(vertex);
              if (local !== undefined) return local;
              local = remapped.size;
              remapped.set(vertex, local);
              p.push(
                positions.getX(vertex),
                positions.getY(vertex),
                positions.getZ(vertex),
              );
              n.push(
                normals.getX(vertex),
                normals.getY(vertex),
                normals.getZ(vertex),
              );
              u.push(uv.getX(vertex), uv.getY(vertex));
              return local;
            });
            geometry
              .setAttribute("position", new THREE.Float32BufferAttribute(p, 3))
              .setAttribute("normal", new THREE.Float32BufferAttribute(n, 3))
              .setAttribute("uv", new THREE.Float32BufferAttribute(u, 2));
            geometry.setIndex(faces);
          }
          const object = new THREE.Mesh(geometry, batch.material);
          object.name = `${batch.receiver || "Terrain"} ${group.name || root.children.length + 1}`;
          object.userData.receiver = batch.receiver;
          object.position.setFromMatrixPosition(mesh.matrixWorld);
          root.add(object);
        }
      }
      await new Promise<void>((resolve) => setTimeout(resolve, 0));
    }
    camera.updateMatrixWorld(true);
    const frame = {
      width,
      height,
      samples,
      position: camera.position.toArray(),
      rotation: camera.quaternion.toArray(),
      near: camera.near,
      far: camera.far,
      fov:
        camera instanceof THREE.PerspectiveCamera
          ? camera.getEffectiveFOV()
          : undefined,
      orthoScale:
        camera instanceof THREE.OrthographicCamera
          ? (camera.right - camera.left) / camera.zoom
          : undefined,
      verticalShift:
        camera instanceof THREE.OrthographicCamera
          ? (camera.top + camera.bottom) /
            (2 *
              Math.max(camera.right - camera.left, camera.top - camera.bottom))
          : 0,
      exposure: settings.exposure,
      dof:
        settings.dofEnabled && camera instanceof THREE.PerspectiveCamera
          ? { distance: settings.dofDistance, aperture: settings.dofAperture }
          : undefined,
      transparent: settings.transparent,
      background: settings.background,
      sky: settings.sky,
      ambient: Math.max(settings.ambient, settings.environment * 0.35),
      environment: settings.environment,
      skyRotation: settings.skyRotation,
      skyElevation: settings.skyElevation,
      skyBrightness: settings.skyBrightness,
      skyImage: await encodeEnvironment(sky),
      fog:
        settings.atmosphere.fog && settings.atmosphere.fogDensity > 0
          ? {
              min: bounds.min.toArray(),
              max: bounds.max.toArray(),
              color: settings.atmosphere.fogColor,
              density: settings.atmosphere.fogDensity,
              anisotropy: settings.atmosphere.fogAnisotropy,
            }
          : undefined,
      sun: {
        position: sun.position.toArray(),
        target: sun.target.position.toArray(),
        color: `#${sun.color.getHexString()}`,
        power: sun.intensity,
        softness: settings.softness,
      },
      studio: studio.map(({ source, light }) => ({
        name: source.name,
        type: source.type,
        color: source.color,
        power: source.power,
        shadows: source.shadows,
        angle: source.angle,
        penumbra: source.penumbra,
        receiver: source.receiver,
        receivers: receiverBlocks(source.receiver),
        receiverMode: source.receiverMode,
        gobo: source.gobo,
        range:
          light instanceof THREE.PointLight || light instanceof THREE.SpotLight
            ? light.distance
            : 0,
        position: light.position.toArray(),
        target:
          light instanceof THREE.SpotLight
            ? light.target.position.toArray()
            : new THREE.Vector3(0, 0, -1)
                .applyQuaternion(light.quaternion)
                .add(light.position)
                .toArray(),
        width: light instanceof THREE.RectAreaLight ? light.width : undefined,
        height: light instanceof THREE.RectAreaLight ? light.height : undefined,
      })),
      planes: planes.map((plane) => ({
        normal: plane.normal.toArray(),
        offset: plane.constant,
      })),
    };
    signal?.throwIfAborted();
    const model = (await new GLTFExporter().parseAsync(root, {
      binary: true,
      onlyVisible: true,
    })) as ArrayBuffer;
    signal?.throwIfAborted();
    const data = await encodeBlob(new Blob([model]));
    return { model: data, config: JSON.stringify(frame) };
  } finally {
    for (const geometry of geoms) geometry.dispose();
    for (const material of materials.values()) material.dispose();
  }
}
