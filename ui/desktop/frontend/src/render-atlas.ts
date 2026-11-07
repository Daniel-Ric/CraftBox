import * as THREE from "three";
import { stableShadowChunk } from "./render-shadow-filter";
export const filteredAlphaTestChunk =
  THREE.ShaderChunk.alphatest_fragment.replace(
    "diffuseColor.a = smoothstep( alphaTest, alphaTest + fwidth( diffuseColor.a ), diffuseColor.a );",
    `float hardCoverage=smoothstep(alphaTest,alphaTest+max(fwidth(diffuseColor.a),0.0001),diffuseColor.a);
 #ifdef USE_MAP
 float footprint=max(length(dFdx(vMapUv)),length(dFdy(vMapUv)))*float(textureSize(map,0).x);
 float filteredCoverage=clamp(diffuseColor.a,0.0,1.0);
 diffuseColor.a=mix(hardCoverage,filteredCoverage,smoothstep(1.0,2.0,footprint)*vAtlasTile.z);
 #else
 diffuseColor.a=hardCoverage;
 #endif`,
  );
type Patch = { x: number; y: number; size: number; pixels: Uint8Array };
const linear = Float64Array.from({ length: 256 }, (_, i) => {
  const v = i / 255;
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
});
const srgb = (v: number) =>
  Math.round(
    255 * (v <= 0.0031308 ? v * 12.92 : 1.055 * v ** (1 / 2.4) - 0.055),
  );



export class AtlasMipmaps {
  textures: THREE.DataTexture[] = [];
  atlasSize: number;
  constructor(atlasSize: number) {
    this.atlasSize = atlasSize;
    const cells = Math.ceil(atlasSize / 18);
    for (let level = 1; level <= 4; level++) {
      const size = THREE.MathUtils.ceilPowerOfTwo(cells * ((16 >> level) + 2));
      const t = new THREE.DataTexture(
        new Uint8Array(size * size * 4),
        size,
        size,
      );
      t.colorSpace = THREE.SRGBColorSpace;
      t.magFilter = THREE.LinearFilter;
      t.minFilter = THREE.LinearFilter;
      t.generateMipmaps = false;
      this.textures.push(t);
    }
  }
  update(patches: Patch[]) {
    for (const patch of patches) {
      if (patch.size !== 18) continue;
      for (let level = 1; level <= 4; level++) {
        const n = 16 >> level,
          block = 1 << level,
          stride = n + 2,
          t = this.textures[level - 1],
          image = t.image;
        const ox = Math.round(patch.x / 18) * stride,
          oy = Math.round(patch.y / 18) * stride;
        const tile = new Uint8Array(n * n * 4);
        for (let y = 0; y < n; y++)
          for (let x = 0; x < n; x++) {
            let alpha = 0,
              r = 0,
              g = 0,
              b = 0;
            for (let by = 0; by < block; by++)
              for (let bx = 0; bx < block; bx++) {
                const i = ((1 + y * block + by) * 18 + 1 + x * block + bx) * 4,
                  a = patch.pixels[i + 3] / 255;
                alpha += a;
                r += linear[patch.pixels[i]] * a;
                g += linear[patch.pixels[i + 1]] * a;
                b += linear[patch.pixels[i + 2]] * a;
              }
            const at = (y * n + x) * 4;
            if (alpha > 0) {
              tile[at] = srgb(r / alpha);
              tile[at + 1] = srgb(g / alpha);
              tile[at + 2] = srgb(b / alpha);
            }
            tile[at + 3] = Math.round((alpha / (block * block)) * 255);
          }
        for (let y = -1; y <= n; y++)
          for (let x = -1; x <= n; x++) {
            const from =
              (Math.max(0, Math.min(n - 1, y)) * n +
                Math.max(0, Math.min(n - 1, x))) *
              4;
            image.data!.set(
              tile.subarray(from, from + 4),
              ((oy + y + 1) * image.width + ox + x + 1) * 4,
            );
          }
      }
    }
    for (const t of this.textures) t.needsUpdate = true;
  }
  install(material: THREE.Material) {
    material.onBeforeCompile = (shader) => {
      this.textures.forEach((t, i) => {
        shader.uniforms[`terrainMip${i + 1}`] = { value: t };
      });
      shader.vertexShader = shader.vertexShader
        .replace(
          "#include <common>",
          "#include <common>\nattribute vec3 atlasTile; varying vec3 vAtlasTile;",
        )
        .replace(
          "#include <uv_vertex>",
          "#include <uv_vertex>\nvAtlasTile=atlasTile;",
        );
      const sample = this.textures
        .map(
          (t, i) =>
            `if(level<${i + 1}.5) return texture2D(terrainMip${i + 1},(vAtlasTile.xy*${(16 >> (i + 1)) + 2}.0+1.0+local*${16 >> (i + 1)}.0)/${t.image.width}.0);`,
        )
        .join("\n");
      const helpers = `
    varying vec3 vAtlasTile;
    #ifdef USE_MAP
    uniform sampler2D terrainMip1,terrainMip2,terrainMip3,terrainMip4;
    vec4 terrainLevel(float level,vec2 local){
      if(level<0.5)return texture2D(map,(vAtlasTile.xy*18.0+1.0+local*16.0)/${this.atlasSize}.0);
      ${sample}
      return vec4(1.0);
    }
    vec4 terrainTrilinear(vec2 local,float lod){local=clamp(local,vec2(0.0),vec2(1.0));float low=floor(lod);return mix(terrainLevel(low,local),terrainLevel(min(low+1.0,4.0),local),fract(lod));}
    vec4 terrainFiltered(vec2 uv){
      if(vAtlasTile.z<0.5)return texture2D(map,uv);
      vec2 dx=dFdx(uv)*${this.atlasSize}.0,dy=dFdy(uv)*${this.atlasSize}.0;
      float major=max(length(dx),length(dy)),minor=min(length(dx),length(dy));
      if(major<=1.0)return texture2D(map,uv);
      float lod=clamp(log2(max(1.0,max(minor,major/4.0))),0.0,4.0);
      vec2 axis=(length(dx)>length(dy)?dx:dy)/16.0;
      vec2 local=(uv*${this.atlasSize}.0-vAtlasTile.xy*18.0-1.0)/16.0;
      return (terrainTrilinear(local-axis*0.375,lod)+terrainTrilinear(local-axis*0.125,lod)+terrainTrilinear(local+axis*0.125,lod)+terrainTrilinear(local+axis*0.375,lod))*0.25;
    }
    #endif`;
      shader.fragmentShader = shader.fragmentShader
        .replace(
          "#include <map_pars_fragment>",
          "#include <map_pars_fragment>\n" + helpers,
        )
        .replace(
          "#include <map_fragment>",
          THREE.ShaderChunk.map_fragment.replace(
            "texture2D( map, vMapUv )",
            "terrainFiltered( vMapUv )",
          ),
        )
        .replace("#include <alphatest_fragment>", filteredAlphaTestChunk)
        .replace(
          "#include <shadowmap_pars_fragment>",
          stableShadowChunk(THREE.ShaderChunk.shadowmap_pars_fragment),
        );
    };
    material.customProgramCacheKey = () =>
      `isolated-terrain-mips-${this.atlasSize}`;
    material.needsUpdate = true;
  }
  dispose() {
    this.textures.forEach((t) => t.dispose());
  }
}

export function atlasTileAttributes(uvs: Float32Array, atlasSize: number) {
  const out = new Uint16Array((uvs.length / 2) * 3);


  for (let v = 0; v + 3 < uvs.length / 2; v += 4) {
    let minX = Infinity,
      minY = Infinity,
      maxX = -Infinity,
      maxY = -Infinity;
    for (let i = 0; i < 4; i++) {
      const x = uvs[(v + i) * 2] * atlasSize,
        y = uvs[(v + i) * 2 + 1] * atlasSize;
      minX = Math.min(minX, x);
      maxX = Math.max(maxX, x);
      minY = Math.min(minY, y);
      maxY = Math.max(maxY, y);
    }
    const cx = Math.floor((minX + maxX) / 36),
      cy = Math.floor((minY + maxY) / 36);
    const safe =
      minX >= cx * 18 + 1 - 0.001 &&
      maxX <= cx * 18 + 17 + 0.001 &&
      minY >= cy * 18 + 1 - 0.001 &&
      maxY <= cy * 18 + 17 + 0.001;
    for (let i = 0; i < 4; i++) out.set([cx, cy, safe ? 1 : 0], (v + i) * 3);
  }
  return out;
}
