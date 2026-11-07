import { useEffect, useRef } from "react";

export default function AssetThumbnail({
  src,
  skin,
  rendered,
}: {
  src: string;
  skin: boolean;
  rendered?: string;
}) {
  const canvas = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (!skin || rendered || !src) return;
    let cancelled = false;
    const image = new Image();
    image.onload = () => {
      if (cancelled) return;
      const context = canvas.current?.getContext("2d");
      if (!context) return;
      context.clearRect(0, 0, 104, 104);
      context.imageSmoothingEnabled = false;
      const factor = image.width / 64;
      const face = (
        x: number,
        y: number,
        matrix: number[],
        hatX: number,
        hatY: number,
      ) => {
        context.save();
        context.setTransform(
          matrix[0] * 2,
          matrix[1] * 2,
          matrix[2] * 2,
          matrix[3] * 2,
          matrix[4] * 2,
          matrix[5] * 2,
        );
        context.drawImage(
          image,
          x * factor,
          y * factor,
          8 * factor,
          8 * factor,
          0,
          0,
          1,
          1,
        );
        context.drawImage(
          image,
          hatX * factor,
          hatY * factor,
          8 * factor,
          8 * factor,
          0,
          0,
          1,
          1,
        );
        context.restore();
      };
      face(8, 0, [21, 10, -21, 10, 26, 3], 40, 0);
      face(8, 8, [21, 10, 0, 24, 5, 13], 40, 8);
      face(16, 8, [21, -10, 0, 24, 26, 23], 48, 8);
    };
    image.src = src;
    return () => {
      cancelled = true;
    };
  }, [src, skin, rendered]);
  return rendered || !skin ? (
    <img src={rendered || src} alt="" />
  ) : (
    <canvas
      ref={canvas}
      width={104}
      height={104}
      aria-hidden="true"
      style={{ width: "100%", height: "100%", imageRendering: "pixelated" }}
    />
  );
}
