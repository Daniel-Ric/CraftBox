import type { Tile } from "./bridge";
import {
  biomeColor,
  heightColor,
  metadataAt,
  type MapStyle,
} from "./map-workbench";

                                                                              
                                                                                 
export function styleMapPixels(
  image: { data: Uint8ClampedArray; width: number; height: number },
  tile: Tile,
  tiles: Map<string, Tile>,
  s: MapStyle,
) {
  const stride = 7,
    profiles = new Float64Array(256 * stride);
  for (let z = 0; z < 16; z++)
    for (let x = 0; x < 16; x++) {
      const i = z * 16 + x,
        p = i * stride,
        h = tile.heights?.[i];
      profiles[p] = 1;
      if (h === -32768) {
        profiles[p + 6] = 1;
        continue;
      }
      let fixed: [number, number, number] | undefined;
      if (s.layer === "biome" && tile.biomes)
        fixed = biomeColor(tile.biomes[i]);
      else if (s.layer === "height" && h !== undefined)
        fixed = heightColor(h, s.elevationMin, s.elevationMax);
      let darken = 1;
      if (s.contours && h !== undefined) {
        const other = metadataAt(
          tiles,
          tile.x * 16 + x,
          tile.z * 16 + z - 1,
        ).height;
        if (
          other !== undefined &&
          other !== -32768 &&
          Math.floor(h / s.contourStep) !== Math.floor(other / s.contourStep)
        )
          darken = 0.55;
      }
                                                                                  
                                                                               
      profiles[p + 5] = darken;
      if (fixed) {
        profiles[p + 6] = 2;
        profiles[p + 2] = fixed[0];
        profiles[p + 3] = fixed[1];
        profiles[p + 4] = fixed[2];
        continue;
      }
      if (h !== undefined && s.layer === "surface") {
        let light = 1;
        if (s.relief) {
          const a = metadataAt(
              tiles,
              tile.x * 16 + x - 1,
              tile.z * 16 + z,
            ).height,
            b = metadataAt(tiles, tile.x * 16 + x, tile.z * 16 + z - 1).height,
            slope =
              (a === undefined || a === -32768 ? 0 : h - a) +
              (b === undefined || b === -32768 ? 0 : h - b);
          light = Math.max(0.65, Math.min(1.18, 1 - slope * 0.035));
        }
        profiles[p] = s.saturation * (1 - s.heightTint) * light;
        profiles[p + 1] = (1 - s.saturation) * (1 - s.heightTint) * light;
        if (s.heightTint > 0) {
          const tint = heightColor(h, s.elevationMin, s.elevationMax);
          for (let j = 0; j < 3; j++)
            profiles[p + 2 + j] = tint[j] * s.heightTint * light;
        }
      }
    }
  const { data, width, height } = image,
    n = width / 16,
    columns = new Int16Array(width);
  for (let x = 0; x < width; x++) columns[x] = Math.floor(x / n) * stride;
  for (let z = 0; z < height; z++) {
    const row = Math.floor(z / n) * 16 * stride,
      contourStrip = z % n < Math.max(1, n * 0.12);
    for (let x = 0; x < width; x++) {
      const p = row + columns[x],
        at = (z * width + x) * 4;
      if (profiles[p + 6] === 1) {
        data[at + 3] = 0;
        continue;
      }
      const shade = contourStrip ? profiles[p + 5] : 1,
        grayFactor = profiles[p + 1];
      if (profiles[p + 6] === 2) {
        data[at] = profiles[p + 2] * shade;
        data[at + 1] = profiles[p + 3] * shade;
        data[at + 2] = profiles[p + 4] * shade;
        continue;
      }
      const gray =
          grayFactor === 0
            ? 0
            : (data[at] * 0.2126 +
                data[at + 1] * 0.7152 +
                data[at + 2] * 0.0722) *
              grayFactor,
        factor = profiles[p];
      data[at] = (data[at] * factor + gray + profiles[p + 2]) * shade;
      data[at + 1] = (data[at + 1] * factor + gray + profiles[p + 3]) * shade;
      data[at + 2] = (data[at + 2] * factor + gray + profiles[p + 4]) * shade;
    }
  }
}
