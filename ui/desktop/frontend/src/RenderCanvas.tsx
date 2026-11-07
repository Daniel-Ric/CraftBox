import { useDocumentScope } from "./document-scope";
import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import * as THREE from "three";
import { StudioLightRig } from "./studio-lights";
import type { LightEditorState } from "./StudioLightControls";
import type { CloudEditorState } from "./CloudLayerControls";
import { TransformControls } from "three/addons/controls/TransformControls.js";
import { DepthOfFieldPass, focusDepth } from "./render-depth-of-field";
import { pickFocusSurface } from "./focus-selection";
import { FocusGuides } from "./focus-guides";
import {
  cubeFaces,
  stitchPanorama,
  decodeHDRFace,
  type PixelImage,
} from "./panorama";
import { renderPanoramaPost } from "./panorama-post";
import { buildCloudClearance } from "./cloud-clearance";
import { MapOverlayPass } from "./map-overlay";
import type { MapPresentation } from "./map-presentation";
import type { CinematicViewState, SavedCameraView } from "./studio-document";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { RenderPass } from "three/addons/postprocessing/RenderPass.js";
import { GTAOPass } from "three/addons/postprocessing/GTAOPass.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import { UnrealBloomPass } from "three/addons/postprocessing/UnrealBloomPass.js";
import { sampleSequence, type CameraPose, type Sequence } from "./cinematic";
import { sampleTarget } from "./cinematic-tracking";
import { temporalSamples, type MotionFrameSample } from "./cinematic-motion";
import { TemporalAccumulator } from "./render-motion-blur";
import { api, type ScenePageRequest } from "./bridge";
import { SceneDecoder } from "./scene-decoder";
import { SceneLedger } from "./scene-state";
import { FrameQueue } from "./scene-queue";
import type { DecodedChunk } from "./scene-wire";
import {
  renderDimensions,
  renderQuality,
  type RenderSettings,
} from "./render-settings";
import { TerrainNormalsPass } from "./render-normals";
import { loadRenderSky, type RenderSky } from "./render-skies";
import { SkyBackgroundPass } from "./render-sky-background";
import { AtlasMipmaps, atlasTileAttributes } from "./render-atlas";
import { ColorGradePass } from "./render-color-grade";
import { AtmospherePass } from "./atmosphere-volume";
import { foliageAttributes, installFoliageLighting } from "./foliage-lighting";
import { polarLimits } from "./viewport-views";
import { FreeCameraNavigation, snapshotRoll } from "./free-camera";
import {
  LightingRig,
  emissionAttributes,
  installEmission,
  installBlockLight,
} from "./render-lighting";
import type { ReactNode } from "react";
import type { ModelAnalysis } from "./auto-direction";
import { AnalysisCache, analysePacked, sampleScene, checkAnalysisAbort, analysisYield, type AnalysisProgressCallback, type AnalysisMesh } from "./scene-analysis";
import { renderTemporalSteps } from "./render-temporal-steps";
import { CloudGizmos } from "./cloud-gizmos";
import { atmosphereAnimating } from "./cloud-render-config";
import { WaterSurface } from "./render-water";
import { PreviewPerformance } from "./preview-performance";
import {PreviewAccumulator,previewJitter} from "./preview-accumulator";
import {runExportTask} from "./export-tasks";
import {
  spatialSamples,
  withCameraJitter,
  fittedOrthoSpan,
} from "./render-quality-export";

export type RenderCanvasHandle = {
	 capturePose?: (point:import("./bridge").CapturePoint)=>CameraPose|undefined;
  flightPosition?: () => [number, number, number];
  teleport?: (position: [number, number, number]) => void;
  lockMouse?: () => Promise<boolean>;
  flightReturn?: () => void;
  beginFlight?: (
    flight: NonNullable<RenderCanvasProps["flight"]>,
  ) => Promise<boolean>;
  seekTime?: (time: number) => void;
  studioState?: () => CinematicViewState;
  analyse: (progress?: AnalysisProgressCallback, signal?: AbortSignal) => Promise<ModelAnalysis>;
  preview?: () => string | undefined;
  capture: () => Promise<string>;
  capturePanorama?: (
    width: number,
    progress?: (value: number) => void,
    cancelled?: () => boolean,
  ) => Promise<string>;
  fit: () => void;
  view: (angle: number, tilt: number) => void;
  snapshot: () => SavedCameraView | undefined;
  renderFrame: (
    pose: CameraPose,
    width: number,
    height: number,
    time?: number,
    samples?: MotionFrameSample[],
    signal?: AbortSignal,
  ) => Promise<HTMLCanvasElement>;
  finishExport: () => void;
};
export type RenderCanvasProps = {
  sourceName?: string;
	viewportQuality?:"editing"|"full";
	viewportFocus?:[number,number];
	 capturePoints?: import("./bridge").CapturePoint[];
  pickingResource?: boolean;
  onResourcePick?: (position: [number, number, number], face: number) => void;
  selectionBounds?: import("./bridge").ChunkBounds;
  active?: boolean;
  flight?: {
    enabled: boolean;
    speed: number;
    fov: number;
    sensitivity: number;
    level: boolean;
    smoothing?: number;
    onExit: () => void;
    onSpeed: (scale: number) => void;
    onFov?: (value: number) => void;
  };
  targetEditor?: {
    selected?: string;
    picking: boolean;
    onSelect: (id: string) => void;
    onPosition: (position: [number, number, number]) => void;
  };
  onLensScale?: (scale: number) => void;
  pickingFocus?: boolean;
  focusGuides?: boolean;
  onFocusCancel?: () => void;
  onFocusPick?: (distance: number, point: [number, number, number]) => void;
  lightEditor?: LightEditorState;
  cloudEditor?: CloudEditorState;
  worldId: string;
  settings: RenderSettings;
  skyFile?: File;
  mapView?: MapPresentation;
  placingMarker?: boolean;
  onMapPlace?: (x: number, z: number) => void;
  initialView?: SavedCameraView;
  initialMapFocus?: [number, number, number];
  onViewChange?: (pose: SavedCameraView) => void;
  onViewDirty?: () => void;
  initialCinematic?: CinematicViewState;
  onCinematicChange?: (state: CinematicViewState) => void;
  onProgress: (loaded: number, total: number, triangles: number) => void;
  onReady: () => void;
  onError: (text: string) => void;
  onSkyLoading: (loading: boolean) => void;
  cinematic?: boolean;
  onExporting?: (busy: boolean) => void;
  onHistoryChange?: (history: { undo: boolean; redo: boolean }) => void;
  projectId?: string;
  projectName?: string;
  projectDuration?: number;
  projectFps?: number;
  viewMenu?: ReactNode;
  shot?: {
    pose: CameraPose;
    sequence: Sequence;
    preview: boolean;
    playing?: boolean;
    showPath: boolean;
    time?: number;
    navigation?: "orbit" | "fly";
    navigationSpeed?: number;
    onNavigationExit?: () => void;
  };
};
type Props = RenderCanvasProps;

export class RenderEngine {
  private analysisRevision = 0;
  private analysisCache = new AnalysisCache();
	private previewAA=new PreviewAccumulator();
	private previewFrame=0;
	private fullDetail=false;
	private detailTask?:Promise<void>;
	private renderedAtlasRevision=0;
	private clearanceTask?:Promise<void>;
	private clearanceGeneration=0;
	pipelineFull?:boolean;
	pipelineFinal?:boolean;
  private flightActive = false;
  private flightStart?: {
    position: THREE.Vector3;
    target: THREE.Vector3;
    span: number;
  };
  private initializing = true;
  private appliedMap?: { rotation: number; zoom: number };
  private viewTimer?: ReturnType<typeof setTimeout>;
  readonly mapOverlay = new MapOverlayPass();
  private mapCamera?: {
    camera: THREE.OrthographicCamera | THREE.PerspectiveCamera;
    target: THREE.Vector3;
    span: number;
  };
  lighting = new LightingRig();
  renderer: THREE.WebGLRenderer;
  scene = new THREE.Scene();
  terrain = new THREE.Group();
  camera: THREE.OrthographicCamera | THREE.PerspectiveCamera =
    new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 50000);
  path = new THREE.Group();
  targetGuides = new THREE.Group();
  targetMarkers = new Map<
    string,
    THREE.Mesh<THREE.OctahedronGeometry, THREE.MeshBasicMaterial>
  >();
  temporal?: TemporalAccumulator;
  bloom = new UnrealBloomPass(new THREE.Vector2(1, 1), 0.12, 0.4, 1.2);
  dof = new DepthOfFieldPass();
  focusGuides: FocusGuides;
  private focusHover?: THREE.Vector3;
  private pendingFocusPointer?: [number, number];
  private lastFocusHover = 0;
  controls: OrbitControls;
  freeCamera: FreeCameraNavigation;
  composer: EffectComposer;
  beauty: RenderPass;
  skyBackground = new SkyBackgroundPass();
  ao: GTAOPass;
  normals: TerrainNormalsPass;
  dofDepth: TerrainNormalsPass;
  sky?: RenderSky;
  environmentMap?: THREE.WebGLRenderTarget;
  pmrem: THREE.PMREMGenerator;
  skyTask: Promise<void> = Promise.resolve();
  skyKey = "";
  skyFile?: File;
  skyGeneration = 0;
  output = new OutputPass();
  grade = new ColorGradePass();
  atmosphere = new AtmospherePass();
  atmosphereStart = performance.now();
  light = new THREE.DirectionalLight(0xffefd7, 3);
  ambient = new THREE.HemisphereLight(0xd9edff, 0x88715b, 1.35);
  studioLights = new StudioLightRig();
  lightGuides = new THREE.Group();
  lightTransform: TransformControls;
  cloudGizmos: CloudGizmos;
  lightAnchor = new THREE.Object3D();
  lightMarkers = new Map<string, THREE.Mesh>();
  lightGuideSignature = "";
  selectedLightGuide?: THREE.PointLightHelper | THREE.SpotLightHelper;
  lightSightline = new THREE.Line(
    new THREE.BufferGeometry(),
    new THREE.LineBasicMaterial({
      color: 0x62aeff,
      depthTest: false,
      transparent: true,
      opacity: 0.8,
    }),
  );
  materials: Record<string, THREE.Material>;
  water = new WaterSurface();
  hasWater = false;
  decoder = new SceneDecoder();
  ledger = new SceneLedger();
  bounds = new THREE.Box3();
  groups = new Map<string, THREE.Group>();
  texture?: THREE.DataTexture;
  atlasMips?: AtlasMipmaps;
  pixels?: Uint8Array<ArrayBuffer>;
  origin?: THREE.Vector3;
  disposed = false;
  frame = 0;
  dirty = true;
  interacting = false;
  settle = 0;
  quality = false;
  private previewPerformance = new PreviewPerformance();
  private viewportSize = new THREE.Vector2();
  exporting = false;
  loading = true;
  loadStarted = false;
  span = 10;
  options: RenderSettings;
  resizeObserver: ResizeObserver;
  queue: FrameQueue<DecodedChunk>;
  host: HTMLDivElement;
  callbacks: { current: Props };
  lost = (event: Event) => {
    event.preventDefault();
    this.callbacks.current.onError(
      "The graphics context was lost. Reopen the map.",
    );
  };
  constructor(host: HTMLDivElement, callbacks: { current: Props }) {
    this.host = host;
    this.callbacks = callbacks;
    this.options = callbacks.current.settings;
    if (callbacks.current.cinematic)
      this.camera = new THREE.PerspectiveCamera(45, 1, 0.1, 50000);
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,

      premultipliedAlpha: false,
      preserveDrawingBuffer: true,
      powerPreference: "high-performance",
    });
    this.focusGuides = new FocusGuides(host, this.renderer.domElement);
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = THREE.PCFShadowMap;
    this.renderer.shadowMap.autoUpdate = false;
    this.renderer.domElement.setAttribute(
      "aria-label",
      "Render preview. Drag to orbit, right-drag to pan, and scroll to zoom.",
    );
    this.renderer.domElement.addEventListener("webglcontextlost", this.lost);
    host.append(this.renderer.domElement);
    this.scene.add(
      this.terrain,
      this.ambient,
      this.light,
      this.light.target,
      this.path,
      this.targetGuides,
      this.lighting.group,
      this.studioLights.group,
    );
    this.pmrem = new THREE.PMREMGenerator(this.renderer);
    this.light.castShadow = false;
    this.light.shadow.bias = -0.00005;
    this.light.shadow.normalBias = 0.025;
    this.light.shadow.radius = 2;
    this.materials = {
      water: this.water.material,
      opaque: new THREE.MeshStandardMaterial({ roughness: 1 }),
      mask: new THREE.MeshStandardMaterial({
        roughness: 1,
        alphaTest: 0.5,
        alphaToCoverage: true,
        side: THREE.DoubleSide,
      }),
      blend: new THREE.MeshPhysicalMaterial({
        roughness: 0.2,
        metalness: 0,
        clearcoat: 1,
        clearcoatRoughness: 0.12,
        ior: 1.333,
        transparent: true,
        opacity: 0.72,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
      unlit: new THREE.MeshBasicMaterial({
        alphaTest: 0.05,
        side: THREE.DoubleSide,
      }),
      "unlit-blend": new THREE.MeshBasicMaterial({
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
    };
    const target = new THREE.WebGLRenderTarget(1, 1, {
      type: THREE.HalfFloatType,
    });
    target.samples = Math.min(2, this.renderer.capabilities.maxSamples);
    this.composer = new EffectComposer(this.renderer, target);
    this.composer.addPass(this.skyBackground);
    this.beauty = new RenderPass(this.scene, this.camera);
    this.normals = new TerrainNormalsPass(
      this.scene,
      this.camera,
      this.terrain,
    );
    this.normals.excluded = [this.path, this.lightGuides, this.targetGuides];
    this.composer.addPass(this.normals);
    this.composer.addPass(this.beauty);
    this.dofDepth = new TerrainNormalsPass(
      this.scene,
      this.camera,
      this.terrain,
    );
    this.dofDepth.excluded = [this.path, this.lightGuides, this.targetGuides];
    this.dofDepth.includeTransparent = true;
    this.dofDepth.enabled = false;
    this.ao = new GTAOPass(this.scene, this.camera, 1, 1);
    this.ao.setGBuffer(
      this.normals.target.depthTexture!,
      this.normals.target.texture,
    );
    this.ao.updateGtaoMaterial({
      radius: 3,
      thickness: 1,
      distanceFallOff: 0.9,
      samples: 16,
      screenSpaceRadius: false,
    });
    this.ao.updatePdMaterial({ radius: 4, rings: 2, samples: 8 });
    this.composer.addPass(this.ao);
    this.composer.addPass(this.atmosphere);
    this.composer.addPass(this.dofDepth);
    this.composer.addPass(this.dof);
    this.atmosphere.localLighting.rig = this.lighting;
    this.composer.addPass(this.bloom);
    this.composer.addPass(this.output);
    this.composer.addPass(this.grade);
    this.composer.addPass(this.previewAA);
    this.composer.addPass(this.mapOverlay);
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.lightTransform = new TransformControls(
      this.camera,
      this.renderer.domElement,
    );
    this.lightTransform.setMode("translate");
    this.lightTransform.setSize(0.8);
    this.lightGuides.add(
      this.lightTransform.getHelper(),
      this.lightAnchor,
      this.lightSightline,
    );
    this.scene.add(this.lightGuides);
    this.cloudGizmos = new CloudGizmos(
      this.camera,
      this.renderer.domElement,
      () => {
        this.dirty = true;
      },
      (value) => {
        this.controls.enabled = !value;
        this.interacting = value;
        this.dirty = true;
      },
    );
    this.scene.add(this.cloudGizmos.group);
    this.normals.excluded.push(this.cloudGizmos.group);
    this.dofDepth.excluded.push(this.cloudGizmos.group);
    this.lightTransform.addEventListener("dragging-changed", (event) => {
      this.controls.enabled = !event.value;
      this.interacting = Boolean(event.value);
      this.dirty = true;
    });
    this.lightTransform.addEventListener("objectChange", () => {
      this.dirty = true;
    });
    this.lightTransform.addEventListener("mouseUp", () => {
      const editor = this.callbacks.current.lightEditor;
      if (!editor?.selected || this.bounds.isEmpty()) return;
      const radius = Math.max(
        1,
        this.bounds.getBoundingSphere(new THREE.Sphere()).radius,
      );
      const p = this.lightAnchor.position
        .clone()
        .sub(this.bounds.getCenter(new THREE.Vector3()))
        .divideScalar(radius);
      editor.onPosition(
        editor.selected,
        p.toArray().map((v) => THREE.MathUtils.clamp(v, -100, 100)) as [
          number,
          number,
          number,
        ],
        !!editor.target,
      );
    });
    let pointerStart = [0, 0];
    this.renderer.domElement.addEventListener("pointermove", (e) => {
      if (
        !this.callbacks.current.pickingFocus ||
        this.loading ||
        this.exporting
      )
        return;
      this.pendingFocusPointer = [e.clientX, e.clientY];
    });
    this.renderer.domElement.addEventListener("pointerleave", () => {
      this.focusHover = undefined;
      this.pendingFocusPointer = undefined;
      this.dirty = true;
    });
    this.renderer.domElement.addEventListener("keydown", (e) => {
      if (e.code === "Escape" && this.callbacks.current.pickingFocus) {
        e.preventDefault();
        e.stopPropagation();
        this.callbacks.current.onFocusCancel?.();
      }
    });
    this.renderer.domElement.addEventListener("pointerdown", (e) => {
      pointerStart = [e.clientX, e.clientY];
    });
    this.renderer.domElement.addEventListener("pointerup", (e) => {
      if (
        this.callbacks.current.pickingResource &&
        e.button === 0 &&
        !this.loading &&
        !this.exporting &&
        Math.hypot(e.clientX - pointerStart[0], e.clientY - pointerStart[1]) <=
          5
      ) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        const ray = new THREE.Raycaster();
        ray.setFromCamera(
          new THREE.Vector2(
            ((e.clientX - rect.left) / rect.width) * 2 - 1,
            (-(e.clientY - rect.top) / rect.height) * 2 + 1,
          ),
          this.camera,
        );
        const hit = ray.intersectObject(this.terrain, true)[0];
        if (hit?.face) {
          const normal = hit.face.normal
            .clone()
            .transformDirection(hit.object.matrixWorld);
          const point = hit.point.clone().addScaledVector(normal, -0.01);
          point.x += this.origin?.x ?? 0;
          point.z += this.origin?.z ?? 0;
          const face =
            Math.abs(normal.y) > 0.5
              ? normal.y > 0
                ? 2
                : 3
              : Math.abs(normal.x) > 0.5
                ? normal.x > 0
                  ? 0
                  : 1
                : normal.z > 0
                  ? 4
                  : 5;
          this.callbacks.current.onResourcePick?.(
            point.toArray().map(Math.floor) as [number, number, number],
            face,
          );
        }
        return;
      }
      if (
        e.button === 0 &&
        !this.loading &&
        !this.exporting &&
        Math.hypot(e.clientX - pointerStart[0], e.clientY - pointerStart[1]) <=
          5 &&
        !this.callbacks.current.pickingFocus &&
        this.pickTarget(e)
      )
        return;
      if (
        this.callbacks.current.pickingFocus &&
        e.button === 0 &&
        !this.loading &&
        !this.exporting &&
        Math.hypot(e.clientX - pointerStart[0], e.clientY - pointerStart[1]) <=
          5
      ) {
        const hit = this.focusHit(e);
        if (hit)
          this.callbacks.current.onFocusPick?.(
            focusDepth(hit.point, this.camera),
            [
              hit.point.x + (this.origin?.x ?? 0),
              hit.point.y,
              hit.point.z + (this.origin?.z ?? 0),
            ],
          );
        return;
      }
      if (
        e.button === 0 &&
        !this.loading &&
        !this.exporting &&
        !this.lightTransform.dragging &&
        Math.hypot(e.clientX - pointerStart[0], e.clientY - pointerStart[1]) <=
          5
      ) {
        this.updateCloudGuides(
          this.callbacks.current.shot?.time ??
            (performance.now() - this.atmosphereStart) / 1000,
        );
        if (
          this.cloudGizmos.pick(
            e,
            this.camera,
            this.renderer.domElement,
            this.terrain,
          )
        )
          return;
        if (this.pickLight(e)) return;
      }
      if (
        !this.callbacks.current.mapView?.enabled ||
        !this.callbacks.current.placingMarker ||
        this.loading ||
        Math.hypot(e.clientX - pointerStart[0], e.clientY - pointerStart[1]) > 5
      )
        return;
      const rect = this.renderer.domElement.getBoundingClientRect(),
        ray = new THREE.Raycaster();
      ray.setFromCamera(
        new THREE.Vector2(
          ((e.clientX - rect.left) / rect.width) * 2 - 1,
          (-(e.clientY - rect.top) / rect.height) * 2 + 1,
        ),
        this.camera,
      );
      const hit = ray.ray.intersectPlane(
        new THREE.Plane(new THREE.Vector3(0, 1, 0), 0),
        new THREE.Vector3(),
      );
      if (hit)
        this.callbacks.current.onMapPlace?.(
          hit.x + (this.origin?.x ?? 0),
          hit.z + (this.origin?.z ?? 0),
        );
    });
    this.renderer.domElement.tabIndex = 0;
    this.renderer.domElement.setAttribute("aria-label", "3D camera view");
    this.freeCamera = new FreeCameraNavigation(
      this.camera,
      this.renderer.domElement,
      this.controls.target,
      () => {
        this.dirty = true;
        this.settle = performance.now() + 180;
        this.scheduleViewChange();
      },
      () => {
        if (this.callbacks.current.flight?.enabled)
          this.callbacks.current.flight.onExit();
        else this.callbacks.current.shot?.onNavigationExit?.();
      },
    );
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.12;
    this.controls.maxPolarAngle = Math.PI * 0.495;
    this.controls.addEventListener("change", () => {
      this.dirty = true;
      this.settle = performance.now() + 180;
      this.scheduleViewChange();
    });
    this.controls.addEventListener("start", () => {
      this.interacting = true;
      this.dirty = true;
    });
    this.controls.addEventListener("end", () => {
      this.interacting = false;
      this.dirty = true;
      this.settle = performance.now() + 180;
      clearTimeout(this.viewTimer);
      this.viewTimer = setTimeout(() => {
        if (!this.disposed && !this.loading)
          this.callbacks.current.onViewChange?.({
            ...this.snapshot(),
            zoom: this.camera.zoom,
          });
      }, 200);
    });
    this.queue = new FrameQueue<DecodedChunk>((chunk) => this.commit(chunk));
    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(host);
    this.apply(this.options, callbacks.current.skyFile);
    this.view(45, 35);
    this.resize();
    let previousFrame = performance.now();
    const tick = () => {
      if (this.disposed) return;
      this.frame = requestAnimationFrame(tick);
      const now = performance.now(),
        frameMs = now - previousFrame,
        delta = Math.min(0.05, frameMs / 1000);
      previousFrame = now;
      this.freeCamera.enabled =
        !this.exporting &&
        !this.loading &&
        (this.callbacks.current.shot?.navigation === "fly" ||
          !!this.callbacks.current.flight?.enabled) &&
        !this.callbacks.current.shot?.preview &&
        !this.callbacks.current.pickingFocus;
      if (this.exporting) return;
      try {
        this.queue.flush();

        if ((this.loading&&this.groups.size===0) || this.callbacks.current.active === false) return;
        if (
          this.callbacks.current.pickingFocus &&
          this.pendingFocusPointer &&
          now - this.lastFocusHover > 80
        ) {
          const [clientX, clientY] = this.pendingFocusPointer;
          this.pendingFocusPointer = undefined;
          this.lastFocusHover = now;
          this.focusHover = this.focusHit({ clientX, clientY })?.point.clone();
          this.dirty = true;
        }
        if (this.callbacks.current.pickingFocus) this.controls.enabled = false;
        if (this.freeCamera.enabled) {
          this.controls.enabled = false;
          const radius = this.bounds.isEmpty()
            ? 10
            : this.bounds.getBoundingSphere(new THREE.Sphere()).radius;
          this.freeCamera.levelFlight =
            this.callbacks.current.flight?.level ?? false;
          this.freeCamera.verticalShortcuts =
            !!this.callbacks.current.flight?.enabled;
          this.freeCamera.sensitivity =
            this.callbacks.current.flight?.sensitivity ?? 0.003;
          this.freeCamera.smoothing = this.callbacks.current.flight?.enabled
            ? (this.callbacks.current.flight.smoothing ?? 0.18)
            : 0;
          this.freeCamera.boost = this.callbacks.current.flight?.enabled
            ? 2.5
            : 5;
          this.freeCamera.onSpeedScale = this.callbacks.current.flight?.onSpeed;
          this.freeCamera.update(
            delta,
            this.callbacks.current.flight?.enabled
              ? this.callbacks.current.flight.speed
              : Math.max(2, radius) *
                  0.5 *
                  (this.callbacks.current.shot?.navigationSpeed ?? 1),
          );
          if (this.callbacks.current.flight?.enabled) {
            this.camera.far = Math.max(
              50000,
              this.camera.position.distanceTo(
                this.bounds.getCenter(new THREE.Vector3()),
              ) +
                radius * 4,
            );
            this.camera.updateProjectionMatrix();
          }
        } else if (!this.callbacks.current.shot?.preview)
          this.controls.update();
        const high = !this.loading && (
          !!this.callbacks.current.mapView?.enabled ||
          (!this.interacting &&
            !this.callbacks.current.shot?.playing &&
            now > this.settle));
        const resolutionChanged = this.updatePreviewResolution(high, frameMs);
        const movingAtmosphere =
          !this.callbacks.current.shot &&
          atmosphereAnimating(this.options.atmosphere);
        const qualityChanged=high!==this.quality;
        if (
          this.dirty ||
          qualityChanged ||
          resolutionChanged ||
		  (this.callbacks.current.viewportQuality!=="full"&&this.previewAA.samples<this.previewAA.maxSamples) ||
          movingAtmosphere
        ) {
          this.quality = high;
          this.ao.enabled =
            high &&
            !this.loading &&
            this.options.ao > 0 &&
            !this.options.transparent;
          this.normals.enabled = this.ao.enabled;
          const progressive=this.callbacks.current.viewportQuality!=="full";
          if(this.dirty||qualityChanged||resolutionChanged||movingAtmosphere||!high)this.previewAA.reset();
          this.previewAA.enabled=progressive;
          if(progressive){
            const size=this.renderer.getDrawingBufferSize(new THREE.Vector2());
            const sample=high?this.previewAA.samples:this.previewFrame++;
            withCameraJitter(this.camera,size.x,size.y,previewJitter(sample),()=>this.draw());
          }else this.draw();
          this.dirty = false;
        }
      } catch (error) {
        this.callbacks.current.onError(String(error));
        this.dirty = false;
      }
    };
    this.frame = requestAnimationFrame(tick);
    if (this.callbacks.current.active !== false) this.ensureLoaded();
  }
  ensureLoaded() {
    if (this.loadStarted || this.disposed) return;
    this.loadStarted = true;
    void this.load().catch((error) => {
      if (!this.disposed)
        this.callbacks.current.onError(
          error instanceof Error ? error.message : String(error),
        );
    });
  }
  draw(override?: CameraPose, time?: number, raw = false) {
    if (this.loading&&this.groups.size===0) return;
	if(this.exporting)this.previewAA.enabled=false;
	if(!raw)this.configureViewportPipeline();
    const current = this.callbacks.current.shot;
    const atmosphereTime =
      time ??
      current?.time ??
      (performance.now() - this.atmosphereStart) / 1000;
    if (
      !raw &&
      !override &&
      !this.exporting &&
      this.callbacks.current.viewportQuality === "full" &&
      current?.preview &&
      current.sequence.motionBlur?.enabled &&
      current.sequence.motionBlur.preview
    ) {
      const samples = temporalSamples(
        current.sequence,
        time ?? current.time ?? 0,
        current.sequence.fps,
        true,
        this.quality ? 8 : 2,
      ).map((sample) => ({
        ...sample,
        pose: sampleSequence(current.sequence, sample.time),
      }));
      if (samples.length > 1) {
        this.drawTemporal(samples, current.pose, time ?? current.time);
        return;
      }
    }
    const shot =
      current && override
        ? { ...current, pose: override, preview: true, showPath: false }
        : current;
    if (shot && this.camera instanceof THREE.PerspectiveCamera) {
      this.controls.enabled =
        !shot.preview &&
        shot.navigation !== "fly" &&
        !this.callbacks.current.pickingFocus;
      this.path.visible = shot.showPath && !shot.preview;
      if (shot.preview) this.applyShotPose(shot.pose);
      else
        this.renderer.toneMappingExposure =
          this.options.exposure * shot.pose.exposure;
    }
    const sky =
      this.options.transparent || this.options.sky === "none"
        ? undefined
        : this.sky?.background;
    this.skyBackground.configure(
      sky,
      this.camera,
      this.options.skyRotation,
      this.options.skyElevation,
      this.options.skyBrightness,
      this.options.sky === "night"
        ? this.light.position
            .clone()
            .sub(this.light.target.position)
            .normalize()
        : undefined,
    );
    this.scene.background = sky
      ? null
      : this.options.transparent
        ? null
        : new THREE.Color(this.options.background);
    this.beauty.clear = !sky;
    this.atmosphere.localLighting.power = this.options.volumeLightPower;
    if (this.studioLights.update(this.options.studioLights ?? [], this.bounds))
      this.renderer.shadowMap.needsUpdate = true;
    this.updateLightGuides();
    this.updateCloudGuides(atmosphereTime);
    this.updateTargetGuides(time ?? current?.time ?? 0);
    if (
      this.lighting.update(this.options, this.camera, this.bounds, this.light)
    )
      this.renderer.shadowMap.needsUpdate = true;
    this.lighting.depth.map = this.texture ?? null;
    this.atmosphere.configure(
      this.exporting||this.callbacks.current.viewportQuality==="full"?this.options:{...this.options,quality:"fast"},
      this.camera,
      this.normals.target.depthTexture!,
      this.light,
      atmosphereTime,
      this.span,
      this.bounds,
      {
        exporting: this.exporting,
        timeline: !!current,
        interactive: !this.exporting && !this.quality,
      },
    );
    this.mapOverlay.configure(
      this.callbacks.current.mapView,
      this.camera,
      this.normals.target.depthTexture!,
      this.bounds,
      this.origin,
      this.renderer.getPixelRatio(),
    );
    let focusDistance = shot?.pose.focusDistance ?? this.options.dofDistance;
    let focusPoint: THREE.Vector3 | undefined;

    if (
      (!shot || this.options.dofMode === "subject") &&
      this.options.dofTargetLocked
    ) {
      focusPoint = new THREE.Vector3(
        this.options.dofTargetX - (this.origin?.x ?? 0),
        this.options.dofTargetY,
        this.options.dofTargetZ - (this.origin?.z ?? 0),
      );
      focusDistance = focusDepth(focusPoint, this.camera);
    }
    if (shot?.pose.focusTarget) {
      const radius = this.bounds.isEmpty()
        ? 10
        : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
      const point = new THREE.Vector3(...shot.pose.focusTarget)
        .multiplyScalar(radius)
        .add(
          this.bounds.isEmpty()
            ? new THREE.Vector3()
            : this.bounds.getCenter(new THREE.Vector3()),
        );
      const depth = focusDepth(point, this.camera);
      focusPoint = point;
      if (depth > 0.001) focusDistance = depth;
    }
    const dofSettings = {
      ...this.options,
      dofDistance: focusDistance,
      quality: this.exporting
        ? ("fine" as const)
        : this.quality
          ? this.options.quality
          : ("fast" as const),
    };
    this.dof.configure(
      shot
        ? {
            ...dofSettings,
            dofEnabled:
              shot.preview &&
              (shot.pose.depthOfField ?? this.options.dofEnabled),
            dofDistance: focusDistance,
            dofAperture: shot.pose.aperture ?? this.options.dofAperture,
          }
        : dofSettings,
      this.camera,
      this.dofDepth.target.depthTexture!,
      !!this.callbacks.current.mapView?.enabled ||
        (!this.exporting && this.callbacks.current.viewportQuality!=="full") ||
        (!!this.callbacks.current.pickingFocus && !this.exporting),
      this.atmosphere.enabled ? this.atmosphere.cloudDepthTexture : undefined,
      focusPoint,
    );
    this.dofDepth.enabled = this.dof.enabled;
    this.focusGuides.update(
      this.camera,
      dofSettings,
      focusPoint,
      this.focusHover,
      !!this.callbacks.current.pickingFocus,
      !this.exporting &&
        !this.callbacks.current.mapView?.enabled &&
        (this.callbacks.current.focusGuides !== false ||
          !!this.callbacks.current.pickingFocus),
    );
    this.normals.enabled =
      this.hasWater ||
      this.ao.enabled ||
      this.atmosphere.enabled ||
      this.mapOverlay.enabled ||
      this.dof.enabled;
    if (
      this.hasWater ||
      this.atmosphere.enabled ||
      this.mapOverlay.enabled ||
      this.dof.enabled
    ) {
      const size = this.renderer.getDrawingBufferSize(new THREE.Vector2());
      if (
        this.normals.target.width !== size.x ||
        this.normals.target.height !== size.y
      )
        this.normals.setSize(size.x, size.y);
    }

    const bufferSize = this.renderer.getDrawingBufferSize(new THREE.Vector2());
    this.water.configure(
      this.options,
      this.camera,
      this.normals.target.depthTexture!,
      bufferSize.x,
      bufferSize.y,
    );
    const overlayEnabled = this.mapOverlay.enabled;
    if (raw) this.mapOverlay.enabled = false;
    // Keep the shadow shader/sampler configuration intact. Navigation only
    // freezes its cached maps; disabling the renderer's shadow system leaves
    // HDR-lit materials using stale sampler bindings in Three.js.
    if (!this.exporting && this.callbacks.current.viewportQuality!=="full" && !this.quality) {
      let ready=!!this.light.shadow.map;
      for(const light of [...this.lighting.points,this.lighting.spot])
        if(light.visible&&light.castShadow&&!light.shadow.map)ready=false;
      this.studioLights.group.traverse(object=>{
        const light=object as THREE.Light & {shadow?:{map:unknown}};
        if(light.isLight&&light.visible&&light.castShadow&&!light.shadow?.map)ready=false;
      });
      // A newly enabled light needs one valid depth map before it can be frozen.
      if(ready)this.renderer.shadowMap.needsUpdate=false;
    }
    try {
      this.composer.render();
    } finally {
      this.mapOverlay.enabled = overlayEnabled;
    }
  }
  applyShotPose(pose: CameraPose) {
    if (!(this.camera instanceof THREE.PerspectiveCamera)) return;
    const center = this.bounds.isEmpty()
      ? new THREE.Vector3()
      : this.bounds.getCenter(new THREE.Vector3());
    const scale = this.bounds.isEmpty()
      ? 10
      : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
    this.camera.position
      .fromArray(pose.position)
      .multiplyScalar(scale)
      .add(center);
    this.controls.target
      .fromArray(pose.target)
      .multiplyScalar(scale)
      .add(center);
    this.camera.up.set(0, 1, 0);
    this.camera.lookAt(this.controls.target);
    this.camera.rotateZ(THREE.MathUtils.degToRad(pose.roll));
    this.camera.fov = pose.fov;
    this.camera.updateProjectionMatrix();
    this.camera.updateMatrixWorld(true);
    this.renderer.toneMappingExposure = this.options.exposure * pose.exposure;
  }
  updateCloudGuides(time: number) {
    const current = this.callbacks.current.shot;
    this.cloudGizmos.update(
      this.options.atmosphere.cloudLayers,
      this.bounds,
      (current || this.options.atmosphere.animate ? time : 0) +
        this.options.atmosphere.time,
      this.callbacks.current.cloudEditor,
      !this.exporting &&
        !current?.preview &&
        !this.callbacks.current.mapView?.enabled,
      this.camera,
    );
  }
  drawTemporal(
    samples: MotionFrameSample[],
    center: CameraPose,
    time?: number,
  ) {
    for (const _step of this.temporalSteps(samples, center, time)) {
    }
  }
  *temporalSteps(
    samples: MotionFrameSample[],
    center: CameraPose,
    time?: number,
  ) {
    this.temporal ??= new TemporalAccumulator(this.renderer, this.mapOverlay);
    this.applyShotPose(center);
    this.mapOverlay.configure(
      this.callbacks.current.mapView,
      this.camera,
      this.normals.target.depthTexture!,
      this.bounds,
      this.origin,
      this.renderer.getPixelRatio(),
    );
    const size = this.renderer.getDrawingBufferSize(new THREE.Vector2());
    const screen = this.composer.renderToScreen,
      bloom = this.bloom.enabled,
      output = this.output.enabled,
      grade = this.grade.enabled,
      overlay = this.mapOverlay.enabled;
    try {
      this.temporal.begin(size.x, size.y);
      this.composer.renderToScreen = false;
      this.bloom.enabled = false;
      this.output.enabled = false;
      this.grade.enabled = false;
      this.mapOverlay.enabled = false;
      for (const sample of samples) {
        if (this.disposed) throw new Error("Render view closed.");
        this.draw(sample.pose, sample.time, true);
        this.temporal.add(
          this.composer.readBuffer.texture,
          sample.weight,
          sample.pose.exposure / Math.max(0.0001, center.exposure),
        );
        yield;
      }
      if (overlay) {
        this.draw(center, time ?? this.callbacks.current.shot?.time ?? 0, true);
      }
      this.mapOverlay.enabled = overlay;
      this.renderer.toneMappingExposure =
        this.options.exposure * center.exposure;
      this.temporal.finish(this.options);
    } finally {
      this.composer.renderToScreen = screen;
      this.bloom.enabled = bloom;
      this.output.enabled = output;
      this.grade.enabled = grade;
      this.mapOverlay.enabled = overlay;
      this.applyShotPose(center);
    }
  }
  updateTargetGuides(time: number) {
    const targets = this.callbacks.current.shot?.sequence.targets ?? [];
    this.targetGuides.visible =
      !!this.callbacks.current.targetEditor &&
      !this.exporting &&
      !this.callbacks.current.shot?.preview &&
      !this.callbacks.current.mapView?.enabled;
    for (const [id, marker] of this.targetMarkers)
      if (!targets.some((t) => t.id === id)) {
        marker.geometry.dispose();
        marker.material.dispose();
        this.targetGuides.remove(marker);
        this.targetMarkers.delete(id);
      }
    const radius = this.bounds.isEmpty()
      ? 10
      : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
    const center = this.bounds.isEmpty()
      ? new THREE.Vector3()
      : this.bounds.getCenter(new THREE.Vector3());
    for (const target of targets) {
      let marker = this.targetMarkers.get(target.id);
      if (!marker) {
        marker = new THREE.Mesh(
          new THREE.OctahedronGeometry(1),
          new THREE.MeshBasicMaterial({ color: 0x62aeff, depthTest: false }),
        );
        this.targetMarkers.set(target.id, marker);
        this.targetGuides.add(marker);
      }
      marker.position
        .fromArray(sampleTarget(target, time))
        .multiplyScalar(radius)
        .add(center);
      marker.scale.setScalar(
        radius *
          (target.id === this.callbacks.current.targetEditor?.selected
            ? 0.035
            : 0.022),
      );
      marker.material.color.setHex(
        target.id === this.callbacks.current.targetEditor?.selected
          ? 0xe7b66d
          : 0x62aeff,
      );
    }
    this.targetGuides.updateMatrixWorld(true);
  }
  pickTarget(event: PointerEvent): boolean {
    const editor = this.callbacks.current.targetEditor;
    if (!editor || this.callbacks.current.mapView?.enabled) return false;
    const rect = this.renderer.domElement.getBoundingClientRect(),
      ray = new THREE.Raycaster();
    ray.setFromCamera(
      new THREE.Vector2(
        ((event.clientX - rect.left) / rect.width) * 2 - 1,
        (-(event.clientY - rect.top) / rect.height) * 2 + 1,
      ),
      this.camera,
    );
    if (editor.picking) {
      const hit = ray.intersectObject(this.terrain, true)[0];
      if (hit && !this.bounds.isEmpty()) {
        const radius = Math.max(
          2,
          this.bounds.getBoundingSphere(new THREE.Sphere()).radius,
        );
        editor.onPosition(
          hit.point
            .clone()
            .sub(this.bounds.getCenter(new THREE.Vector3()))
            .divideScalar(radius)
            .toArray() as [number, number, number],
        );
      }
      return true;
    }
    if (
      !this.targetGuides.visible ||
      this.lightTransform.axis ||
      this.callbacks.current.lightEditor?.placing
    )
      return false;
    const hit = ray.intersectObjects([...this.targetMarkers.values()])[0];
    if (!hit) return false;
    for (const [id, marker] of this.targetMarkers)
      if (marker === hit.object) editor.onSelect(id);
    return true;
  }
  updateLightGuides() {
    const editor = this.callbacks.current.lightEditor;
    const visible =
      !!editor &&
      !this.exporting &&
      !this.callbacks.current.mapView?.enabled &&
      !this.callbacks.current.shot?.preview;
    this.lightGuides.visible = visible;
    this.lightTransform.enabled = visible && !editor?.placing;
    if (!visible || this.bounds.isEmpty()) {
      this.lightTransform.detach();
      return;
    }
    const objects = this.options.studioLights ?? [];
    const signature = JSON.stringify([
      objects,
      editor.selected,
      editor.target,
      this.bounds.min.toArray(),
      this.bounds.max.toArray(),
    ]);
    if (signature !== this.lightGuideSignature) {
      this.lightGuideSignature = signature;
      for (const marker of this.lightMarkers.values()) {
        marker.geometry.dispose();
        (marker.material as THREE.Material).dispose();
        marker.removeFromParent();
      }
      this.lightMarkers.clear();
      this.selectedLightGuide?.dispose();
      this.selectedLightGuide?.removeFromParent();
      this.selectedLightGuide = undefined;
      const radius = Math.max(
        1,
        this.bounds.getBoundingSphere(new THREE.Sphere()).radius,
      );
      const center = this.bounds.getCenter(new THREE.Vector3());
      for (const o of objects) {
        const marker = new THREE.Mesh(
          new THREE.SphereGeometry(radius * 0.015, 10, 8),
          new THREE.MeshBasicMaterial({
            color: o.id === editor.selected ? 0x62aeff : o.color,
            wireframe: !o.enabled,
            depthTest: false,
            transparent: true,
            opacity: o.enabled ? 1 : 0.45,
          }),
        );
        marker.position
          .fromArray(o.position)
          .multiplyScalar(radius)
          .add(center);
        marker.renderOrder = 10000;
        this.lightMarkers.set(o.id, marker);
        this.lightGuides.add(marker);
      }
      const selected = objects.find((o) => o.id === editor.selected);
      this.lightSightline.visible = !!selected && selected.type !== "point";
      if (selected) {
        const target = new THREE.Vector3()
          .fromArray(selected.target)
          .multiplyScalar(radius)
          .add(center);
        const position = new THREE.Vector3()
          .fromArray(selected.position)
          .multiplyScalar(radius)
          .add(center);
        this.lightAnchor.position.copy(editor.target ? target : position);
        this.lightSightline.geometry.dispose();
        this.lightSightline.geometry = new THREE.BufferGeometry().setFromPoints(
          [position, target],
        );
        const light = this.studioLights.lights.get(selected.id);
        if (light instanceof THREE.SpotLight)
          this.selectedLightGuide = new THREE.SpotLightHelper(light, 0x62aeff);
        if (light instanceof THREE.PointLight)
          this.selectedLightGuide = new THREE.PointLightHelper(
            light,
            radius * 0.035,
            0x62aeff,
          );
        if (this.selectedLightGuide)
          this.lightGuides.add(this.selectedLightGuide);
        this.lightTransform.attach(this.lightAnchor);
      } else this.lightTransform.detach();
    }

    if (
      editor.selected &&
      this.lightMarkers.has(editor.selected) &&
      !this.lightTransform.object
    )
      this.lightTransform.attach(this.lightAnchor);
    this.lightGuides.updateMatrixWorld(true);
  }
  pickLight(event: PointerEvent): boolean {
    const editor = this.callbacks.current.lightEditor;
    if (
      !editor ||
      this.callbacks.current.mapView?.enabled ||
      this.callbacks.current.shot?.preview
    )
      return false;
    if (this.lightTransform.axis) return false;
    const rect = this.renderer.domElement.getBoundingClientRect();
    const ray = new THREE.Raycaster();
    ray.setFromCamera(
      new THREE.Vector2(
        ((event.clientX - rect.left) / rect.width) * 2 - 1,
        -((event.clientY - rect.top) / rect.height) * 2 + 1,
      ),
      this.camera,
    );
    if (!editor.placing) {
      const hit = ray.intersectObjects([...this.lightMarkers.values()])[0];
      if (!hit) return false;
      for (const [id, marker] of this.lightMarkers)
        if (marker === hit.object) editor.onSelect(id);
      return true;
    }
    if (!editor.selected || this.bounds.isEmpty()) return false;
    const hit = ray.intersectObject(this.terrain, true)[0];
    if (!hit) return true;
    const radius = Math.max(
      1,
      this.bounds.getBoundingSphere(new THREE.Sphere()).radius,
    );
    const point = hit.point.clone();

    if (!editor.target && hit.face)
      point.add(
        hit.face.normal
          .clone()
          .transformDirection(hit.object.matrixWorld)
          .multiplyScalar(radius * 0.025),
      );
    point.sub(this.bounds.getCenter(new THREE.Vector3())).divideScalar(radius);
    editor.onPosition(
      editor.selected,
      point.toArray().map((v) => THREE.MathUtils.clamp(v, -100, 100)) as [
        number,
        number,
        number,
      ],
      !!editor.target,
    );
    return true;
  }
  updatePreviewResolution(high: boolean, frameMs = 16) {
    if (this.exporting) return false;
    this.renderer.getSize(this.viewportSize);
    const ratio = this.callbacks.current.viewportQuality==="full"&&high ? Math.min(window.devicePixelRatio||1,2) : this.previewPerformance.update(
      !high,
      this.viewportSize.x,
      this.viewportSize.y,
      window.devicePixelRatio || 1,
      frameMs,
    );
    if (Math.abs(this.renderer.getPixelRatio() - ratio) < 0.001) return false;
    this.renderer.setPixelRatio(ratio);
    this.composer.setPixelRatio(ratio);
    this.size(this.viewportSize.x, this.viewportSize.y);
    return true;
  }
  resize() {
    const bounds = this.host.getBoundingClientRect();
    if (bounds.width < 2 || bounds.height < 2 || this.exporting) return;
    const aspect = this.options.aspect;
    const width = Math.max(
        1,
        Math.floor(Math.min(bounds.width, bounds.height * aspect)),
      ),
      height = Math.max(1, Math.floor(width / aspect));
    this.size(width, height);
    this.renderer.domElement.style.width = `${width}px`;
    this.renderer.domElement.style.height = `${height}px`;
    this.dirty = true;
  }
  size(width: number, height: number) {
    const aspect = width / height;
    if (this.camera instanceof THREE.OrthographicCamera) {
      this.camera.left = (-this.span * aspect) / 2;
      this.camera.right = (this.span * aspect) / 2;
      const shift = this.callbacks.current.mapView?.enabled
        ? 0
        : this.options.frameVerticalShift;
      this.camera.top = this.span * (0.5 + shift);
      this.camera.bottom = this.span * (-0.5 + shift);
    } else this.camera.aspect = aspect;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(width, height, false);
    this.composer.setSize(width, height);
	const fullViewport=this.exporting||this.callbacks.current.viewportQuality==="full";
    const scale = Math.min(
      renderQuality[fullViewport?this.options.quality:"fast"].aoScale,
      (fullViewport?2048:512) / width,
    );
    const aoWidth = Math.max(1, Math.round(width * scale)),
      aoHeight = Math.max(1, Math.round(height * scale));
    this.ao.setSize(aoWidth, aoHeight);
    this.normals.setSize(aoWidth, aoHeight);
  }
  apply(options: RenderSettings, skyFile?: File) {
    if (this.loading) return;
    const old = this.options;
    this.options = options;
	this.pipelineFull=undefined;
    if (old.pitchLocked !== options.pitchLocked) {
      [this.controls.minPolarAngle, this.controls.maxPolarAngle] = polarLimits(
        options.pitchLocked,
        this.controls.getPolarAngle(),
      );
    }
    this.scene.background = options.transparent
      ? null
      : this.sky && options.sky !== "none"
        ? this.sky.background
        : new THREE.Color(options.background);
    this.renderer.setClearColor(
      options.background,
      options.transparent ? 0 : 1,
    );
    this.renderer.toneMappingExposure = options.exposure;
    this.grade.configure(options);
    this.bloom.strength = options.bloom;
    this.bloom.enabled = options.bloom > 0 && !options.transparent;
    for (const material of Object.values(this.materials)) {
      if (material instanceof THREE.MeshStandardMaterial) {
        material.envMapIntensity = options.reflections;
        material.roughness = material.transparent
          ? Math.max(0.04, options.roughness * 0.25)
          : options.roughness;
      }
    }
    this.ambient.intensity = options.ambient;
    this.light.intensity = options.sun;
    this.light.color.set(options.warmth);
    this.light.shadow.radius = (options.softness * options.shadow) / 2048;
    this.scene.environmentIntensity = options.environment;
    this.scene.backgroundIntensity = options.skyBrightness;
    this.scene.backgroundRotation.y = this.scene.environmentRotation.y =
      (options.skyRotation * Math.PI) / 180;
    this.ao.blendIntensity = options.ao;
	const fullViewport=this.exporting||this.callbacks.current.viewportQuality==="full";
    const quality = renderQuality[fullViewport?options.quality:"fast"];
    this.ao.updateGtaoMaterial({
      radius: options.aoRadius,
      samples: quality.samples,
    });
    this.ao.updatePdMaterial({ radius: 6, rings: 2, samples: quality.denoise });
    for (const target of [
      this.composer.renderTarget1,
      this.composer.renderTarget2,
    ]) {
      const samples = Math.min(
        fullViewport?quality.aa:0,
        this.renderer.capabilities.maxSamples,
      );
      if (target.samples !== samples) {
        target.samples = samples;
        target.dispose();
      }
    }
    const resolution = Math.min(
      fullViewport?options.shadow:1024,
      this.renderer.capabilities.maxTextureSize,
    );
    if (this.light.shadow.mapSize.x !== resolution) {
      this.light.shadow.mapSize.set(resolution, resolution);
      this.light.shadow.map?.dispose();
      this.light.shadow.map = null;
    }
    this.updateSun();
    if (old.aspect !== options.aspect || old.quality !== options.quality)
      this.resize();
    if (
      (old.frameMargin !== options.frameMargin ||
        old.frameVerticalShift !== options.frameVerticalShift) &&
      this.camera instanceof THREE.OrthographicCamera &&
      !this.callbacks.current.mapView?.enabled
    )
      this.fit();
    if (this.skyKey !== options.sky || this.skyFile !== skyFile) {
      this.skyKey = options.sky;
      this.skyFile = skyFile;
      this.skyTask = this.setSky(options.sky, skyFile);
    }
    this.dirty = true;
  }
  async setSky(id: string, file?: File) {
    const generation = ++this.skyGeneration;
    this.callbacks.current.onSkyLoading(true);
    try {
      if (id === "none") {
        this.scene.environment = null;
        this.scene.background = this.options.transparent
          ? null
          : new THREE.Color(this.options.background);
        this.sky?.background.dispose();
        this.sky?.lighting.dispose();
        this.sky = undefined;
        this.environmentMap?.dispose();
        this.environmentMap = undefined;
        this.dirty = true;
        return;
      }
      const sky = await loadRenderSky(id, file);
      if (this.disposed || generation !== this.skyGeneration) {
        sky.background.dispose();
        sky.lighting.dispose();
        return;
      }
      let map: THREE.WebGLRenderTarget;
      try {
        map = this.pmrem.fromEquirectangular(sky.lighting);
      } catch (error) {
        sky.background.dispose();
        sky.lighting.dispose();
        throw error;
      }
      this.sky?.background.dispose();
      this.sky?.lighting.dispose();
      this.environmentMap?.dispose();
      this.sky = sky;
      this.environmentMap = map;
      this.scene.environment = map.texture;
      this.scene.background = this.options.transparent ? null : sky.background;
      this.updateSun();
      this.dirty = true;
    } catch (error) {
      if (!this.disposed && generation === this.skyGeneration)
        this.callbacks.current.onError(
          `Skybox: ${error instanceof Error ? error.message : String(error)}`,
        );
    } finally {
      if (!this.disposed && generation === this.skyGeneration)
        this.callbacks.current.onSkyLoading(false);
    }
  }
  updateSun() {
    const center = this.bounds.isEmpty()
      ? new THREE.Vector3()
      : this.bounds.getCenter(new THREE.Vector3());
    const radius = this.bounds.isEmpty()
      ? 10
      : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
    const az = (this.options.azimuth * Math.PI) / 180,
      el = (this.options.elevation * Math.PI) / 180;
    let direction = new THREE.Vector3(
      Math.sin(az) * Math.cos(el),
      Math.sin(el),
      Math.cos(az) * Math.cos(el),
    );
    if (this.options.sunLinked && this.sky && this.options.sky !== "none")
      direction = this.sky.sun
        .clone()
        .applyAxisAngle(
          new THREE.Vector3(0, 1, 0),
          (this.options.skyRotation * Math.PI) / 180,
        );
    this.light.target.position.copy(center);
    this.light.position.copy(center).add(direction.multiplyScalar(radius * 3));
    const camera = this.light.shadow.camera;
    camera.position.copy(this.light.position);
    camera.lookAt(center);
    camera.updateMatrixWorld(true);
    const fit = new THREE.Box3();
    if (!this.bounds.isEmpty())
      for (const x of [this.bounds.min.x, this.bounds.max.x])
        for (const y of [this.bounds.min.y, this.bounds.max.y])
          for (const z of [this.bounds.min.z, this.bounds.max.z])
            fit.expandByPoint(
              new THREE.Vector3(x, y, z).applyMatrix4(
                camera.matrixWorldInverse,
              ),
            );
    if (fit.isEmpty()) {
      camera.left = camera.bottom = -radius;
      camera.right = camera.top = radius;
      camera.near = 0.1;
      camera.far = radius * 7 + 100;
    } else {
      camera.left = fit.min.x - 1;
      camera.right = fit.max.x + 1;
      camera.bottom = fit.min.y - 1;
      camera.top = fit.max.y + 1;
      camera.near = Math.max(0.1, -fit.max.z - 4);
      camera.far = Math.max(camera.near + 1, -fit.min.z + 4);
    }
    this.light.shadow.normalBias = Math.max(
      0.005,
      Math.min(
        0.06,
        ((camera.right - camera.left) / this.light.shadow.mapSize.x) * 0.075,
      ),
    );
    camera.updateProjectionMatrix();
    this.renderer.shadowMap.needsUpdate = true;
  }
  fit() {
    if (this.exporting) return;
    if (this.bounds.isEmpty()) return;
    const sphere = this.bounds.getBoundingSphere(new THREE.Sphere());
    this.span =
      Math.max(2, sphere.radius * 2.1) / Math.min(1, this.options.aspect);
    if (this.callbacks.current.mapView?.enabled) this.span = this.mapSpan();
    this.camera.zoom = 1;
    const direction = this.camera.position
      .clone()
      .sub(this.controls.target)
      .normalize();
    this.controls.target.copy(sphere.center);
    this.camera.position
      .copy(sphere.center)
      .add(direction.multiplyScalar(Math.max(10, sphere.radius * 3)));
    this.camera.far = Math.max(1000, sphere.radius * 12);
    this.camera.updateProjectionMatrix();
    this.controls.update();
    if (
      this.camera instanceof THREE.OrthographicCamera &&
      !this.callbacks.current.mapView?.enabled
    ) {
      this.span = fittedOrthoSpan(
        this.bounds,
        this.camera,
        this.options.aspect,
        this.options.frameMargin,
        this.options.frameVerticalShift,
      );
    }
    this.resize();
    this.dirty = true;
    this.updatePath();
  }
  flightPosition(): [number, number, number] {
    return this.camera.position
      .clone()
      .add(this.origin ?? new THREE.Vector3())
      .toArray() as [number, number, number];
  }
  private focusHit(event: Pick<PointerEvent, "clientX" | "clientY">) {
    const rect = this.renderer.domElement.getBoundingClientRect();
    return pickFocusSurface(
      this.terrain,
      this.camera,
      new THREE.Vector2(
        ((event.clientX - rect.left) / rect.width) * 2 - 1,
        -((event.clientY - rect.top) / rect.height) * 2 + 1,
      ),
    );
  }
  configureFocusPicking() {
    const picking = !!this.callbacks.current.pickingFocus;
    this.focusHover = undefined;
    this.pendingFocusPointer = undefined;
    this.renderer.domElement.style.cursor = picking ? "crosshair" : "";
    this.controls.enabled =
      !picking &&
      !this.callbacks.current.flight?.enabled &&
      !this.callbacks.current.shot?.preview &&
      this.callbacks.current.shot?.navigation !== "fly";
    if (picking) {
      this.freeCamera.enabled = false;
      this.renderer.domElement.focus();
    }
    this.dirty = true;
  }
  teleport(position: [number, number, number]) {
    if (!position.every((n) => Number.isFinite(n) && Math.abs(n) <= 30000000))
      return;
    const next = new THREE.Vector3(...position).sub(
      this.origin ?? new THREE.Vector3(),
    );
    const delta = next.sub(this.camera.position);
    this.camera.position.add(delta);
    this.freeCamera.stop();
    this.controls.target.add(delta);
    this.dirty = true;
    this.scheduleViewChange();
    this.renderer.domElement.focus();
  }
  flightReturn() {
    if (!this.flightStart) return;
    this.freeCamera.stop();
    this.camera.position.copy(this.flightStart.position);
    this.controls.target.copy(this.flightStart.target);
    this.camera.lookAt(this.controls.target);
    this.dirty = true;
    this.scheduleViewChange();
  }
  configureFlight() {
    if (this.exporting || this.callbacks.current.cinematic) return;
    const flight = this.callbacks.current.flight;
    let fov = flight?.fov ?? 70;
    if (flight?.enabled && !this.flightActive) {
      this.clearOrbitMomentum();
      this.flightActive = true;
      this.flightStart = {
        position: this.camera.position.clone(),
        target: this.controls.target.clone(),
        span: this.span,
      };
      if (this.camera instanceof THREE.OrthographicCamera) {
        const height =
          (this.camera.top - this.camera.bottom) / this.camera.zoom;
        const distance = Math.max(
          0.01,
          this.camera.position.distanceTo(this.controls.target),
        );
        fov = THREE.MathUtils.clamp(
          THREE.MathUtils.radToDeg(2 * Math.atan(height / (2 * distance))),
          1,
          150,
        );
        const next = new THREE.PerspectiveCamera(
          fov,
          1,
          0.05,
          Math.max(50000, this.camera.far),
        );
        next.position.copy(this.camera.position);
        next.quaternion.copy(this.camera.quaternion);
        this.swapNavigationCamera(next);
      } else fov = this.camera.fov;
      flight.onFov?.(fov);
      const direction = this.camera.getWorldDirection(new THREE.Vector3());
      this.controls.target
        .copy(this.camera.position)
        .addScaledVector(direction, 16);
      this.controls.enabled = false;
      this.freeCamera.enabled = true;
      this.renderer.domElement.focus();
    } else if (!flight?.enabled && this.flightActive) {
      this.flightActive = false;
      this.freeCamera.enabled = false;

      if (!this.callbacks.current.mapView?.enabled) {
        const direction = this.camera.getWorldDirection(new THREE.Vector3());
        const hit = pickFocusSurface(
          this.terrain,
          this.camera,
          new THREE.Vector2(),
        );
        const distance = hit
          ? Math.max(0.25, this.camera.position.distanceTo(hit.point))
          : Math.max(4, this.camera.position.distanceTo(this.controls.target));
        this.controls.target
          .copy(this.camera.position)
          .addScaledVector(direction, distance);
        [this.controls.minPolarAngle, this.controls.maxPolarAngle] =
          polarLimits(
            this.options.pitchLocked,
            Math.acos(THREE.MathUtils.clamp(-direction.y, -1, 1)),
          );
        this.controls.enableDamping = true;
        this.clearOrbitMomentum();
      } else this.controls.enableDamping = false;
      this.controls.enabled = true;
    }
    if (flight?.enabled && this.camera instanceof THREE.PerspectiveCamera) {
      this.camera.fov = fov;
      this.camera.updateProjectionMatrix();
    }
    this.resize();
    this.dirty = true;
    this.scheduleViewChange();
  }
  private clearOrbitMomentum() {
    const position = this.camera.position.clone(),
      quaternion = this.camera.quaternion.clone(),
      target = this.controls.target.clone(),
      zoom = this.camera.zoom,
      damping = this.controls.enableDamping;
    this.controls.enableDamping = false;
    this.controls.update();
    this.camera.position.copy(position);
    this.camera.quaternion.copy(quaternion);
    this.controls.target.copy(target);
    this.camera.zoom = zoom;
    this.camera.updateProjectionMatrix();
    this.controls.enableDamping = damping;
    this.camera.updateMatrixWorld(true);
  }
  private swapNavigationCamera(
    camera: THREE.OrthographicCamera | THREE.PerspectiveCamera,
  ) {
    camera.updateMatrixWorld(true);
    this.camera = camera;
    this.beauty.camera = camera;
    this.normals.setCamera(camera);
    this.dofDepth.setCamera(camera);
    this.ao.camera = camera;
    this.ao.gtaoMaterial.defines.PERSPECTIVE_CAMERA =
      camera instanceof THREE.PerspectiveCamera ? 1 : 0;
    this.ao.gtaoMaterial.needsUpdate = true;
    this.controls.object = camera;
    this.controls.enableDamping = camera instanceof THREE.OrthographicCamera;
    this.freeCamera.setCamera(camera);
    this.lightTransform.camera = camera;
    this.cloudGizmos.transform.camera = camera;
  }
  snapshot(): SavedCameraView {
    const center = this.bounds.isEmpty()
      ? new THREE.Vector3()
      : this.bounds.getCenter(new THREE.Vector3());
    const scale = this.bounds.isEmpty()
      ? 10
      : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
    return {
      projection:
        this.camera instanceof THREE.PerspectiveCamera
          ? "perspective"
          : "orthographic",
      span:
        this.camera instanceof THREE.OrthographicCamera ? this.span : undefined,
      zoom: this.camera.zoom,
      position: this.camera.position
        .clone()
        .sub(center)
        .divideScalar(scale)
        .toArray() as [number, number, number],
      target: this.controls.target
        .clone()
        .sub(center)
        .divideScalar(scale)
        .toArray() as [number, number, number],
      fov:
        this.camera instanceof THREE.PerspectiveCamera ? this.camera.fov : 45,
      roll: snapshotRoll(this.camera, this.controls.target),
      exposure: this.callbacks.current.shot?.pose.exposure ?? 1,
    };
  }
	 capturePose(point:import("./bridge").CapturePoint):CameraPose|undefined {
	  if(this.loading||this.bounds.isEmpty())return;
	  const center=this.bounds.getCenter(new THREE.Vector3()),radius=Math.max(2,this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
	  const position=new THREE.Vector3(...point.position).sub(this.origin??new THREE.Vector3()).sub(center).divideScalar(radius);
	  const yaw=THREE.MathUtils.degToRad(point.yaw),pitch=THREE.MathUtils.degToRad(point.pitch);
	  const target=position.clone().add(new THREE.Vector3(-Math.sin(yaw)*Math.cos(pitch),-Math.sin(pitch),Math.cos(yaw)*Math.cos(pitch)).multiplyScalar(16/radius));
	  if([...position.toArray(),...target.toArray()].some(value=>!Number.isFinite(value)||Math.abs(value)>10000))return;
	  return {position:position.toArray() as [number,number,number],target:target.toArray() as [number,number,number],fov:70,roll:0,exposure:1};
	 }
  updatePath() {
    this.path.traverse((item) => {
      if (item instanceof THREE.Line || item instanceof THREE.Mesh) {
        item.geometry.dispose();
        (item.material as THREE.Material).dispose();
      }
    });
    this.path.clear();
    const shot = this.callbacks.current.shot;
    if (shot) {
      const center = this.bounds.isEmpty()
        ? new THREE.Vector3()
        : this.bounds.getCenter(new THREE.Vector3());
      const scale = this.bounds.isEmpty()
        ? 10
        : Math.max(2, this.bounds.getBoundingSphere(new THREE.Sphere()).radius);
      const world = (v: number[]) =>
        new THREE.Vector3(...(v as [number, number, number]))
          .multiplyScalar(scale)
          .add(center);
      const points = Array.from({ length: 257 }, (_, i) =>
        world(
          sampleSequence(shot.sequence, (i / 256) * shot.sequence.duration)
            .position,
        ),
      );
      this.path.add(
        new THREE.Line(
          new THREE.BufferGeometry().setFromPoints(points),
          new THREE.LineBasicMaterial({
            color: 0x9c95ff,
            depthTest: false,
            transparent: true,
            opacity: 0.9,
          }),
        ),
      );
      for (const key of shot.sequence.keys) {
        const pose = sampleSequence(shot.sequence, key.time);
        const point = new THREE.Mesh(
          new THREE.SphereGeometry(scale * 0.018, 8, 8),
          new THREE.MeshBasicMaterial({ color: 0xffce80, depthTest: false }),
        );
        point.position.copy(world(pose.position));
        this.path.add(point);
        this.path.add(
          new THREE.Line(
            new THREE.BufferGeometry().setFromPoints([
              world(pose.position),
              world(pose.target),
            ]),
            new THREE.LineBasicMaterial({
              color: 0xffce80,
              transparent: true,
              opacity: 0.25,
              depthTest: false,
            }),
          ),
        );
      }
    }
    this.dirty = true;
  }
  view(angle: number, tilt: number) {
    if (this.exporting) return;
    if (
      !this.callbacks.current.cinematic &&
      !this.callbacks.current.flight?.enabled &&
      this.camera instanceof THREE.PerspectiveCamera
    ) {
      this.span = Math.max(
        0.1,
        this.camera.position.distanceTo(this.controls.target) *
          2 *
          Math.tan(THREE.MathUtils.degToRad(this.camera.getEffectiveFOV() / 2)),
      );
      const next = new THREE.OrthographicCamera(
        -1,
        1,
        1,
        -1,
        0.05,
        this.camera.far,
      );
      next.position.copy(this.camera.position);
      next.quaternion.copy(this.camera.quaternion);
      this.swapNavigationCamera(next);
      this.resize();
    }
    this.controls.minPolarAngle = 0;
    this.controls.maxPolarAngle = Math.PI;
    const a = (angle * Math.PI) / 180,
      t = (tilt * Math.PI) / 180;
    const distance = Math.max(
      10,
      this.camera.position.distanceTo(this.controls.target),
    );
    this.camera.position
      .copy(this.controls.target)
      .add(
        new THREE.Vector3(
          Math.sin(a) * Math.cos(t),
          Math.sin(t),
          Math.cos(a) * Math.cos(t),
        ).multiplyScalar(distance),
      );
    this.controls.update();
    [this.controls.minPolarAngle, this.controls.maxPolarAngle] = polarLimits(
      this.options.pitchLocked,
      this.controls.getPolarAngle(),
    );
    this.dirty = true;
  }
  commit(chunk: DecodedChunk) {
    this.ledger.update(chunk);
    if (!this.origin)
      this.origin = new THREE.Vector3(chunk.x * 16, 0, chunk.z * 16);
    const group = new THREE.Group();
    group.position.set(
      chunk.x * 16 - this.origin.x,
      0,
      chunk.z * 16 - this.origin.z,
    );
    const sourceLights = chunk.lights ?? [];
    group.userData.emitterCount = sourceLights.length;
    this.analysisRevision++;
    group.userData.sceneSignals = chunk.signals ?? {};
    this.lighting.setChunk(
      `${chunk.x},${chunk.z}`,
      sourceLights,
      group.position,
      chunk.parts.every((part) => !!part.blockLight),
    );
    for (const part of chunk.parts) {
      if (part.mode === "billboard") continue;
      const material = this.materials[part.mode];
      if (!material) throw new Error("Unknown render material");
      const geometry = new THREE.BufferGeometry();
      geometry.setAttribute(
        "position",
        new THREE.BufferAttribute(part.positions, 3),
      );
      geometry.setAttribute(
        "normal",
        new THREE.BufferAttribute(part.normals, 3, true),
      );
      geometry.setAttribute("uv", new THREE.BufferAttribute(part.uvs, 2));
      geometry.setAttribute(
        "blockLight",
        new THREE.BufferAttribute(
          part.blockLight ?? new Uint8Array(part.positions.length / 3),
          1,
        ),
      );
      geometry.setAttribute(
        "foliage",
        new THREE.BufferAttribute(
          part.mode === "mask"
            ? foliageAttributes(part.positions)
            : new Uint8Array(part.positions.length / 3),
          1,
        ),
      );
      geometry.setAttribute(
        "atlasTile",
        new THREE.BufferAttribute(
          atlasTileAttributes(part.uvs, this.texture?.image.width ?? 2048),
          3,
        ),
      );
      geometry.setIndex(new THREE.BufferAttribute(part.indices, 1));
      geometry.setAttribute(
        "blockEmission",
        new THREE.BufferAttribute(
          emissionAttributes(
            part.positions,
            part.normals,
            part.indices,
            sourceLights,
          ),
          1,
		  true,
        ),
      );
      const b = part.bounds;
      geometry.boundingBox = new THREE.Box3(
        new THREE.Vector3(b[0], b[1], b[2]),
        new THREE.Vector3(b[3], b[4], b[5]),
      );
      geometry.boundingSphere = geometry.boundingBox.getBoundingSphere(
        new THREE.Sphere(),
      );
      const mesh = new THREE.Mesh(geometry, material);
      if (part.mode === "water") this.hasWater = true;
      mesh.castShadow =
        part.mode !== "water" &&
        part.mode !== "blend" &&
        part.mode !== "unlit-blend";
      mesh.receiveShadow = true;
      mesh.customDistanceMaterial = this.lighting.depth;
      group.add(mesh);
      const box = geometry.boundingBox.clone().translate(group.position);
      this.bounds.union(box);
    }
    const key = `${chunk.x},${chunk.z}`;
    const old = this.groups.get(key);
    if (old) {
      old.traverse((item) => {
        if (item instanceof THREE.Mesh) item.geometry.dispose();
      });
      this.terrain.remove(old);
    }
    this.groups.set(key, group);
    this.terrain.add(group);
    this.renderer.shadowMap.needsUpdate = true;
    this.dirty = true;
  }
  async analyse(progress?: AnalysisProgressCallback, signal?: AbortSignal): Promise<ModelAnalysis> {
    checkAnalysisAbort(signal);
    if (this.loading || this.disposed || !this.pixels || !this.texture)
      throw new Error("The model has not finished loading.");
    // Full terrain is already loaded (previewStep=0). Cloud-clearance rasterizing
    // is independent secondary lighting and must never hold up scene analysis.
    const revision = this.analysisRevision, source = this.callbacks.current.sourceName;
    // Geometry evidence stays cached across navigation. Cloud framing must use
    // the current camera, including top-down and cinematic camera orientations.
    const withView = (value: ModelAnalysis): ModelAnalysis => {
      if (!this.camera || !this.controls) return value;
      this.camera.updateMatrixWorld();
      const direction = this.camera.getWorldDirection(new THREE.Vector3());
      const distance = Math.max(1,this.camera.position.distanceTo(this.controls.target));
      return {...value,view:{projection:this.camera instanceof THREE.PerspectiveCamera?"perspective":"orthographic",position:this.camera.position.toArray(),target:this.camera.position.clone().addScaledVector(direction,distance).toArray(),
        up:new THREE.Vector3(0,1,0).applyQuaternion(this.camera.quaternion).toArray()}};
    };
    const cached = this.analysisCache.get(revision, source);
    if (cached) {
      progress?.(0.91, {stage:"context", label:"Reusing analysis of the unchanged scene", detail:"Geometry, atlas and block evidence are unchanged", cached:true});
      await analysisYield(); checkAnalysisAbort(signal);
      if (this.disposed || this.analysisRevision !== revision)
        throw new Error("The scene changed during analysis. Please run Auto styling again.");
      return withView(cached);
    }
    const groups = [...this.groups.entries()].sort(([a], [b]) => a.localeCompare(b));
    const meshes: AnalysisMesh[] = [];
    const materialModes = new Map(Object.entries(this.materials).map(([mode,material]) => [material,mode]));
    const signals: Record<string,number> = {};
    let lights = 0;
    for (const [,group] of groups) {
      lights += group.userData.emitterCount ?? 0;
      for (const [key,value] of Object.entries(group.userData.sceneSignals ?? {}))
        if (typeof value === "number" && Number.isFinite(value) && value >= 0) signals[key] = (signals[key]??0)+value;
      for (const node of group.children) if (node instanceof THREE.Mesh)
        meshes.push({geometry:node.geometry, mode:materialModes.get(node.material)??"solid", offset:group.position.toArray()});
    }
    const alive = (value: number, status?: Parameters<AnalysisProgressCallback>[1]) => {
      if (this.disposed) throw new Error("The model view was closed during analysis.");
      checkAnalysisAbort(signal); progress?.(value,status);
    };
    const packed = await sampleScene(meshes, {pixels:this.pixels, width:this.texture.image.width, height:this.texture.image.height}, alive, signal);
    const analysis = await analysePacked(packed, {sourceName:source, signals, lights, triangles:packed.triangles,
      bounds:[...this.bounds.min.toArray(),...this.bounds.max.toArray()]}, alive, signal);
    checkAnalysisAbort(signal);
    if (this.disposed || this.analysisRevision !== revision)
      throw new Error("The scene changed during analysis. Please run Auto styling again.");
    this.analysisCache.set(revision, source, analysis);
    return withView(analysis);
  }
  async load() {
	const previousView=this.loading?undefined:this.snapshot();
	const previousBounds=previousView?this.bounds.clone():undefined;
	this.loading=true;
    const request: ScenePageRequest = {
	  previewStep:0,
      after: 0,
      target: 0,
      cursor: null,
      focus: this.callbacks.current.viewportFocus??[0,0],
      limit: 8,
      atlasRevision: this.renderedAtlasRevision,
      scope: 0,
      terrainOnly: true,
      bounds: this.callbacks.current.selectionBounds,
    };
    let loaded = 0;
    while (!this.disposed) {
      const page = await api.GetRenderScenePage(
        this.callbacks.current.worldId,
        request,
      );
      if (this.disposed) return;
      if (page.reset && loaded > 0 && !previousView) {
        this.terrain.traverse((item) => {
          if (item instanceof THREE.Mesh) item.geometry.dispose();
        });
        this.terrain.clear();
        this.groups.clear();
        this.lighting.clear();
        this.ledger.clear();
        this.bounds.makeEmpty();
        this.origin = undefined;
        loaded = 0;
        this.dirty = true;
        this.renderer.shadowMap.needsUpdate = true;
      }
      const data = await this.decoder.decode(page);
      if (this.disposed) return;
      if (!this.texture) {
        const size = page.atlasSize;
        this.pixels = new Uint8Array(size * size * 4);
        this.texture = new THREE.DataTexture(
          this.pixels,
          size,
          size,
          THREE.RGBAFormat,
        );
        this.texture.colorSpace = THREE.SRGBColorSpace;
        this.texture.flipY = false;
        this.texture.magFilter = THREE.NearestFilter;

        this.texture.minFilter = THREE.LinearFilter;
        this.texture.generateMipmaps = false;
        this.atlasMips = new AtlasMipmaps(size);
        for (const material of Object.values(this.materials)) {
          (material as THREE.MeshStandardMaterial).map = this.texture;
          material.needsUpdate = true;
          this.atlasMips.install(material);
          if (material === this.materials.mask)
            installFoliageLighting(material);
          this.atmosphere.installCloudShadows(material);
          if (material instanceof THREE.MeshStandardMaterial) {
            installEmission(material, this.lighting.emission);
            installBlockLight(material, this.lighting.surfacePower);
          }
        }
        this.water.install();
      }
      for (const patch of data.patches) {
        this.analysisRevision++;
        for (let y = 0; y < patch.size; y++)
          this.pixels!.set(
            patch.pixels.subarray(y * patch.size * 4, (y + 1) * patch.size * 4),
            ((patch.y + y) * page.atlasSize + patch.x) * 4,
          );
      }
      this.texture.needsUpdate = true;
      this.atlasMips?.update(data.patches);
      await this.queue.append(data.chunks);
      if (this.disposed) return;
      loaded += page.chunks.length;
      this.callbacks.current.onProgress(
        loaded,
        page.total,
        this.ledger.snapshot().triangles,
      );
      if (loaded === page.chunks.length) {
        this.updateSun();
        this.fit();
      }
      request.target = page.sequence;
      request.scope = page.scope;
      request.cursor = page.next;
      request.atlasRevision = page.atlasRevision;
	  this.renderedAtlasRevision=page.atlasRevision;
      if (page.complete) {
		this.fullDetail=true;
        await this.finishLoading();
		if(previousBounds){this.bounds.copy(previousBounds);if(previousView)this.restoreView(previousView);this.updateSun()}
        return;
      }
    }
  }
  async finishLoading() {
    if (this.disposed) return;
    const generation=++this.clearanceGeneration;
    this.clearanceTask=buildCloudClearance(
      this.terrain,
      () => this.disposed||generation!==this.clearanceGeneration,
    ).then(clearance=>{
      if(this.disposed||generation!==this.clearanceGeneration){clearance?.texture.dispose();return}
      this.atmosphere.setClearance(clearance);this.dirty=true;
    });
    // Prepare secondary cloud lighting without holding the editor's ready state.
    void this.clearanceTask.catch(error=>{if(!this.disposed)this.callbacks.current.onError(String(error))});
    this.loading = false;
    this.apply(this.callbacks.current.settings, this.callbacks.current.skyFile);
    this.light.castShadow = true;
    this.fit();
    this.setMapView();
    if (
      this.callbacks.current.initialView &&
      !this.callbacks.current.initialMapFocus
    )
      this.restoreView(this.callbacks.current.initialView);
    if (this.callbacks.current.initialMapFocus) {
      const point = new THREE.Vector3(
        ...this.callbacks.current.initialMapFocus,
      ).sub(this.origin ?? new THREE.Vector3());
      this.controls.target.copy(point);
      this.camera.position.copy(point).add(new THREE.Vector3(22, 18, 22));
      this.camera.up.set(0, 1, 0);
      this.camera.lookAt(point);
      this.controls.update();
      this.camera.updateProjectionMatrix();
    }
    await this.skyTask;
    if (this.disposed) return;
    this.initializing = false;
    this.dirty = true;
    this.callbacks.current.onReady();
  }
  scheduleViewChange() {
    if (
      this.loading ||
      this.initializing ||
      this.disposed ||
      this.exporting ||
      this.callbacks.current.shot?.preview
    )
      return;
    this.callbacks.current.onViewDirty?.();
    clearTimeout(this.viewTimer);
    this.viewTimer = setTimeout(() => {
      if (!this.disposed)
        this.callbacks.current.onViewChange?.({
          ...this.snapshot(),
          zoom: this.camera.zoom,
        });
    }, 150);
  }
  setMapView() {
    if (this.loading) return;
    const map = this.callbacks.current.mapView;
    if (map?.enabled && !this.mapCamera) this.clearOrbitMomentum();
    if (map?.enabled && this.camera instanceof THREE.PerspectiveCamera) {
      if (!this.mapCamera)
        this.mapCamera = {
          camera: this.camera.clone(),
          target: this.controls.target.clone(),
          span: this.span,
        };
      const next = new THREE.OrthographicCamera(
        -1,
        1,
        1,
        -1,
        0.05,
        Math.max(50000, this.camera.far),
      );
      next.position.copy(this.camera.position);
      next.quaternion.copy(this.camera.quaternion);
      this.swapNavigationCamera(next);
    }
    if (map?.enabled && this.camera instanceof THREE.OrthographicCamera) {
      if (!this.mapCamera)
        this.mapCamera = {
          camera: this.camera.clone(),
          target: this.controls.target.clone(),
          span: this.span,
        };
      this.controls.enableDamping = false;
      const center = this.bounds.isEmpty()
          ? this.controls.target.clone()
          : this.bounds.getCenter(new THREE.Vector3()),
        radius = Math.max(
          10,
          this.bounds.getBoundingSphere(new THREE.Sphere()).radius * 3,
        );
      this.controls.enableRotate = false;
      this.controls.target.copy(center);
      this.controls.minZoom = 0.1;
      this.controls.maxZoom = 20;
      this.controls.minPolarAngle = 0;
      this.controls.maxPolarAngle = Math.PI;
      this.controls.mouseButtons.LEFT = THREE.MOUSE.PAN;
      this.controls.touches.ONE = THREE.TOUCH.PAN;
      this.camera.position.copy(center).add(new THREE.Vector3(0, radius, 0));
      const angle = (map.rotation * Math.PI) / 180;
      this.camera.up.set(Math.sin(angle), 0, -Math.cos(angle));
      if (this.appliedMap?.zoom !== map.zoom) this.camera.zoom = map.zoom;
      this.appliedMap = { rotation: map.rotation, zoom: map.zoom };
      this.span = this.mapSpan();
      this.resize();
      this.camera.lookAt(center);
      this.controls.update();
      this.camera.updateProjectionMatrix();
      this.dirty = true;
    } else if (
      this.mapCamera &&
      this.camera instanceof THREE.OrthographicCamera
    ) {
      this.swapNavigationCamera(this.mapCamera.camera);
      this.controls.target.copy(this.mapCamera.target);
      this.span = this.mapCamera.span;
      this.mapCamera = undefined;
      this.appliedMap = undefined;
      this.controls.enableRotate = true;
      this.controls.minZoom = 0;
      this.controls.maxZoom = Infinity;
      [this.controls.minPolarAngle, this.controls.maxPolarAngle] = polarLimits(
        this.options.pitchLocked,
        Math.acos(
          THREE.MathUtils.clamp(
            this.camera.position.clone().sub(this.controls.target).normalize()
              .y,
            -1,
            1,
          ),
        ),
      );
      this.controls.enableDamping = true;
      this.controls.mouseButtons.LEFT = THREE.MOUSE.ROTATE;
      this.controls.touches.ONE = THREE.TOUCH.ROTATE;
      this.controls.update();
      this.camera.updateProjectionMatrix();
      this.resize();
      this.dirty = true;
    }
  }
  mapSpan() {
    const size = this.bounds.getSize(new THREE.Vector3()),
      r = ((this.callbacks.current.mapView?.rotation ?? 0) * Math.PI) / 180;
    return (
      Math.max(
        2,
        (Math.abs(Math.cos(r)) * size.x + Math.abs(Math.sin(r)) * size.z) /
          this.options.aspect,
        Math.abs(Math.cos(r)) * size.z + Math.abs(Math.sin(r)) * size.x,
      ) * 1.15
    );
  }
  restoreView(pose: SavedCameraView) {
    if (pose.span !== undefined && Number.isFinite(pose.span) && pose.span > 0)
      this.span = pose.span;
    if (
      !this.callbacks.current.mapView?.enabled &&
      !this.callbacks.current.cinematic
    ) {
      const perspective = pose.projection === "perspective";
      if (perspective !== this.camera instanceof THREE.PerspectiveCamera) {
        this.swapNavigationCamera(
          perspective
            ? new THREE.PerspectiveCamera(pose.fov, 1, 0.05, 50000)
            : new THREE.OrthographicCamera(-1, 1, 1, -1, 0.05, 50000),
        );
      }
    }
    const center = this.bounds.getCenter(new THREE.Vector3()),
      scale = Math.max(
        2,
        this.bounds.getBoundingSphere(new THREE.Sphere()).radius,
      );
    this.camera.position
      .fromArray(pose.position)
      .multiplyScalar(scale)
      .add(center);
    this.controls.target
      .fromArray(pose.target)
      .multiplyScalar(scale)
      .add(center);
    if (!this.callbacks.current.mapView?.enabled) this.camera.up.set(0, 1, 0);
    if (this.camera instanceof THREE.PerspectiveCamera)
      this.camera.fov = pose.fov;
    this.camera.zoom = pose.zoom ?? 1;
    this.camera.far = Math.max(
      this.camera.far,
      this.camera.position.distanceTo(center) + scale * 4,
    );
    this.camera.lookAt(this.controls.target);
    [this.controls.minPolarAngle, this.controls.maxPolarAngle] = polarLimits(
      this.options.pitchLocked,
      Math.acos(
        THREE.MathUtils.clamp(
          this.camera.position.clone().sub(this.controls.target).normalize().y,
          -1,
          1,
        ),
      ),
    );
    this.clearOrbitMomentum();
    this.controls.update();
    this.camera.updateProjectionMatrix();
    this.resize();
    this.dirty = true;
  }
  drawCapture(previewQuality = false) {
    const shot = this.callbacks.current.shot;
    if (shot) {
      const samples = temporalSamples(
        shot.sequence,
        shot.time ?? 0,
        shot.sequence.fps,
        previewQuality,
      ).map((sample) => ({
        ...sample,
        pose: sampleSequence(shot.sequence, sample.time),
      }));
      if (samples.length > 1) this.drawTemporal(samples, shot.pose, shot.time);
      else this.draw(shot.pose, shot.time);
    } else this.draw();
  }
  preview() {
    if (this.loading || this.disposed || this.exporting) return undefined;
    if (this.previewPerformance) this.updatePreviewResolution(true);
    const saved = this.camera.clone(),
      target = this.controls.target.clone(),
      enabled = this.controls.enabled,
      pathVisible = this.path.visible,
      exposure = this.renderer.toneMappingExposure;
    try {
      this.exporting = true;
      this.drawCapture(true);
      return this.renderer.domElement.toDataURL("image/png");
    } finally {
      this.exporting = false;
      this.camera.position.copy(saved.position);
      this.camera.quaternion.copy(saved.quaternion);
      this.camera.up.copy(saved.up);
      if (
        this.camera instanceof THREE.PerspectiveCamera &&
        saved instanceof THREE.PerspectiveCamera
      )
        this.camera.fov = saved.fov;
      this.camera.updateProjectionMatrix();
      this.controls.target.copy(target);
      this.controls.enabled = enabled;
      this.path.visible = pathVisible;
      this.renderer.toneMappingExposure = exposure;
      this.restoreEditingDetail();
      this.draw();
    }
  }
  async capture(signal?:AbortSignal,progress?:(done:number,total:number)=>void) {
    signal?.throwIfAborted();
    if (this.disposed || this.exporting)
      throw new Error("Render view is not ready");
	await this.ensureFullDetail();
    await this.skyTask;
    if (this.disposed) throw new Error("Render view closed");
    const gl = this.renderer.getContext();
    const max = Math.min(
      4096,
      this.renderer.capabilities.maxTextureSize,
      gl.getParameter(gl.MAX_RENDERBUFFER_SIZE) as number,
    );
    const { width, height } = renderDimensions(
      this.options.edge,
      this.options.aspect,
      max,
    );
    this.exporting = true;
    this.renderer.setPixelRatio(1);
    this.composer.setPixelRatio(1);
	this.configureViewportPipeline();
    try {
      await new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      );
      if (this.disposed) throw new Error("Render view closed");
      signal?.throwIfAborted();
      this.size(width, height);
      this.ao.enabled = this.options.ao > 0 && !this.options.transparent;
      this.normals.enabled = this.ao.enabled;
      this.renderer.shadowMap.needsUpdate = true;

      const shot = this.callbacks.current.shot;
      const shutterSamples = shot
        ? temporalSamples(
            shot.sequence,
            shot.time ?? 0,
            shot.sequence.fps,
            false,
          ).length
        : 1;
      if (shutterSamples > 1 || this.options.exportSamples === 1) {
        this.drawCapture();
        signal?.throwIfAborted();progress?.(1,1);
      } else {
        await this.captureSpatial(width, height,signal,progress);
      }
      return this.renderer.domElement.toDataURL("image/png");
    } finally {
      this.finishExport();
    }
  }
  async captureSpatial(width: number, height: number,signal?:AbortSignal,progress?:(done:number,total:number)=>void) {
    this.temporal ??= new TemporalAccumulator(this.renderer, this.mapOverlay);
    const screen = this.composer.renderToScreen;
    const passes = [this.bloom, this.output, this.grade, this.mapOverlay];
    const enabled = passes.map((p) => p.enabled);
    const shot = this.callbacks.current.shot;
    const time =
      shot?.time ?? (performance.now() - this.atmosphereStart) / 1000;
    const samples = spatialSamples(this.options.exportSamples);
    try {
      this.temporal.begin(width, height);
      this.composer.renderToScreen = false;
      passes.forEach((p) => (p.enabled = false));
      for (const offset of samples) {
        signal?.throwIfAborted();
        if (this.disposed) throw new Error("Render view closed");
        withCameraJitter(this.camera, width, height, offset, () => {
          this.draw(shot?.pose, time, true);
          this.temporal!.add(
            this.composer.readBuffer.texture,
            1 / samples.length,
            1,
          );
        });
        progress?.(samples.indexOf(offset)+1,samples.length);
        await new Promise<void>((resolve) => setTimeout(resolve, 0));
      }

      if (enabled[3]) this.draw(shot?.pose, time, true);
      this.mapOverlay.enabled = enabled[3];
      this.temporal.finish(this.options);
    } finally {
      this.composer.renderToScreen = screen;
      passes.forEach((p, i) => (p.enabled = enabled[i]));
    }
  }
  async capturePanorama(
    width: number,
    progress?: (value: number) => void,
    cancelled?: () => boolean,
  ) {
    if (
      this.loading ||
      this.disposed ||
      this.exporting ||
      this.callbacks.current.shot ||
      this.callbacks.current.mapView?.enabled
    )
      throw new Error("Panorama export requires a ready 3D render view.");
    if (![1024, 2048, 4096].includes(width))
      throw new Error("Panorama width must be 1024, 2048 or 4096.");
	await this.ensureFullDetail();
    await this.skyTask;
    if (this.disposed) throw new Error("Render view closed.");
    const original = this.camera,
      options = this.options;
    const controlsEnabled = this.controls.enabled;
    const screen = this.composer.renderToScreen,
      bloom = this.bloom.enabled,
      output = this.output.enabled,
      grade = this.grade.enabled;
    const aoEnabled = this.ao.enabled,
      target = this.controls.target.clone();
    const perspectiveDefine = this.ao.gtaoMaterial.defines.PERSPECTIVE_CAMERA;
    const camera = new THREE.PerspectiveCamera(
      92,
      1,
      original.near,
      original.far,
    );
    camera.position.copy(original.position);
    const orientation = original.quaternion.clone();
    const faceSize = Math.ceil((width / 4) * Math.tan((46 * Math.PI) / 180));
    const faces: PixelImage[] = [];
    this.exporting = true;
    this.controls.enabled = false;
    try {
      this.camera = camera;
      this.beauty.camera = camera;
      this.normals.setCamera(camera);
      this.dofDepth.setCamera(camera);
      this.ao.camera = camera;
      this.ao.gtaoMaterial.defines.PERSPECTIVE_CAMERA = 1;
      this.ao.gtaoMaterial.needsUpdate = true;

      this.options = { ...options, dofEnabled: false, vignette: 0 };
	  this.configureViewportPipeline();
      this.grade.configure(this.options);
      this.composer.renderToScreen = false;
      this.bloom.enabled = false;
      this.output.enabled = false;
      this.grade.enabled = false;
      this.renderer.setPixelRatio(1);
      this.composer.setPixelRatio(1);
      this.size(faceSize, faceSize);
      this.ao.enabled = options.ao > 0 && !options.transparent;
      for (let i = 0; i < cubeFaces.length; i++) {
        if (this.disposed || cancelled?.())
          throw new Error("Panorama export canceled.");
        const face = cubeFaces[i];
        camera.up.fromArray(face.up).applyQuaternion(orientation);
        camera.lookAt(
          camera.position
            .clone()
            .add(
              new THREE.Vector3()
                .fromArray(face.direction)
                .applyQuaternion(orientation),
            ),
        );
        camera.updateMatrixWorld(true);
        this.renderer.shadowMap.needsUpdate = true;
        this.draw(undefined, 0, true);
        const encoded = new Uint16Array(faceSize * faceSize * 4);
        this.renderer.readRenderTargetPixels(
          this.composer.readBuffer,
          0,
          0,
          faceSize,
          faceSize,
          encoded,
        );
        faces.push(decodeHDRFace(encoded, faceSize));
        progress?.((i + 1) / 12);
        await new Promise<void>((resolve) => setTimeout(resolve, 0));
      }
      const result = await stitchPanorama(
        faces,
        width,
        (p) => progress?.(0.5 + p * 0.5),
        () => this.disposed || !!cancelled?.(),
      );
      return renderPanoramaPost(this.renderer, result, options);
    } finally {
      this.camera = original;
      this.beauty.camera = original;
      this.normals.setCamera(original);
      this.dofDepth.setCamera(original);
      this.ao.camera = original;
      this.ao.gtaoMaterial.defines.PERSPECTIVE_CAMERA = perspectiveDefine;
      this.ao.gtaoMaterial.needsUpdate = true;
      this.options = options;
      this.grade.configure(options);
      this.ao.enabled = aoEnabled;
      this.composer.renderToScreen = screen;
      this.bloom.enabled = bloom;
      this.output.enabled = output;
      this.grade.enabled = grade;
      this.controls.target.copy(target);
      this.controls.enabled = controlsEnabled;
      this.finishExport();
    }
  }
  async renderFrame(
    pose: CameraPose,
    width: number,
    height: number,
    time?: number,
    samples?: MotionFrameSample[],
    signal?: AbortSignal,
  ) {
    await this.ensureFullDetail();
    await this.skyTask;
    signal?.throwIfAborted();
    if (this.disposed || this.loading) throw new Error("World is not ready");
    this.exporting = true;
    if (this.renderer.getPixelRatio() !== 1) {
      this.renderer.setPixelRatio(1);
      this.composer.setPixelRatio(1);
    }
	this.configureViewportPipeline();
    this.size(width, height);
    this.ao.enabled = this.options.ao > 0 && !this.options.transparent;
    this.normals.enabled = this.ao.enabled;
    if (samples && samples.length > 1)
      await renderTemporalSteps(
        this.temporalSteps(samples, pose, time),
        signal,
      );
    else this.draw(pose, time);
    return this.renderer.domElement;
  }
  finishExport() {
    this.exporting = false;
    if (this.disposed) return;
    if (
      this.callbacks.current.flight &&
      this.flightActive !== this.callbacks.current.flight.enabled
    )
      this.configureFlight();
    const ratio = Math.min(window.devicePixelRatio || 1, 2);
    this.renderer.setPixelRatio(ratio);
    this.composer.setPixelRatio(ratio);
    this.resize();
    this.dirty = true;
	this.restoreEditingDetail();
  }
	restoreEditingDetail(){
	 this.previewAA.reset();this.pipelineFull=undefined;this.dirty=true;
	}
	async ensureFullDetail(){
	 if(this.detailTask)await this.detailTask;
	 if(this.fullDetail){await this.clearanceTask;return;}
	 if(this.loading)throw new Error("Wait for the viewport to finish loading");
	 this.detailTask=this.load().catch(error=>{this.loading=false;throw error}).finally(()=>{this.detailTask=undefined});
	 await this.detailTask;
	}
	configureViewportPipeline(){
	 const finalQuality=this.exporting||this.callbacks.current.viewportQuality==="full";
	 const full=finalQuality||this.quality;
	 this.previewAA.enabled=!this.exporting&&this.callbacks.current.viewportQuality!=="full";
	 this.bloom.enabled=full&&this.options.bloom>0&&!this.options.transparent;
	 this.ao.enabled=full&&(this.exporting||this.quality)&&this.options.ao>0&&!this.options.transparent;
	 if(this.pipelineFull===full&&this.pipelineFinal===finalQuality)return;
	 this.pipelineFull=full;this.pipelineFinal=finalQuality;
	 this.renderer.shadowMap.enabled=true;
	 this.renderer.shadowMap.needsUpdate=true;
	 const quality=renderQuality[full?this.options.quality:"fast"];
	 this.ao.updateGtaoMaterial({radius:this.options.aoRadius,samples:quality.samples});this.ao.updatePdMaterial({radius:6,rings:2,samples:quality.denoise});
	 for(const target of [this.composer.renderTarget1,this.composer.renderTarget2]){const samples=finalQuality?Math.min(quality.aa,this.renderer.capabilities.maxSamples):0;if(target.samples!==samples){target.samples=samples;target.dispose()}}
	 const resolution=Math.min(finalQuality?this.options.shadow:1024,this.renderer.capabilities.maxTextureSize);
	 if(this.light.shadow.mapSize.x!==resolution){this.light.shadow.mapSize.set(resolution,resolution);this.light.shadow.map?.dispose();this.light.shadow.map=null;this.renderer.shadowMap.needsUpdate=true}
	}
  dispose() {
    this.previewAA.dispose();
    this.temporal?.dispose();
    for (const marker of this.targetMarkers.values()) {
      marker.geometry.dispose();
      marker.material.dispose();
    }
    this.targetMarkers.clear();
    this.lightTransform.dispose();
    this.cloudGizmos.dispose();
    this.selectedLightGuide?.dispose();
    for (const marker of this.lightMarkers.values()) {
      marker.geometry.dispose();
      (marker.material as THREE.Material).dispose();
    }
    this.lightSightline.geometry.dispose();
    this.lightSightline.material.dispose();
    this.studioLights.dispose();
    clearTimeout(this.viewTimer);
    this.lighting.dispose();
    this.disposed = true;
    cancelAnimationFrame(this.frame);
    this.resizeObserver.disconnect();
    this.decoder.dispose();
    this.queue.cancel();
    this.controls.dispose();
    this.freeCamera.dispose();
    this.focusGuides.dispose();
    this.terrain.traverse((item) => {
      if (item instanceof THREE.Mesh) item.geometry.dispose();
    });
    Object.values(this.materials).forEach((material) => material.dispose());
    this.texture?.dispose();
    this.atlasMips?.dispose();
    this.skyGeneration++;
    this.sky?.background.dispose();
    this.sky?.lighting.dispose();
    this.environmentMap?.dispose();
    this.pmrem.dispose();
    this.normals.dispose();
    this.dofDepth.dispose();
    this.path.traverse((item) => {
      if (item instanceof THREE.Mesh || item instanceof THREE.Line) {
        item.geometry.dispose();
        (item.material as THREE.Material).dispose();
      }
    });
    this.bloom.dispose();
    this.dof.dispose();
    this.skyBackground.dispose();
    this.ao.dispose();
    this.output.dispose();
    this.grade.dispose();
    this.atmosphere.dispose();
    this.mapOverlay.dispose();
    this.composer.dispose();
    this.light.shadow.map?.dispose();
    this.renderer.domElement.removeEventListener("webglcontextlost", this.lost);
    this.renderer.dispose();
    this.renderer.forceContextLoss();
    this.renderer.domElement.remove();
  }
}

export default forwardRef<RenderCanvasHandle, Props>(
  function RenderCanvas(props, ref) {
    const scope = useDocumentScope();
    const awake = scope.active || !!scope.busy;
    const viewKey = props.worldId + JSON.stringify(props.selectionBounds);
    const cached = useRef<{ key: string; view: SavedCameraView } | undefined>(
      undefined,
    );
    const host = useRef<HTMLDivElement>(null),
      engine = useRef<RenderEngine | null>(null),
      callbacks = useRef(props);
    callbacks.current = {
      ...props,
      initialView:
        cached.current?.key === viewKey
          ? cached.current.view
          : props.initialView,
    };
    useImperativeHandle(
      ref,
      () => ({
		 capturePose:(point)=>engine.current?.capturePose(point),
        analyse: (progress, signal) =>
          engine.current?.analyse(progress, signal) ??
          Promise.reject(new Error("Renderer unavailable")),
        capture: () => runExportTask("Render image",task=>{
          if(!engine.current)throw new Error("Render view unavailable");
          task.update("Rendering samples…");return engine.current.capture(task.signal,(done,total)=>task.update("Rendering samples…",done,total));
        },{cancelable:true}),
        capturePanorama: (width, progress, cancelled) => runExportTask("Render panorama",task=>{
          if(!engine.current)throw new Error("Render view unavailable");
          return engine.current.capturePanorama(width,p=>{progress?.(p);task.update("Rendering panorama…",p*100,100)},()=>task.signal.aborted||!!cancelled?.());
        },{cancelable:true}),
        fit: () => engine.current?.fit(),
        flightPosition: () => engine.current?.flightPosition() ?? [0, 0, 0],
        teleport: (position) => engine.current?.teleport(position),
        flightReturn: () => engine.current?.flightReturn(),
        beginFlight: async (flight) => {
          const e = engine.current;
          if (!e || e.loading || e.exporting) return false;
          e.callbacks.current.flight = flight;
          e.configureFlight();
          return e.freeCamera.lockPointer();
        },
        lockMouse: () =>
          engine.current?.freeCamera.lockPointer() ?? Promise.resolve(false),
        view: (angle, tilt) => engine.current?.view(angle, tilt),
        snapshot: () =>
          engine.current && !engine.current.loading
            ? engine.current.snapshot()
            : cached.current?.key === viewKey
              ? cached.current.view
              : callbacks.current.initialView,
        preview: () => engine.current?.preview(),
        renderFrame: (pose, width, height, time, samples, signal) =>
          engine.current?.renderFrame(
            pose,
            width,
            height,
            time,
            samples,
            signal,
          ) ?? Promise.reject(new Error("Renderer unavailable")),
        finishExport: () => engine.current?.finishExport(),
      }),
      [viewKey],
    );
    useEffect(() => {
      if (!awake) return;
      try {
        engine.current = new RenderEngine(host.current!, callbacks);
      } catch (error) {
        props.onError(error instanceof Error ? error.message : String(error));
      }
      return () => {
        if (engine.current && !engine.current.loading)
          cached.current = { key: viewKey, view: engine.current.snapshot() };
        engine.current?.dispose();
        engine.current = null;
      };
    }, [
      awake,
      props.worldId,
      props.selectionBounds?.minX,
      props.selectionBounds?.maxX,
      props.selectionBounds?.minZ,
      props.selectionBounds?.maxZ,
    ]);
    useEffect(
      () => engine.current?.apply(props.settings, props.skyFile),
      [props.settings, props.skyFile],
    );
	useEffect(()=>{const e=engine.current;if(!e)return;e.pipelineFull=undefined;e.dirty=true;if(props.viewportQuality==="full"&&!e.loading){void e.ensureFullDetail().catch(error=>{if(!e.disposed)props.onError(String(error))})}else e.restoreEditingDetail()},[props.viewportQuality]);
    useEffect(() => {
      if (engine.current) engine.current.dirty = true;
    }, [props.shot]);
    useEffect(() => {
      if (engine.current && props.shot) {
        engine.current.settle = performance.now() + 180;
        engine.current.dirty = true;
      }
    }, [props.shot?.time, props.shot?.preview, props.shot?.playing]);
    useEffect(() => engine.current?.updatePath(), [props.shot?.sequence]);
    useEffect(() => engine.current?.setMapView(), [props.mapView]);
    useEffect(() => {
      if (props.active !== false && engine.current) {
        engine.current.ensureLoaded();
        engine.current.resize();
        engine.current.dirty = true;
      }
    }, [props.active]);
    useEffect(
      () => engine.current?.configureFlight(),
      [props.flight?.enabled, props.flight?.fov],
    );
    useEffect(
      () => engine.current?.configureFocusPicking(),
      [props.pickingFocus],
    );
    useEffect(() => {
      if (engine.current) engine.current.dirty = true;
    }, [props.focusGuides]);
	useEffect(()=>{const e=engine.current;if(!e||e.loading)return;if(props.pickingResource||props.pickingFocus){void e.ensureFullDetail().catch(error=>{if(!e.disposed)props.onError(String(error))})}else e.restoreEditingDetail()},[props.pickingResource,props.pickingFocus]);
    useEffect(() => {
      if (engine.current) engine.current.dirty = true;
    }, [props.lightEditor]);
    useEffect(() => {
      if (engine.current) engine.current.dirty = true;
    }, [props.cloudEditor]);
    return (
      <div
        className="render-canvas"
        ref={host}
        style={props.active === false ? { display: "none" } : undefined}
      />
    );
  },
);
