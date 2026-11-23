export type SkinTextureAnimation = {
  ImageWidth: number;
  ImageHeight: number;
  ImageData: string;
  FrameCount: number;
  ExpressionType: number;
  AnimationType: number;
};

export function decodeSkinTextureAnimation(animation: SkinTextureAnimation) {
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
    throw Error("Animated skin image exceeds preview limits");
  const raw = atob(animation.ImageData);
  if (raw.length !== width * height * 4)
    throw Error("Animated skin image has invalid dimensions");
  return {
    width,
    height,
    count,
    pixels: Uint8Array.from(raw, (character) => character.charCodeAt(0)),
  };
}
