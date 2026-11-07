import * as THREE from "three";
import { RenderEngine, type RenderCanvasProps } from "../src/RenderCanvas";
import { renderDefaults } from "../src/render-settings";
import { decodePart } from "../src/scene-wire";
import { createSequence } from "../src/cinematic";
import type { ScenePage } from "../src/bridge";

export async function runNavigationCheck(page: ScenePage, progress = (_stage: string) => {}) {
  const reports = [];
  const errors: string[] = [];
  for (const cinematic of [false, true]) {
    const editor = cinematic ? "cinematic" : "render";
    progress(editor+": setup");
    const host = document.createElement("div");
    host.style.cssText = "position:fixed;inset:0;width:480px;height:300px";
    document.body.append(host);
    const settings = {
      ...renderDefaults,
      sky: "none",
      quality: "fine",
      edge: 480,
      aspect: 1.6,
      exportSamples: 4,
      dofEnabled: true,
      dofDistance: 60,
      atmosphere: {
        ...renderDefaults.atmosphere,
        animate: false,
        fog: true,
        clouds: true,
        cloudPreviewQuality: "high",
        cloudExportQuality: "high",
      },
    } as RenderCanvasProps["settings"];
    const callbacks = {
      current: {
        worldId: "navigation-check",
        settings,
        cinematic,
        active: false,
        onError: (s: string) => errors.push(s),
        onReady() {},
        onProgress() {},
        onSkyLoading() {},
      } as RenderCanvasProps,
    };
    const load = RenderEngine.prototype.load;
    RenderEngine.prototype.load = async function () {
      for (const chunk of page.chunks)
        for (const packed of chunk.parts) {
          if (["billboard", "unlit-blend"].includes(packed.mode)) continue;
          const p = decodePart(packed);
          const geometry = new THREE.BufferGeometry();
          geometry.setAttribute(
            "position",
            new THREE.BufferAttribute(p.positions, 3),
          );
          geometry.setAttribute(
            "normal",
            new THREE.BufferAttribute(p.normals, 3, true),
          );
          geometry.setAttribute("uv", new THREE.BufferAttribute(p.uvs, 2));
          geometry.setIndex(new THREE.BufferAttribute(p.indices, 1));
          const mesh = new THREE.Mesh(geometry, this.materials.opaque);
          mesh.position.set(chunk.x * 16, 0, chunk.z * 16);
          mesh.castShadow = mesh.receiveShadow = true;
          this.terrain.add(mesh);
        }
      this.bounds.setFromObject(this.terrain);
      this.loading = false;
      this.apply(settings);
      this.fit();
    };
    let engine: RenderEngine;
    try {
      engine = new RenderEngine(host, callbacks);
    } finally {
      RenderEngine.prototype.load = load;
    }
    try {
      cancelAnimationFrame(engine.frame);
      if (cinematic) {
        callbacks.current.shot = {
          pose: { ...engine.snapshot(), exposure: 1 },
          sequence: createSequence(),
          preview: true,
          showPath: false,
          time: 0,
        };
      }
      const gl = engine.renderer.getContext();
      const measure = (high: boolean) => {
        if (gl.isContextLost()) throw Error("Graphics context was lost");
        engine.quality = high;
        engine.updatePreviewResolution(high);
        engine.ao.enabled = high && settings.ao > 0;
        const start = performance.now();
        engine.draw(undefined, 0, true);
        gl.finish();
        return {
          ms: performance.now() - start,
          ratio: engine.renderer.getPixelRatio(),
          pixels:
            engine.renderer.domElement.width *
            engine.renderer.domElement.height,
          dofSamples: engine.dof.uniforms.samples.value.toArray(),
          cloudSteps: engine.atmosphere.uniforms.steps.value,
          ao: engine.ao.enabled,
        };
      };
                                                                              
      progress(editor+": warm shaders");
      measure(true);
      measure(false);
      measure(true);
      const full = measure(true);
      progress(editor+": baseline export");
      const baselineExport = await engine.capture();
      const moving = measure(false);
      const settled = measure(true);
      progress(editor+": export after navigation");
      const afterExport = await engine.capture();
      if (baselineExport !== afterExport)
        throw Error("Export pixels changed after navigation");
      const bitmap = await createImageBitmap(await (await fetch(baselineExport)).blob());
      const pixels = document.createElement("canvas");
      pixels.width = bitmap.width;
      pixels.height = bitmap.height;
      const context = pixels.getContext("2d")!;
      context.drawImage(bitmap, 0, 0);
      bitmap.close();
      const image = context.getImageData(0, 0, pixels.width, pixels.height).data;
      const colors = new Set<number>();
      for (let i = 0; i < image.length; i += 4*17) colors.add((image[i]<<16)|(image[i+1]<<8)|image[i+2]);
      if (colors.size < 32) throw Error("Export comparison used an empty or uniform image");
      if (moving.pixels >= full.pixels || moving.cloudSteps >= full.cloudSteps)
        throw Error("Movement budget did not reduce work");
      if (
        settled.pixels !== full.pixels ||
        settled.dofSamples.join() !== full.dofSamples.join() ||
        settled.cloudSteps !== full.cloudSteps
      )
        throw Error("Settled preview quality did not restore");
      if (gl.isContextLost() || gl.getError() !== gl.NO_ERROR) throw Error("WebGL error");
      reports.push({
        editor: cinematic ? "cinematic" : "render",
        full,
        moving,
        settled,
        identicalExport: true,
        distinctColors: colors.size,
        exportProofImage: baselineExport,
        exportWidth: 480,
        exportHeight: 300,
      });
    } finally {
      engine.dispose();
      host.remove();
    }
  }
  if (errors.length) throw Error(errors.join("\n"));
  return reports;
}
