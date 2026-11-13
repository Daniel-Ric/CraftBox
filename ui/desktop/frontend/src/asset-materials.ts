import * as T from "three";
import type { PackObject } from "./bridge";
const originalTextures = new WeakMap<T.Texture, T.Texture>();

export function emissivePixels(source: Uint8ClampedArray) {
  const color = new Uint8ClampedArray(source),
    emission = new Uint8ClampedArray(source.length);
  for (let i = 0; i < source.length; i += 4) {
    const alpha = source[i + 3],
      strength = alpha ? 1 - alpha / 255 : 0;
    color[i + 3] = alpha ? 255 : 0;
    for (let channel = 0; channel < 3; channel++)
      emission[i + channel] = Math.round(source[i + channel] * strength);
    emission[i + 3] = 255;
  }
  return { color, emission };
}
export function configureMaterial(
  material: T.MeshStandardMaterial,
  original: T.Texture | null,
  mode: string,
  textures: T.Texture[],
) {
  material.userData.bedrockMaterial = mode;
  if (original) original = originalTextures.get(original) ?? original;
  material.map = original;
  material.side = /one_sided|single_sided/.test(mode)
    ? T.FrontSide
    : T.DoubleSide;
  material.transparent =
    /alphablend|slime_outer/.test(mode) || mode === "blend";
  material.depthWrite = !material.transparent;
  material.alphaTest = mode === "opaque" || mode === "blend" ? 0 : 0.01;
  material.emissive.set(0);
  material.emissiveMap = null;
  if (!mode.includes("emissive") || !original) return;
  const image = original.image as HTMLImageElement;
  const width = image.naturalWidth || image.width,
    height = image.naturalHeight || image.height;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (!context) return;
  context.drawImage(image, 0, 0);
  const pixels = context.getImageData(0, 0, width, height),
    values = emissivePixels(pixels.data);
  const convert = (data: Uint8ClampedArray) => {
    pixels.data.set(data);
    context.putImageData(pixels, 0, 0);
    const copy = document.createElement("canvas");
    copy.width = width;
    copy.height = height;
    copy.getContext("2d")!.drawImage(canvas, 0, 0);
    const texture = new T.CanvasTexture(copy);
    texture.colorSpace = T.SRGBColorSpace;
    texture.magFilter = T.NearestFilter;
    texture.minFilter = T.NearestFilter;
    texture.generateMipmaps = false;
    textures.push(texture);
    originalTextures.set(texture, original!);
    return texture;
  };
  material.map = convert(values.color);
  material.emissiveMap = convert(values.emission);
  material.emissive.set(0xffffff);
}
export function applyObjectRules(
  root: T.Group,
  object: PackObject | undefined,
  textures: T.Texture[],
) {
  if (!object) return;
  const configured = new Map<T.Material, Map<string, T.MeshStandardMaterial>>();
  const matches = (pattern: string, name: string) =>
    new RegExp(
      "^" +
        pattern
          .split("*")
          .map((v) => v.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
          .join(".*") +
        "$",
      "i",
    ).test(name);
  root.traverse((node) => {
    if (!(node instanceof T.Mesh)) return;
    for (const rule of object.visibility ?? [])
      if (matches(rule.pattern, node.name)) {
        node.userData.controllerVisible = rule.visible;
        node.visible = rule.visible;
      }
    const rule = [...(object.boneMaterials ?? [])]
      .reverse()
      .find((v) => matches(v.pattern, node.name));
    if (rule) {
      const materials = (
        Array.isArray(node.material) ? node.material : [node.material]
      ).map((source) => {
        const key =
          rule.material +
          ":" +
          (node.userData.authoredCulling ? source.side : "auto");
        let variants = configured.get(source);
        if (!variants) {
          variants = new Map();
          configured.set(source, variants);
        }
        const cached = variants.get(key);
        if (cached) return cached;
        const material = source.clone() as T.MeshStandardMaterial;
        const sided = source.side;
        configureMaterial(material, material.map, rule.material, textures);
        if (node.userData.authoredCulling) material.side = sided;
        variants.set(key, material);
        return material;
      });
      node.material = materials.length === 1 ? materials[0] : materials;
    }
  });
}
