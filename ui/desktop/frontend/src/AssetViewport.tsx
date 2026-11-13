import { useDocumentScope } from "./document-scope";
import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import * as T from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { GLTFExporter } from "three/addons/exporters/GLTFExporter.js";
import { runExportTask } from "./export-tasks";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import type { PackObject, PackObjectPreview } from "./bridge";
import { configureMaterial, applyObjectRules } from "./asset-materials";
import { followBones, captureBoneBindings } from "./asset-rig";
import { motionPlayer, particlePlayer } from "./asset-motion";
import { molang, molangScripts, previewContext } from "./asset-molang";
import { createHumanRig } from "./human-skeleton";
import { createRigControls, type RigEditorOptions } from "./rig-controls";
import { createWireframeOverlay } from "./wireframe-overlay";
import { transformBlockModel } from "./block-preview";
import { createPreviewLoop } from "./preview-loop";
import { compactPreviewModel } from "./preview-model";
import type { WireframeSettings, WireframeStats } from "./wireframe-settings";
import {
  bakeModel,
  bedrockModel,
  bedrockVector,
  classicSkin,
  poseSkin,
  poseAnimation,
  releaseModel,
  type SkinPose,
} from "./asset-geometry";

export type AssetViewportHandle = {
  capture: () => string;
  model: () => Promise<string>;
  reset: () => void;
  rigPosition: (id: string) => [number, number, number] | undefined;
  focusRig: (id: string) => void;
  view: (direction: "front" | "back" | "left" | "right" | "top") => void;
};
export type MaterialMaps = {
  color?: string;
  normal?: string;
  mer?: string;
  height?: string;
};
export type SkinTextureAnimation = {
  ImageWidth: number;
  ImageHeight: number;
  ImageData: string;
  FrameCount: number;
  ExpressionType: number;
  AnimationType: number;
};
type Props = {
  cameraState?: { position: number[]; target: number[] };
  onCameraChange?: (view: { position: number[]; target: number[] }) => void;
  rigEditor?: RigEditorOptions;
  onThumbnail?: (data: string) => void;
  effects?: boolean;
  skin?: boolean;
  slim?: boolean;
  legacy?: boolean;
  texture?: string;
  cape?: string;
  face?: {
    geometry: unknown;
    texture: string;
    animation?: SkinTextureAnimation;
  };
  animatedLayers?: { geometry: unknown; animation: SkinTextureAnimation }[];
  modelOnly?: boolean;
  geometry?: unknown;
  attachments?: PackObjectPreview[];
  materialTextures?: Record<string, string>;
  materialMode?: string;
  object?: PackObject;
  initialPose?: { name: string; animation: unknown }[];
  motion?: PackObjectPreview["motion"];
  textureAnimation?: PackObjectPreview["textureAnimation"];
  maps?: MaterialMaps;
  animation?: unknown;
  animationName?: string;
  shape?: "cube" | "sphere";
  pose: SkinPose;
  walking: boolean;
  outer: boolean;
  showCape: boolean;
  rotate: boolean;
  wireframe: boolean;
  wireframeOptions?: WireframeSettings;
  onWireframeStats?: (stats: WireframeStats) => void;
  light: number;
  onError: (message: string) => void;
};
export type SkinRigScene = Pick<
  Props,
  | "texture"
  | "geometry"
  | "face"
  | "animatedLayers"
  | "cape"
  | "slim"
  | "legacy"
  | "outer"
  | "showCape"
  | "light"
>;

function atlasModel(root: T.Group): T.Group {
  const baked = bakeModel(root),
    maps: T.Texture[] = [];
  baked.traverse((o) => {
    if (o instanceof T.Mesh)
      for (const m of Array.isArray(o.material) ? o.material : [o.material]) {
        const map = (m as T.MeshStandardMaterial).map;
        if (map && !maps.includes(map)) maps.push(map);
      }
  });
  if (maps.length <= 1) return baked;
  const width = maps.reduce(
      (n, t) => n + (t.image as HTMLImageElement).width,
      1,
    ),
    height = Math.max(
      1,
      ...maps.map((t) => (t.image as HTMLImageElement).height),
    );
  if (width > 8192 || height > 8192)
    throw new Error("Combined model textures exceed 8192 pixels.");
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d")!,
    offsets = new Map<T.Texture, number>();
  let x = 0;
  for (const map of maps) {
    offsets.set(map, x);
    ctx.drawImage(map.image as HTMLImageElement, x, 0);
    x += (map.image as HTMLImageElement).width;
  }
  ctx.fillStyle = "#b5c7dd";
  ctx.fillRect(width - 1, 0, 1, height);
  const atlas = new T.CanvasTexture(canvas);
  atlas.colorSpace = T.SRGBColorSpace;
  atlas.magFilter = T.NearestFilter;
  atlas.userData.exportOnly = true;
  baked.traverse((o) => {
    if (!(o instanceof T.Mesh)) return;
    const materials = (
        Array.isArray(o.material) ? o.material : [o.material]
      ) as T.MeshStandardMaterial[],
      uv = o.geometry.attributes.uv,
      index = o.geometry.index,
      seen = new Set<number>();
    const groups = o.geometry.groups.length
      ? o.geometry.groups
      : [{ start: 0, count: index?.count ?? uv.count, materialIndex: 0 }];
    for (const group of groups) {
      const material = materials[group.materialIndex ?? 0] ?? materials[0],
        map = material.map;
      for (let i = group.start; i < group.start + group.count; i++) {
        const vertex = index ? index.getX(i) : i;
        if (seen.has(vertex)) continue;
        seen.add(vertex);
        if (map) {
          const image = map.image as HTMLImageElement;
          uv.setXY(
            vertex,
            ((offsets.get(map) ?? 0) + uv.getX(vertex) * image.width) / width,
            1 - ((1 - uv.getY(vertex)) * image.height) / height,
          );
        } else uv.setXY(vertex, (width - 0.5) / width, 0.5);
      }
    }
    for (const material of materials) material.dispose();
    o.geometry.clearGroups();
    o.material = new T.MeshStandardMaterial({
      map: atlas,
      alphaTest: 0.1,
      side: T.DoubleSide,
      roughness: 1,
    });
  });
  return baked;
}

export default forwardRef<AssetViewportHandle, Props>(
  function AssetViewport(p, ref) {
    const scope = useDocumentScope();
    const awake = scope.active || !!scope.busy;
    const cachedCamera = useRef<
      | {
          geometry: unknown;
          texture: Props["texture"];
          view: Props["cameraState"];
        }
      | undefined
    >(undefined);
    const host = useRef<HTMLDivElement>(null),
      state = useRef<{
        renderer: T.WebGLRenderer;
        scene: T.Scene;
        camera: T.PerspectiveCamera;
        controls: OrbitControls;
        root: T.Group;
        grid: T.GridHelper;
        fit: () => void;
        rig?: ReturnType<typeof createHumanRig>;
      } | null>(null),
      latest = useRef(p);
    latest.current = p;
    const invalidate = useRef<(() => void) | undefined>(undefined);
    useEffect(() => {
      invalidate.current?.();
    });
    useImperativeHandle(
      ref,
      () => ({
        capture() {
          const s = state.current;
          if (!s) throw new Error("Preview is still loading.");
          s.grid.visible = false;
          s.renderer.render(s.scene, s.camera);
          const data = s.renderer.domElement.toDataURL("image/png");
          s.grid.visible = true;
          invalidate.current?.();
          return data;
        },
        async model() {
          return runExportTask("Prepare 3D model", async (task) => {
            task.update("Baking model and textures…");
            await new Promise<void>((resolve) =>
              requestAnimationFrame(() => resolve()),
            );
            const s = state.current;
            if (!s) throw new Error("Preview is still loading.");
            const root = atlasModel(s.root);
            try {
              const result = await new GLTFExporter().parseAsync(root, {
                binary: true,
                onlyVisible: true,
              });
              if (!(result instanceof ArrayBuffer))
                throw new Error("Model export did not produce a GLB.");
              const bytes = new Uint8Array(result);
              let text = "";
              for (let i = 0; i < bytes.length; i += 8192)
                text += String.fromCharCode(...bytes.subarray(i, i + 8192));
              return btoa(text);
            } finally {
              root.traverse((o) => {
                if (o instanceof T.Mesh)
                  for (const m of Array.isArray(o.material)
                    ? o.material
                    : [o.material]) {
                    const texture = (m as T.MeshStandardMaterial).map;
                    if (texture?.userData.exportOnly) texture.dispose();
                  }
              });
              releaseModel(root);
            }
          });
        },
        reset() {
          const s = state.current;
          s?.fit();
          if (s)
            latest.current.onCameraChange?.({
              position: s.camera.position.toArray(),
              target: s.controls.target.toArray(),
            });
        },
        rigPosition(id) {
          return state.current?.rig?.position(id).toArray() as
            [number, number, number] | undefined;
        },
        focusRig(id) {
          const s = state.current;
          if (!s?.rig) return;
          const point = s.rig.worldPosition(id),
            offset = s.camera.position
              .clone()
              .sub(s.controls.target)
              .multiplyScalar(0.6);
          s.controls.target.copy(point);
          s.camera.position.copy(point).add(offset);
          s.controls.update();
          latest.current.onCameraChange?.({
            position: s.camera.position.toArray(),
            target: s.controls.target.toArray(),
          });
        },
        view(direction) {
          const s = state.current;
          if (!s) return;
          const vectors = {
            front: [0, 0.1, -1],
            back: [0, 0.1, 1],
            left: [1, 0.1, 0],
            right: [-1, 0.1, 0],
            top: [0, 1, 0.001],
          };
          const distance = s.camera.position.distanceTo(s.controls.target);
          s.camera.position
            .copy(s.controls.target)
            .add(
              new T.Vector3(...(vectors[direction] as [number, number, number]))
                .normalize()
                .multiplyScalar(distance),
            );
          s.camera.up.set(0, 1, 0);
          s.controls.update();
        },
      }),
      [],
    );
    useEffect(() => {
      const element = host.current;
      if (!element || !awake) return;
      let dead = false;
      const textures: T.Texture[] = [];
      let renderer: T.WebGLRenderer;
      try {
        renderer = new T.WebGLRenderer({
          antialias: true,
          alpha: true,
          preserveDrawingBuffer: false,
        });
      } catch (e) {
        latest.current.onError(String(e));
        return;
      }
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.outputColorSpace = T.SRGBColorSpace;
      renderer.toneMapping = T.ACESFilmicToneMapping;
      element.appendChild(renderer.domElement);
      const scene = new T.Scene(),
        camera = new T.PerspectiveCamera(35, 1, 0.01, 10000),
        controls = new OrbitControls(camera, renderer.domElement);
      controls.enableDamping = true;
      controls.addEventListener("end", () =>
        latest.current.onCameraChange?.({
          position: camera.position.toArray(),
          target: controls.target.toArray(),
        }),
      );
      const root = new T.Group();
      const wireframes = createWireframeOverlay(root);
      const updateEffects: ((time: number, enabled: boolean) => void)[] = [];
      const motionPlayers: ((time: number, enabled: boolean) => void)[] = [];
      const surfaceMeshes: T.Mesh[] = [],
        capes: T.Object3D[] = [];
      let hasBoneFollowers = false;
      let animatedEffects = false;
      let humanRig: ReturnType<typeof createHumanRig> | undefined;
      let rigControls: ReturnType<typeof createRigControls> | undefined;
      scene.add(root);
      const ambient = new T.HemisphereLight(0xffffff, 0x505c78, 2),
        light = new T.DirectionalLight(0xffffff, 3);
      light.position.set(4, 6, 8);
      scene.add(ambient, light);
      const grid = new T.GridHelper(6, 12, 0x696969, 0x454545);
      scene.add(grid);
      const environment = new RoomEnvironment();
      const pmrem = new T.PMREMGenerator(renderer);
      const env = pmrem.fromScene(environment, 0.04);
      scene.environment = env.texture;
      environment.dispose();
      pmrem.dispose();
      function fit() {
        const box = new T.Box3().setFromObject(root);
        const center = box.getCenter(new T.Vector3()),
          size = box.getSize(new T.Vector3());
        const radius = Math.max(size.x, size.y, size.z, 0.1);
        controls.target.copy(center);
        camera.position
          .copy(center)
          .add(new T.Vector3(radius * 1.1, radius * 0.5, -radius * 2));
        camera.near = Math.max(0.001, radius / 1000);
        camera.far = radius * 100;
        camera.updateProjectionMatrix();
        grid.position.y = box.min.y - 0.005;
        grid.scale.setScalar(radius / 2);
        controls.update();
      }
      const resize = new ResizeObserver(() => {
        const w = element.clientWidth,
          h = element.clientHeight;
        if (!w || !h) return;
        renderer.setSize(w, h);
        camera.aspect = w / h;
        camera.updateProjectionMatrix();
        invalidate.current?.();
      });
      resize.observe(element);
      const textureLoads = new Map<string, Promise<T.Texture>>();
      function load(url: string, color = true): Promise<T.Texture> {
        const key = `${color ? "color" : "data"}:${url}`;
        const cached = textureLoads.get(key);
        if (cached) return cached;
        const pending = new T.TextureLoader().loadAsync(url).then((texture) => {
          if (dead) {
            texture.dispose();
            throw new Error("Preview was closed.");
          }
          textures.push(texture);
          texture.colorSpace = color ? T.SRGBColorSpace : T.NoColorSpace;
          texture.magFilter = T.NearestFilter;
          texture.minFilter = T.NearestFilter;
          texture.generateMipmaps = false;
          return texture;
        });
        textureLoads.set(key, pending);
        void pending.catch(() => {
          if (textureLoads.get(key) === pending) textureLoads.delete(key);
        });
        return pending;
      }
      const particleDefinitions = new Set<string>();
      function animatedTexture(animation: SkinTextureAnimation) {
        animatedEffects = true;
        const width = animation.ImageWidth,
          height = animation.ImageHeight,
          count = Math.floor(animation.FrameCount);
        if (
          !Number.isInteger(width) ||
          !Number.isInteger(height) ||
          !Number.isInteger(count) ||
          width < 1 ||
          width > 2048 ||
          height < 1 ||
          height > 8192 ||
          count < 1 ||
          count > 1024 ||
          typeof animation.ImageData !== "string" ||
          animation.ImageData.length > 48 * 1024 * 1024
        )
          throw new Error("Animated skin image exceeds preview limits");
        const raw = atob(animation.ImageData);
        if (raw.length !== width * height * 4)
          throw new Error("Animated skin image has invalid dimensions");
        const map = new T.DataTexture(
          Uint8Array.from(raw, (c) => c.charCodeAt(0)),
          width,
          height,
          T.RGBAFormat,
        );
        map.flipY = true;
        map.colorSpace = T.SRGBColorSpace;
        map.magFilter = map.minFilter = T.NearestFilter;
        map.wrapT = T.RepeatWrapping;
        map.needsUpdate = true;
        textures.push(map);
        updateEffects.push((time, enabled) => {
          const blink = time % 4.3 > 4.15;
          const frame = !enabled
            ? 0
            : animation.ExpressionType === 1
              ? blink
                ? Math.min(1, count - 1)
                : 0
              : Math.floor(time * 7) % count;
          map.offset.y = -frame / count;
        });
        return map;
      }
      async function registerRuntime(
        model: T.Group,
        source: Pick<PackObjectPreview, "motion" | "textureAnimation">,
        definition: string,
      ) {
        if (source.motion) {
          animatedEffects ||=
            (source.motion.visibility?.length ?? 0) > 0 ||
            Object.keys(source.motion.animations ?? {}).length > 0 ||
            Object.keys(source.motion.controllers ?? {}).length > 0 ||
            Object.keys(source.motion.particles ?? {}).length > 0;
          const play = motionPlayer(model, source.motion);
          const runtime = particleDefinitions.has(definition)
            ? { ...source.motion, particles: {} }
            : source.motion;
          particleDefinitions.add(definition);
          const particles = await particlePlayer(model, runtime, load);
          model.userData.playMotion = (time: number, enabled: boolean) => {
            model.userData.motionResult = play(time, enabled);
          };
          updateEffects.push((time, enabled) => {
            const result = model.userData.motionResult;
            if (result)
              particles(time, result.context, result.particles, enabled);
          });
        }
        if (source.textureAnimation) {
          animatedEffects = true;
          const sequence = source.textureAnimation;
          const maps = await Promise.all(
            sequence.frames.map((url) => load(url)),
          );
          const framesByMaterial = new Map<
            T.Material,
            T.MeshStandardMaterial[]
          >();
          const assignments: {
            mesh: T.Mesh;
            frames: T.Material[][];
            array: boolean;
          }[] = [];
          model.traverse((node) => {
            if (!(node instanceof T.Mesh)) return;
            const sources = Array.isArray(node.material)
              ? node.material
              : [node.material];
            const frames = sources.map((original) => {
              let frames = framesByMaterial.get(original);
              if (!frames) {
                frames = maps.map((map) => {
                  const material = original.clone() as T.MeshStandardMaterial;
                  configureMaterial(
                    material,
                    map,
                    original.userData.bedrockMaterial ?? "",
                    textures,
                  );
                  if (node.userData.authoredCulling)
                    material.side = original.side;
                  return material;
                });
                framesByMaterial.set(original, frames);
              }
              return frames;
            });
            assignments.push({
              mesh: node,
              frames,
              array: Array.isArray(node.material),
            });
          });
          let previous = -1;
          updateEffects.push((time, enabled) => {
            const context = previewContext(enabled ? time : 0);
            molangScripts(sequence.scripts, context);
            const frame =
              ((Math.floor(molang(sequence.expression, context)) %
                maps.length) +
                maps.length) %
              maps.length;
            if (frame === previous) return;
            previous = frame;
            for (const { mesh, frames, array } of assignments) {
              const materials = frames.map((set) => set[frame]);
              mesh.material = array ? materials : materials[0];
            }
          });
          model.userData.frameMaterials = [
            [...framesByMaterial.values()].flat(),
            [...framesByMaterial.keys()],
          ].flat();
        }
      }
      async function start() {
        try {
          const map =
            p.maps?.color || p.texture
              ? await load(p.maps?.color || p.texture!)
              : null;
          if (dead) return;
          const material = new T.MeshStandardMaterial({
            map,
            color: map ? 0xffffff : 0xb5c7dd,
            roughness: 0.8,
            alphaTest: 0.1,
            side: T.DoubleSide,
          });
          configureMaterial(material, map, p.materialMode ?? "", textures);
          if (p.maps?.normal)
            material.normalMap = await load(p.maps.normal, false);
          if (p.maps?.height) {
            material.bumpMap = await load(p.maps.height, false);
            material.bumpScale = 0.06;
          }
          if (p.maps?.mer) {
            const mer = await load(p.maps.mer, false);
            const image = mer.image as HTMLImageElement;
            const canvas = document.createElement("canvas");
            canvas.width = image.width;
            canvas.height = image.height;
            const ctx = canvas.getContext("2d")!;
            ctx.drawImage(image, 0, 0);
            const pixels = ctx.getImageData(0, 0, image.width, image.height);
            for (let i = 0; i < pixels.data.length; i += 4) {
              const metal = pixels.data[i],
                emissive = pixels.data[i + 1],
                rough = pixels.data[i + 2];
              pixels.data[i] = emissive;
              pixels.data[i + 1] = rough;
              pixels.data[i + 2] = metal;
              pixels.data[i + 3] = 255;
            }
            ctx.putImageData(pixels, 0, 0);
            const packed = new T.CanvasTexture(canvas);
            packed.colorSpace = T.NoColorSpace;
            textures.push(packed);
            material.roughnessMap = packed;
            material.metalnessMap = packed;
            material.metalness = 1;
            material.roughness = 1;
            const emission = document.createElement("canvas");
            emission.width = image.width;
            emission.height = image.height;
            const ec = emission.getContext("2d")!;
            const ep = ctx.getImageData(0, 0, image.width, image.height);
            for (let i = 0; i < ep.data.length; i += 4)
              ep.data[i + 1] = ep.data[i + 2] = ep.data[i];
            ec.putImageData(ep, 0, 0);
            material.emissiveMap = new T.CanvasTexture(emission);
            textures.push(material.emissiveMap);
            material.emissive.set(0xffffff);
          }
          if (dead) {
            material.dispose();
            return;
          }
          let model: T.Group;
          if (p.modelOnly) {
            model = new T.Group();
          } else if (p.geometry) {
            const table: Record<string, T.Material> = { "*": material };
            await Promise.all(
              [...new Set(Object.values(p.materialTextures ?? {}))].map((url) =>
                load(url),
              ),
            );
            if (dead) {
              material.dispose();
              return;
            }
            for (const [alias, url] of Object.entries(
              p.materialTextures ?? {},
            )) {
              const m = material.clone();
              configureMaterial(
                m,
                url === p.texture ? map : await load(url),
                p.object?.materialModes?.[alias] ??
                  p.object?.materialModes?.["*"] ??
                  p.materialMode ??
                  "",
                textures,
              );
              m.color.set(0xffffff);
              table[alias] = m;
            }
            model = bedrockModel(
              p.geometry,
              Object.keys(table).length > 1 ? table : material,
              p.object?.boneMaterials,
            );
          } else if (p.skin) {
            model = classicSkin(material, !!p.slim, !!p.legacy);
          } else {
            model = new T.Group();
            model.add(
              new T.Mesh(
                p.shape === "sphere"
                  ? new T.SphereGeometry(8, 48, 32)
                  : new T.BoxGeometry(16, 16, 16),
                material,
              ),
            );
          }
          for (const initial of p.initialPose ?? [])
            poseAnimation(model, initial.animation, initial.name, 0);
          applyObjectRules(model, p.object, textures);
          if (!p.skin) compactPreviewModel(model);
          if (p.object?.kind === "block")
            model = transformBlockModel(model, p.object.transformation);
          if (p.face && !p.modelOnly) {
            const image = p.face.animation
              ? animatedTexture(p.face.animation)
              : await load(p.face.texture);
            const face = bedrockModel(
              p.face.geometry,
              new T.MeshStandardMaterial({
                map: image,
                alphaTest: 0.01,
                side: T.DoubleSide,
                roughness: 0.8,
              }),
            );
            face.name = "faceGroup";
            model.add(face);
          }
          for (const layer of p.animatedLayers ?? []) {
            const group = bedrockModel(
              layer.geometry,
              new T.MeshStandardMaterial({
                map: animatedTexture(layer.animation),
                alphaTest: 0.01,
                side: T.DoubleSide,
              }),
            );
            model.add(group);
          }
          await registerRuntime(model, p, p.object?.definition ?? "body");
          if (p.skin && !p.modelOnly) humanRig = createHumanRig(model);
          const anchors = new Map<string, T.Object3D>();
          model.traverse((node) => {
            if (
              node instanceof T.Group &&
              node.userData.baseRotation &&
              !anchors.has(node.name.toLowerCase())
            )
              anchors.set(node.name.toLowerCase(), node);
          });
          const cosmeticControls = new Map<string, T.Group>();
          const targets = new Map(anchors);
          for (const part of humanRig?.parts ?? []) {
            const target = humanRig?.object(part.id);
            if (target) {
              targets.set(part.id, target);
              targets.set(part.id.toLowerCase(), target);
            }
          }
          root.userData.cosmeticTargets = targets;
          for (const attachment of p.attachments ?? []) {
            const map = await load(attachment.image);
            const material = new T.MeshStandardMaterial({
              map,
              alphaTest: 0.1,
              side: T.DoubleSide,
              roughness: 0.8,
            });
            configureMaterial(
              material,
              map,
              attachment.object.material,
              textures,
            );
            const cosmetic = bedrockModel(
              attachment.geometry,
              material,
              attachment.object?.boneMaterials,
            );
            applyObjectRules(cosmetic, attachment.object, textures);
            cosmetic.name = "cosmetic";
            cosmetic.traverse((node) => {
              node.userData.cosmetic = true;
              node.userData.cosmeticKey = attachment.object.definition;
              node.userData.followSpace = model;
              const binding = node.userData.binding as string | undefined;
              const socket = binding?.includes("item_slot_to_bone_name")
                ? (
                    {
                      head: "head",
                      chest: "body",
                      body: "body",
                      legs: "waist",
                      feet: "rightleg",
                    } as Record<string, string>
                  )[attachment.capturedSlot ?? ""]
                : binding?.match(/^['"]([\w]+)['"]$/)?.[1].toLowerCase();
              if (socket && anchors.has(socket)) {
                node.userData.followBone = anchors.get(socket);
                node.userData.socketRoot = cosmetic;
                return;
              }
              if (
                node instanceof T.Group &&
                anchors.has(node.name.toLowerCase()) &&
                [
                  "root",
                  "waist",
                  "body",
                  "head",
                  "leftarm",
                  "rightarm",
                  "leftleg",
                  "rightleg",
                ].includes(node.name.toLowerCase())
              )
                node.userData.followBone = anchors.get(node.name.toLowerCase());
            });
            let control = cosmeticControls.get(attachment.object.definition);
            if (!control) {
              control = new T.Group();
              control.name = "cosmetic:" + attachment.object.definition;
              const raw = attachment.geometry as {
                "minecraft:geometry"?: {
                  bones?: { name: string; pivot?: [number, number, number] }[];
                }[];
              };
              const socket =
                attachment.capturedSlot === "head" ? "head" : "body";
              const pivot = raw["minecraft:geometry"]?.[0]?.bones?.find(
                (b) => b.name.toLowerCase() === socket,
              )?.pivot ?? [0, 24, 0];
              control.position.copy(bedrockVector(pivot));
              control.userData.basePosition = control.position.clone();
              control.userData.baseRotation = control.rotation.clone();
              control.userData.cosmetic = true;
              control.userData.editableCosmetic = true;
              control.userData.cosmeticKey = attachment.object.definition;
              control.userData.followSpace = model;
              model.add(control);
              cosmeticControls.set(attachment.object.definition, control);
              humanRig?.register(
                control.name,
                attachment.object.name.split(" / ")[0],
                control,
                "cosmetic",
              );
            }
            model.add(cosmetic);
            control.attach(cosmetic);
            humanRig?.registerCosmetic(cosmetic, attachment.object.definition);
            await registerRuntime(
              cosmetic,
              attachment,
              attachment.object.definition,
            );
          }
          if (
            p.skin &&
            p.cape &&
            !p.modelOnly &&
            !p.attachments?.some((a) => a.motion?.hideCape)
          ) {
            const texture = await load(p.cape);
            const capeMaterial = new T.MeshStandardMaterial({
              map: texture,
              alphaTest: 0.1,
              side: T.DoubleSide,
            });
            const cape = bedrockModel(
              {
                "minecraft:geometry": [
                  {
                    description: { texture_width: 64, texture_height: 32 },
                    bones: [
                      {
                        name: "cape",
                        pivot: [0, 24, 2.5],
                        rotation: [12, 0, 0],
                        cubes: [
                          {
                            origin: [-5, 8, 2.5],
                            size: [10, 16, 1],
                            uv: [0, 0],
                          },
                        ],
                      },
                    ],
                  },
                ],
              },
              capeMaterial,
            );
            cape.name = "capeGroup";
            const body = anchors.get("body");
            if (body) body.attach(cape);
            else model.add(cape);
          }
          if (dead) {
            releaseModel(model);
            return;
          }
          root.add(model);
          root.traverse((node) => {
            if (node instanceof T.Mesh && !node.userData.previewOnly)
              surfaceMeshes.push(node);
            if (node.name === "capeGroup") capes.push(node);
            if (typeof node.userData.playMotion === "function")
              motionPlayers.push(node.userData.playMotion);
            if (
              node.userData.followBone ||
              node.userData.socketRoot ||
              node.userData.editableCosmetic
            )
              hasBoneFollowers = true;
          });
          root.scale.setScalar(1 / 16);
          state.current = {
            renderer,
            scene,
            camera,
            controls,
            root,
            grid,
            fit,
            rig: humanRig,
          };
          captureBoneBindings(root);
          poseSkin(root, p.pose, 0, false);
          humanRig?.update(p.pose);
          root.traverse((node) =>
            node.userData.playMotion?.(2, p.effects !== false),
          );
          if (hasBoneFollowers) followBones(root, p.pose);
          fit();
          renderer.render(scene, camera);
          if (humanRig && p.rigEditor)
            rigControls = createRigControls(
              scene,
              camera,
              renderer.domElement,
              controls,
              model,
              humanRig,
              () => ({
                editor: latest.current.rigEditor,
                pose: latest.current.pose,
              }),
            );
          let thumbnailAllowed = !p.skin;
          if (p.skin) {
            const bounds = new T.Box3();
            root.traverse((node) => {
              if (
                node instanceof T.Group &&
                node.name.toLowerCase() === "head" &&
                !node.userData.cosmetic
              ) {
                const box = new T.Box3().setFromObject(node);
                if (!box.isEmpty()) bounds.union(box);
              }
            });
            if (!bounds.isEmpty()) {
              thumbnailAllowed = true;
              const center = bounds.getCenter(new T.Vector3()),
                size = bounds.getSize(new T.Vector3());
              const headCamera = camera.clone();
              const halfAngle = Math.atan(
                Math.tan(T.MathUtils.degToRad(headCamera.fov / 2)) *
                  Math.min(headCamera.aspect, 1),
              );
              const distance =
                (Math.max(size.length() / 2, 0.05) / Math.sin(halfAngle)) *
                1.08;
              headCamera.position
                .copy(center)
                .add(
                  new T.Vector3(0.8, 0.25, -1.9)
                    .normalize()
                    .multiplyScalar(distance),
                );
              headCamera.lookAt(center);
              headCamera.updateProjectionMatrix();
              grid.visible = false;
              renderer.render(scene, headCamera);
              grid.visible = true;
            }
          }
          const thumbnail = document.createElement("canvas");
          thumbnail.width = thumbnail.height = 96;
          const context = thumbnail.getContext("2d");
          if (context && thumbnailAllowed) {
            const source = renderer.domElement;
            const side = Math.min(source.width, source.height);
            context.drawImage(
              source,
              (source.width - side) / 2,
              (source.height - side) / 2,
              side,
              side,
              0,
              0,
              96,
              96,
            );
            latest.current.onThumbnail?.(thumbnail.toDataURL("image/png"));
          }
          renderer.render(scene, camera);
        } catch (e) {
          if (!dead)
            latest.current.onError(e instanceof Error ? e.message : String(e));
        }
      }
      void start().then(() => {
        if (dead) return;
        const cached = cachedCamera.current;
        const v =
          cached &&
          cached.geometry === p.geometry &&
          cached.texture === p.texture
            ? cached.view
            : latest.current.cameraState;
        if (
          v?.position?.length === 3 &&
          v?.target?.length === 3 &&
          [...v.position, ...v.target].every(Number.isFinite)
        ) {
          camera.position.fromArray(v.position);
          controls.target.fromArray(v.target);
          controls.update();
        }
        invalidate.current?.();
      });
      let last = performance.now(),
        time = 0;
      let wireframeStatsKey = "";
      function draw(now: number) {
        const dt = Math.min((now - last) / 1000, 0.1);
        last = now;
        time += dt;
        const options = latest.current;
        light.intensity = options.light;
        controls.autoRotate = options.rotate;
        controls.autoRotateSpeed = 1.5;
        const cameraChanged = controls.update();
        for (const cape of capes) cape.visible = options.showCape;
        poseSkin(root, options.pose, time, !!options.skin && options.walking);
        if (!options.motion)
          for (const initial of options.initialPose ?? [])
            poseAnimation(root, initial.animation, initial.name, 0);
        for (const play of motionPlayers) play(time, options.effects !== false);
        for (const o of surfaceMeshes) {
          o.visible =
            o.userData.controllerVisible !== false &&
            (!o.userData.layer || options.outer);
          for (const m of Array.isArray(o.material) ? o.material : [o.material])
            if (m instanceof T.MeshStandardMaterial) m.wireframe = false;
        }
        if (options.animation && options.animationName)
          poseAnimation(root, options.animation, options.animationName, time);
        humanRig?.update(options.pose);
        if (hasBoneFollowers) followBones(root, options.pose);
        for (const update of updateEffects)
          update(time, options.effects !== false);
        rigControls?.update();
        wireframes.update(
          options.wireframe,
          options.wireframeOptions,
          renderer.domElement.clientWidth,
          renderer.domElement.clientHeight,
        );
        const stats = wireframes.stats();
        const statsKey = JSON.stringify(stats);
        if (statsKey !== wireframeStatsKey) {
          wireframeStatsKey = statsKey;
          latest.current.onWireframeStats?.(stats);
        }
        renderer.render(scene, camera);
        return cameraChanged;
      }
      const loop = createPreviewLoop({
        draw,
        visible: () =>
          !dead &&
          !document.hidden &&
          element.clientWidth > 0 &&
          element.clientHeight > 0 &&
          !element.closest("[hidden]"),
        continuous: () => {
          const options = latest.current;
          return (
            options.rotate ||
            (!!options.skin && options.walking) ||
            (!!options.animation && !!options.animationName) ||
            (animatedEffects && options.effects !== false) ||
            !!options.rigEditor
          );
        },
      });
      const requestDraw = () => {
        loop.invalidate();
      };
      const visibility = () => {
        last = performance.now();
        loop.invalidate();
      };
      invalidate.current = requestDraw;
      controls.addEventListener("change", requestDraw);
      renderer.domElement.addEventListener("pointermove", requestDraw);
      renderer.domElement.addEventListener("webglcontextrestored", requestDraw);
      document.addEventListener("visibilitychange", visibility);
      loop.invalidate();
      return () => {
        cachedCamera.current = {
          geometry: p.geometry,
          texture: p.texture,
          view: {
            position: camera.position.toArray(),
            target: controls.target.toArray(),
          },
        };
        dead = true;
        loop.dispose();
        if (invalidate.current === requestDraw) invalidate.current = undefined;
        controls.removeEventListener("change", requestDraw);
        renderer.domElement.removeEventListener("pointermove", requestDraw);
        renderer.domElement.removeEventListener("webglcontextrestored", requestDraw);
        document.removeEventListener("visibilitychange", visibility);
        resize.disconnect();
        controls.dispose();
        rigControls?.dispose();
        wireframes.dispose();
        state.current = null;
        releaseModel(root);
        root.traverse((node) => {
          node.userData.disposeParticles?.();
          if (node instanceof T.Sprite) {
            node.material.map?.dispose();
            node.material.dispose();
          }
          for (const material of node.userData.frameMaterials ?? [])
            material.dispose();
        });
        textures.forEach((t) => t.dispose());
        grid.geometry.dispose();
        (grid.material as T.Material).dispose();
        env.dispose();
        renderer.dispose();
        renderer.forceContextLoss();
        renderer.domElement.remove();
      };
    }, [
      awake,
      p.texture,
      p.cape,
      p.face,
      p.animatedLayers,
      p.modelOnly,
      p.geometry,
      p.attachments,
      p.motion,
      p.textureAnimation,
      p.materialTextures,
      p.materialMode,
      p.object,
      p.initialPose,
      p.maps,
      p.shape,
      p.slim,
      p.legacy,
      p.skin,
    ]);
    return (
      <div
        ref={host}
        className="asset-viewport"
        role="img"
        aria-label={
          p.skin
            ? "Interactive skin preview"
            : "Interactive model and material preview"
        }
      />
    );
  },
);
