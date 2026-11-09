import FeedbackFlyout from "./FeedbackFlyout";
import {ModelExportDialog} from "./ExportStudio";

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import {
  Crosshair,
  Download,
  LoaderCircle,
  Scan,
  Orbit,
} from "lucide-react";
import { WorldIcon } from "./studio-icons";
import { Button } from "@/components/ui/button";
import { api, type ScenePageRequest } from "./bridge";
import { SceneLedger, sceneStats } from "./scene-state";
import { SceneDecoder } from "./scene-decoder";
import { FrameQueue } from "./scene-queue";
import type { DecodedChunk } from "./scene-wire";
import { modelFrame } from "./scene-camera";
import {PreviewPerformance} from "./preview-performance";
import { billboardAttributes } from "./scene-billboard";
import { foliageAttributes, installFoliageLighting } from "./foliage-lighting";
import {
  emissionAttributes,
  installEmission,
  installBlockLight,
} from "./render-lighting";

type Engine = {
  retry: () => void;
  fit: () => void;
  player: (position: [number, number, number]) => void;
  setCentered: (enabled: boolean) => void;
};
const count = new Intl.NumberFormat("en-US");

export default function WorldScene({
  worldId,
  name,
  player,
  hasPlayer,
  initialCentered = false,
  onCenteredChange,
  initialFocus,
}: {
  worldId: string;
  name: string;
  player: [number, number, number];
  hasPlayer: boolean;
  initialCentered?: boolean;
  onCenteredChange?: (enabled: boolean) => void;
  initialFocus?: [number, number, number];
}) {
  const host = useRef<HTMLDivElement>(null),
    engine = useRef<Engine | null>(null);
  const playerRef = useRef(player);
  const centeredRef = useRef(initialCentered);
  const [centered, setCentered] = useState(initialCentered);
  playerRef.current = player;
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState(sceneStats(new Map()));
  const [error, setError] = useState(""),
    [source, setSource] = useState("");
  const [loading, setLoading] = useState(true),
    [exporting, setExporting] = useState(false);

  useEffect(() => {
    const element = host.current;
    if (!element || !worldId) {
      setLoading(false);
      return;
    }
    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({
        antialias: true,
        powerPreference: "high-performance",
      });
    } catch {
      setError(
        "The 3D view requires WebGL. Check hardware acceleration or use the 2D map.",
      );
      setLoading(false);
      return;
    }
    const scene = new THREE.Scene();
    let needsRender = true;
    const applyTheme = () => {
      scene.background = new THREE.Color(
        getComputedStyle(element).getPropertyValue("--scene-sky").trim() ||
          "#18242c",
      );
      needsRender = true;
    };
    applyTheme();
    const themeObserver = new MutationObserver(applyTheme);
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "data-theme"],
    });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    element.append(renderer.domElement);
    renderer.domElement.tabIndex = 0;
    renderer.domElement.setAttribute(
      "aria-label",
      "3D world. Drag to orbit, right-drag to pan, and scroll to zoom. Use WASD and Q/E to move the camera.",
    );
    const camera = new THREE.PerspectiveCamera(55, 1, 0.1, 100000);
    camera.position.set(40, 100, 60);
    const controls = new OrbitControls(camera, renderer.domElement);
    let centerLocked = centeredRef.current;
    let lockedCenter: THREE.Vector3 | undefined;
    controls.enablePan = !centerLocked;
    controls.enableDamping = !window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    controls.maxDistance = 30000;
    const hemisphere = new THREE.HemisphereLight(0xffffff, 0x9ba8b1, 2);
    const sun = new THREE.DirectionalLight(0xffffff, 2.4);
    sun.position.set(-70, 140, 90);
    scene.add(hemisphere, sun);
    const terrain = new THREE.Group();
    scene.add(terrain);
    let texture: THREE.DataTexture | undefined;
    let atlasPixels: Uint8Array<ArrayBuffer> | undefined;
    const ledger = new SceneLedger();
    const decoder = new SceneDecoder();
    let pageRequest: ScenePageRequest = {
      previewStep:0,
      after: 0,
      target: 0,
      cursor: null,
      focus: [
        Math.floor((initialFocus ?? playerRef.current)[0] / 16),
        Math.floor((initialFocus ?? playerRef.current)[2] / 16),
      ],
      limit: 8,
      atlasRevision: 0,
      scope: 0,
    };
    const groups = new Map<string, THREE.Group>();
    const materials = {
      water: new THREE.MeshStandardMaterial({
        roughness: 0.25,
        transparent: true,
        opacity: 0.85,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
      "unlit-blend": new THREE.MeshBasicMaterial({
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
      billboard: new THREE.MeshBasicMaterial({
        alphaTest: 0.05,
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
      unlit: new THREE.MeshBasicMaterial({
        alphaTest: 0.05,
        side: THREE.DoubleSide,
      }),
      opaque: new THREE.MeshStandardMaterial({ roughness: 1 }),
      mask: new THREE.MeshStandardMaterial({
        roughness: 1,
        alphaTest: 0.5,
        side: THREE.DoubleSide,
      }),
      blend: new THREE.MeshStandardMaterial({
        roughness: 0.65,
        transparent: true,
        opacity: 0.72,
        depthWrite: false,
        side: THREE.DoubleSide,
      }),
    };
    installFoliageLighting(materials.mask);
    for (const material of Object.values(materials)) {
      if (material instanceof THREE.MeshStandardMaterial) {
        installEmission(material, { value: 1 });
        installBlockLight(material, { value: 1 });
      }
    }
    materials.billboard.onBeforeCompile = (shader) => {
      shader.vertexShader =
        "attribute vec3 billboardCenter;\nattribute vec2 billboardOffset;\n" +
        shader.vertexShader;
      shader.vertexShader = shader.vertexShader.replace(
        "#include <project_vertex>",
        `
        vec4 mvPosition = modelViewMatrix * vec4(billboardCenter, 1.0);
        mvPosition.xy += billboardOffset;
        gl_Position = projectionMatrix * mvPosition;
      `,
      );
    };
    materials.billboard.customProgramCacheKey = () => "craftbox-hologram-v1";
    let disposed = false,
      timer: ReturnType<typeof setTimeout>,
      frame = 0,
      fitted = false;
    let origin: THREE.Vector3 | undefined;
    let initialCycleComplete = false,
      userNavigated = false;
    let boundsChanged = false;
    const navigationStarted = () => {
      userNavigated = true;
    };
    controls.addEventListener("start", navigationStarted);
    const keys = new Set<string>();
    const clearGroup = (group: THREE.Group) => {
      group.traverse((object) => {
        if (object instanceof THREE.Mesh) object.geometry.dispose();
      });
      terrain.remove(group);
    };
    const fit = (preserveDirection = false) => {
      needsRender = true;
      const bounds = new THREE.Box3().setFromObject(terrain);
      const framing = modelFrame(bounds, camera.fov, camera.aspect);
      if (!framing) return;
      const { center, distance, far } = framing;
      boundsChanged = false;
      const direction = preserveDirection
        ? camera.position.clone().sub(controls.target)
        : new THREE.Vector3(1, 0.9, 1);
      if (!direction.lengthSq()) direction.set(1, 0.9, 1);
      lockedCenter = centerLocked ? center.clone() : undefined;
      controls.target.copy(center);
      camera.position
        .copy(center)
        .add(direction.normalize().multiplyScalar(distance));
      controls.maxDistance = Math.max(30000, distance * 10);
      camera.far = Math.max(far, controls.maxDistance + framing.radius * 2);
      camera.updateProjectionMatrix();
      controls.update();
    };
    engine.current = {
      retry: () => {},
      fit: () => {
        userNavigated = true;
        fit();
      },
      player: (position) => {
        if (centerLocked) return;
        userNavigated = true;
        needsRender = true;
        const center = new THREE.Vector3(...position).sub(
          origin ?? new THREE.Vector3(),
        );
        controls.target.copy(center);
        camera.position.copy(center).add(new THREE.Vector3(22, 18, 22));
        controls.update();
      },
      setCentered: (enabled) => {
        centerLocked = enabled;
        controls.enablePan = !enabled;
        keys.clear();
        if (enabled) fit(true);
        else lockedCenter = undefined;
        renderer.domElement.setAttribute(
          "aria-label",
          enabled
            ? "3D world. Centered orbit: drag to rotate and scroll to zoom. The pivot is fixed at the model center."
            : "3D world. Drag to orbit, right-drag to pan, and scroll to zoom. Use WASD and Q/E to move the camera.",
        );
        needsRender = true;
      },
    };
    engine.current.setCentered(centerLocked);
    const resize = new ResizeObserver(() => {
      const width = element.clientWidth,
        height = element.clientHeight;
      if (!width || !height) return;
      needsRender = true;
      renderer.setSize(width, height);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      if (centerLocked) fit(true);
    });
    resize.observe(element);
    const keydown = (event: KeyboardEvent) => {
      if (centerLocked && event.code.startsWith("Arrow")) {
        event.preventDefault();
        return;
      }
      if (
        [
          "KeyW",
          "KeyA",
          "KeyS",
          "KeyD",
          "KeyQ",
          "KeyE",
          "ShiftLeft",
          "ShiftRight",
        ].includes(event.code)
      ) {
        if (centerLocked) {
          event.preventDefault();
          return;
        }
        keys.add(event.code);
        userNavigated = true;
        event.preventDefault();
      }
    };
    const keyup = (event: KeyboardEvent) => keys.delete(event.code);
    const blur = () => keys.clear();
    renderer.domElement.addEventListener("keydown", keydown);
    renderer.domElement.addEventListener("keyup", keyup);
    renderer.domElement.addEventListener("blur", blur);
    const focus = () => renderer.domElement.focus();
    renderer.domElement.addEventListener("pointerdown", focus);
    const contextLost = (event: Event) => {
      event.preventDefault();
      setError("The graphics context was lost. Switch to 2D, then reopen 3D.");
    };
    renderer.domElement.addEventListener("webglcontextlost", contextLost);
    const commitChunk = (chunk: DecodedChunk) => {
      const key = `${chunk.x},${chunk.z}`,
        old = groups.get(key);
      ledger.update(chunk);
      if (!origin) origin = new THREE.Vector3(chunk.x * 16, 0, chunk.z * 16);
      const group = new THREE.Group();
      group.position.set(chunk.x * 16 - origin.x, 0, chunk.z * 16 - origin.z);
      for (const part of chunk.parts) {
        const geometry = new THREE.BufferGeometry();
        geometry.setAttribute(
          "blockLight",
          new THREE.BufferAttribute(
            part.blockLight ?? new Uint8Array(part.positions.length / 3),
            1,
          ),
        );
        geometry.setAttribute(
          "blockEmission",
          new THREE.BufferAttribute(
            emissionAttributes(
              part.positions,
              part.normals,
              part.indices,
              chunk.lights ?? [],
            ),
            1,
			true,
          ),
        );
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
          "foliage",
          new THREE.BufferAttribute(
            part.mode === "mask"
              ? foliageAttributes(part.positions)
              : new Uint8Array(part.positions.length / 3),
            1,
          ),
        );
        if (part.mode === "billboard") {
          const { centers, offsets } = billboardAttributes(part.positions);
          geometry.setAttribute(
            "billboardCenter",
            new THREE.BufferAttribute(centers, 3),
          );
          geometry.setAttribute(
            "billboardOffset",
            new THREE.BufferAttribute(offsets, 2),
          );
        }
        geometry.setIndex(new THREE.BufferAttribute(part.indices, 1));
        const b = part.bounds;
        geometry.boundingBox = new THREE.Box3(
          new THREE.Vector3(b[0], b[1], b[2]),
          new THREE.Vector3(b[3], b[4], b[5]),
        );
        geometry.boundingSphere = geometry.boundingBox.getBoundingSphere(
          new THREE.Sphere(),
        );
        const mesh = new THREE.Mesh(geometry, materials[part.mode]);

        if (part.mode === "billboard") mesh.frustumCulled = false;
        group.add(mesh);
      }
      if (old) clearGroup(old);
      groups.set(key, group);
      terrain.add(group);
      boundsChanged = true;
      needsRender = true;
      if (!fitted && chunk.parts.length) {
        fit();
        fitted = true;
      }
    };
    const uploads = new FrameQueue<DecodedChunk>(commitChunk);
    const forward = new THREE.Vector3(),
      right = new THREE.Vector3(),
      velocity = new THREE.Vector3(),
      lockDrift = new THREE.Vector3();
    let lastTime = performance.now();
	const previewPerformance=new PreviewPerformance();

    const animate = (now: number) => {
      if (disposed) return;
      if(renderer.domElement.closest("[hidden]")){lastTime=now;frame=requestAnimationFrame(animate);return;}
      const dt = Math.min((now - lastTime) / 1000, 0.1);
      lastTime = now;
      if (keys.size && !centerLocked) {
        camera.getWorldDirection(forward);
        right.crossVectors(forward, camera.up).normalize();
        velocity.set(0, 0, 0);
        needsRender = true;
        if (keys.has("KeyW")) velocity.add(forward);
        if (keys.has("KeyS")) velocity.sub(forward);
        if (keys.has("KeyD")) velocity.add(right);
        if (keys.has("KeyA")) velocity.sub(right);
        if (keys.has("KeyE")) velocity.y += 1;
        if (keys.has("KeyQ")) velocity.y -= 1;
        velocity
          .normalize()
          .multiplyScalar(
            dt * (keys.has("ShiftLeft") || keys.has("ShiftRight") ? 150 : 30),
          );
        camera.position.add(velocity);
        controls.target.add(velocity);
      }
      if (!document.hidden && uploads.flush() > 0) needsRender = true;
      const changed = controls.update();

      if (centerLocked && lockedCenter) {
        lockDrift.copy(lockedCenter).sub(controls.target);
        if (lockDrift.lengthSq() > 0) {
          camera.position.add(lockDrift);
          controls.target.copy(lockedCenter);
          needsRender = true;
        }
      }
      if (!document.hidden && (needsRender || changed)) {
		const ratio=previewPerformance.update(!!changed||keys.size>0,element.clientWidth,element.clientHeight,window.devicePixelRatio,dt*1000);
		if(Math.abs(renderer.getPixelRatio()-ratio)>0.01)renderer.setPixelRatio(ratio);
        renderer.render(scene, camera);
        needsRender = false;
      }
      frame = requestAnimationFrame(animate);
    };
    frame = requestAnimationFrame(animate);

    let busy = false,
      failures = 0;
    const poll = async () => {
      if (disposed || busy) return;
      if (document.hidden || renderer.domElement.closest("[hidden]")) {
        timer = setTimeout(poll, 500);
        return;
      }
      busy = true;
      let delay = 500;
      try {
        const update = await api.GetScenePage(worldId, pageRequest);
        if (disposed) return;
        const decoded = await decoder.decode(update);
        if (disposed) return;
        if (!texture) {
          if (
            !Number.isInteger(update.atlasSize) ||
            update.atlasSize < 1 ||
            update.atlasSize > 4096
          )
            throw new Error("Invalid texture atlas");
          atlasPixels = new Uint8Array(update.atlasSize * update.atlasSize * 4);
          texture = new THREE.DataTexture(
            atlasPixels,
            update.atlasSize,
            update.atlasSize,
            THREE.RGBAFormat,
          );
          texture.colorSpace = THREE.SRGBColorSpace;
          texture.flipY = false;
          texture.magFilter = THREE.NearestFilter;
          texture.minFilter = THREE.NearestFilter;
          texture.generateMipmaps = false;
          for (const material of Object.values(materials)) {
            material.map = texture;
            material.needsUpdate = true;
          }
        }
        for (const patch of decoded.patches) {
          for (let y = 0; y < patch.size; y++)
            atlasPixels!.set(
              patch.pixels.subarray(
                y * patch.size * 4,
                (y + 1) * patch.size * 4,
              ),
              ((patch.y + y) * update.atlasSize + patch.x) * 4,
            );
        }
        if (decoded.patches.length || pageRequest.atlasRevision === 0) {
          texture.needsUpdate = true;
          needsRender = true;
        }
        if (update.reset) {
          for (const group of groups.values()) clearGroup(group);
          groups.clear();
          ledger.clear();
          needsRender = true;
        }
        setTotal(update.total);
        setSource(update.source);
        setLoading(!update.complete);
        setError("");
        failures = 0;
        await uploads.append(decoded.chunks);
        if (disposed) return;
        if (update.complete && centerLocked && boundsChanged) fit(true);
        if (update.complete && !initialCycleComplete) {
          if (!userNavigated && !centerLocked) {
            if (initialFocus) engine.current?.player(initialFocus);
            else fit();
          }
          initialCycleComplete = true;
        }
        setStats(ledger.snapshot());
        pageRequest = {
          ...pageRequest,
          target: update.complete ? 0 : update.sequence,
          cursor: update.next,
          after: update.complete ? update.sequence : pageRequest.after,
          scope: update.scope,
          atlasRevision: update.atlasRevision,
        };
        delay = update.complete ? 500 : 0;
      } catch (reason) {
        if (!disposed) {
          const message = String(
            reason instanceof Error ? reason.message : reason,
          );
          setError(
            /deadline|timed? ?out|timeout/i.test(message)
              ? "Scene loading slowly. Retrying automatically."
              : message,
          );
          setLoading(false);
          failures++;
          delay = Math.min(1000 * 2 ** Math.min(failures - 1, 3), 8000);
        }
      } finally {
        busy = false;
        if (!disposed) timer = setTimeout(poll, delay);
      }
    };
    engine.current.retry = () => {
      if (!busy) {
        clearTimeout(timer);
        void poll();
      }
    };
    void poll();
    return () => {
      disposed = true;
      clearTimeout(timer);
      cancelAnimationFrame(frame);
      resize.disconnect();
      engine.current = null;
      renderer.domElement.removeEventListener("keydown", keydown);
      renderer.domElement.removeEventListener("keyup", keyup);
      renderer.domElement.removeEventListener("blur", blur);
      renderer.domElement.removeEventListener("pointerdown", focus);
      renderer.domElement.removeEventListener("webglcontextlost", contextLost);
      uploads.cancel();
      decoder.dispose();
      themeObserver.disconnect();
      controls.dispose();
      controls.removeEventListener("start", navigationStarted);
      for (const group of groups.values()) clearGroup(group);
      Object.values(materials).forEach((material) => material.dispose());
      texture?.dispose();
      renderer.dispose();
      renderer.domElement.remove();
    };
  }, [worldId]);

  return (
    <section className="scene-viewer" aria-label={`3D world ${name}`}>
      <div ref={host} className="scene-canvas" /><ModelExportDialog open={exporting} onOpenChange={setExporting} worldId={worldId} name={name}/>
      <div className="scene-toolbar">
        <Button
          variant="secondary"
          onClick={() => engine.current?.fit()}
          data-tooltip="Show all terrain"
        >
          <Scan />
          Overview
        </Button>
        <Button
          variant={centered ? "default" : "secondary"}
          aria-pressed={centered}
          data-tooltip="Fix the pivot at the model center; orbit and zoom only"
          onClick={() => {
            const enabled = !centered;
            centeredRef.current = enabled;
            setCentered(enabled);
            onCenteredChange?.(enabled);
            engine.current?.setCentered(enabled);
          }}
        >
          <Orbit />
          Centered orbit
        </Button>
        <Button
          variant="secondary"
          disabled={!hasPlayer || centered}
          onClick={() => engine.current?.player(player)}
        >
          <Crosshair />
          Player
        </Button>
        <Button
          disabled={!stats.triangles}
          onClick={() => setExporting(true)}
        >
          <Download />
          Export 3D
        </Button>
      </div>
      {loading && (
        <div
          className={
            stats.triangles ? "scene-loading-progress" : "scene-message"
          }
          role="status"
        >
          <LoaderCircle className="animate-spin" />
          <span>
            Loading 3D terrain
            {total > 0
              ? ` · ${count.format(stats.chunks)} of ${count.format(total)} Chunks`
              : " …"}
          </span>
        </div>
      )}
      {!loading && !error && !stats.triangles && (
        <div className="scene-message">
          <WorldIcon />
          <p>Captured terrain will appear here in 3D.</p>
        </div>
      )}
      {error && (
        <FeedbackFlyout title="World preview">
          <span>{error}</span>
          <Button variant="outline" onClick={() => engine.current?.retry()}>
            Try again
          </Button>
        </FeedbackFlyout>
      )}

      <div className="scene-caption">
        <span>
          {count.format(stats.chunks)} Chunks · {count.format(stats.triangles)}{" "}
          Triangles
        </span>
        <span>
          {centered
            ? "Pivot fixed at model center · Drag to orbit · Scroll to zoom"
            : "Drag to orbit · Right-drag to pan · Scroll to zoom · Fly: WASD, Q/E, Shift"}
        </span>
        {(stats.missing > 0 || stats.approximate > 0) && (
          <span className="scene-quality">
            {count.format(stats.missing)} Untextured blocks ·{" "}
            {count.format(stats.approximate)} simplified shapes
          </span>
        )}
        <span className="scene-source">{source}</span>
      </div>
    </section>
  );
}
