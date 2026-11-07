export const imageFormats = [
  {
    id: "png",
    label: "PNG",
    detail: "Lossless · Transparency",
    use: "Screenshots, compositing and pixel art",
    alpha: true,
  },
  {
    id: "jpeg",
    label: "JPEG",
    detail: "Compact · Opaque",
    use: "Sharing and smaller image files",
    alpha: false,
  },
  {
    id: "webp",
    label: "WebP",
    detail: "Compact · Transparency",
    use: "Web publishing and previews",
    alpha: true,
  },
  {
    id: "tiff",
    label: "TIFF",
    detail: "Lossless · Transparency",
    use: "Image editing and print workflows",
    alpha: true,
  },
  {
    id: "bmp",
    label: "BMP",
    detail: "Uncompressed · Opaque",
    use: "Compatibility with older desktop tools",
    alpha: false,
  },
] as const;
export const modelFormats = [
  {
    id: "glb",
    label: "GLB",
    detail: "Recommended for Blender",
    use: "One file with embedded textures and glTF materials. Best starting point for Blender.",
    extension: ".glb",
  },
  {
    id: "gltf",
    label: "glTF",
    detail: "Editable scene package",
    use: "glTF scene and binary buffer in a ZIP. Preserves the same materials as GLB.",
    extension: ".gltf.zip",
  },
  {
    id: "obj",
    label: "OBJ",
    detail: "Broad compatibility",
    use: "OBJ mesh and basic diffuse MTL materials in a ZIP, with optional PNG textures. No advanced glTF lighting features.",
    extension: ".obj.zip",
  },
  {
    id: "fbx",
    label: "FBX",
    detail: "Binary FBX 7.4",
    use: "Static meshes, normals, UVs and diffuse materials in a ZIP, with optional PNG textures. Advanced glTF materials are simplified.",
    extension: ".fbx.zip",
  },
  {
    id: "stl",
    label: "STL",
    detail: "No materials or textures",
    use: "Binary triangles for CAD and printing workflows. No textures, materials or color. Check watertightness in your printing software.",
    extension: ".stl",
  },
] as const;
export const videoFormats = [
  {
    id: "mp4",
    label: "MP4",
    detail: "H.264 · AAC",
    use: "Broad playback support and sharing",
  },
  {
    id: "webm",
    label: "WebM",
    detail: "VP9 / VP8 · Opus",
    use: "Web playback and open media workflows",
  },
  {
    id: "mov",
    label: "MOV",
    detail: "H.264 · AAC",
    use: "QuickTime container for video editing",
  },
  {
    id: "mkv",
    label: "MKV",
    detail: "VP9 / VP8 · Opus",
    use: "Matroska container for flexible media workflows",
  },
  {
    id: "gif",
    label: "GIF",
    detail: "Animated · No audio",
    use: "Short looping previews with up to 256 colors per frame",
  },
] as const;
export type ImageFormat = (typeof imageFormats)[number]["id"];
export type ModelFormat = (typeof modelFormats)[number]["id"];
export type VideoFormat = (typeof videoFormats)[number]["id"];
export type ExportOptions = {
  format: ImageFormat | VideoFormat;
  quality: number;
  background: string;
  fps: number;
  bitrate: number;
  includeAudio: boolean;
  start: number;
  end: number;
};
export function isImageFormat(value: string): value is ImageFormat {
  return imageFormats.some((f) => f.id === value);
}
export async function imageExportData(
  png: string,
  format: ImageFormat,
  quality: number,
) {
  if (format !== "webp") return png;
  const image = new Image();
  await new Promise<void>((resolve, reject) => {
    image.onload = () => resolve();
    image.onerror = () =>
      reject(new Error("Could not decode the image for WebP export"));
    image.src = png;
  });
  const canvas = document.createElement("canvas");
  canvas.width = image.naturalWidth;
  canvas.height = image.naturalHeight;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Image encoder unavailable");
  ctx.drawImage(image, 0, 0);
  const data = canvas.toDataURL("image/webp", quality / 100);
  if (!data.startsWith("data:image/webp;"))
    throw new Error(
      "WebP encoding is unavailable in this WebView. Choose PNG or TIFF.",
    );
  return data;
}
