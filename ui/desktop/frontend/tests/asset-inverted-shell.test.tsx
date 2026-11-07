import { expect, it } from "vitest";
import * as T from "three";
import { bedrockModel } from "../src/asset-geometry";
it("preserves inward negative-size shells used behind atlas-based text models", () => {
  for (const size of [
    [16, 16, 16],
    [-16, 16, 16],
    [-16, -16, -16],
    [-16, -16, 16],
  ]) {
    const root = bedrockModel(
      {
        "minecraft:geometry": [
          {
            description: { texture_width: 64, texture_height: 64 },
            bones: [
              {
                name: "shell",
                cubes: [{ origin: [0, 0, 0], size, uv: [0, 0] }],
              },
            ],
          },
        ],
      },
      new T.MeshStandardMaterial(),
    );
    root.traverse((node) => {
      if (!(node instanceof T.Mesh)) return;
      const g = node.geometry,
        p = g.attributes.position,
        index = g.index!,
        n = g.attributes.normal;
      const a = new T.Vector3().fromBufferAttribute(p, index.getX(0)),
        b = new T.Vector3().fromBufferAttribute(p, index.getX(1)),
        c = new T.Vector3().fromBufferAttribute(p, index.getX(2));
      const cross = b.clone().sub(a).cross(c.clone().sub(a)),
        center = a.clone().add(b).add(c).divideScalar(3);
      expect(Math.sign(cross.dot(center))).toBe(
        Math.sign(size.reduce((a, b) => a * b, 1)),
      );
      expect(
        cross.dot(new T.Vector3().fromBufferAttribute(n, index.getX(0))),
      ).toBeGreaterThan(0);
    });
  }
});

it("keeps per-face atlas corners aligned on signed cubes, including rotated UVs", () => {
  // Independent corner convention from Blockbench's Bedrock importer and
  // CubeFace coordinates: x is reflected; up/down reverse both UV axes.
  for (const signs of [
    [1, 1, 1],
    [-1, 1, 1],
    [1, -1, 1],
    [1, 1, -1],
    [-1, -1, -1],
  ]) {
    for (const rotation of [0, 90, 180, 270]) {
      const [w, h, d] = signs.map((sign, i) => sign * [7, 11, 13][i]);
      const corners = [
        [
          [-w, h, -d],
          [-w, h, d],
          [-w, -h, -d],
          [-w, -h, d],
        ],
        [
          [w, h, d],
          [w, h, -d],
          [w, -h, d],
          [w, -h, -d],
        ],
        [
          [-w, h, -d],
          [w, h, -d],
          [-w, h, d],
          [w, h, d],
        ],
        [
          [-w, -h, d],
          [w, -h, d],
          [-w, -h, -d],
          [w, -h, -d],
        ],
        [
          [-w, h, d],
          [w, h, d],
          [-w, -h, d],
          [w, -h, d],
        ],
        [
          [w, h, -d],
          [-w, h, -d],
          [w, -h, -d],
          [-w, -h, -d],
        ],
      ];
      const faces = ["west", "east", "up", "down", "south", "north"];
      const root = bedrockModel(
        {
          "minecraft:geometry": [
            {
              description: { texture_width: 64, texture_height: 64 },
              bones: [
                {
                  name: "atlas",
                  cubes: [
                    {
                      origin: [0, 0, 0],
                      size: [w, h, d],
                      uv: Object.fromEntries(
                        faces.map((face) => [
                          face,
                          {
                            uv: [17, 23],
                            uv_size: [-5, 9],
                            uv_rotation: rotation,
                          },
                        ]),
                      ),
                    },
                  ],
                },
              ],
            },
          ],
        },
        new T.MeshStandardMaterial(),
      );
      root.traverse((node) => {
        if (!(node instanceof T.Mesh)) return;
        const { position, uv } = node.geometry.attributes;
        for (let face = 0; face < 6; face++) {
          for (let corner = 0; corner < 4; corner++) {
            const expected = corners[face][corner].map((value) => value / 2);
            const vertex = [0, 1, 2, 3]
              .map((i) => face * 4 + i)
              .find((i) =>
                expected.every(
                  (value, axis) =>
                    Math.abs(position.array[i * 3 + axis] - value) < 1e-6,
                ),
              );
            expect(vertex).toBeDefined();
            let s = corner % 2,
              t = Math.floor(corner / 2);
            for (let step = 0; step < rotation / 90; step++)
              [s, t] = [t, 1 - s];
            if (face === 2 || face === 3) {
              s = 1 - s;
              t = 1 - t;
            }
            expect(uv.getX(vertex!)).toBeCloseTo((17 - s * 5) / 64);
            expect(uv.getY(vertex!)).toBeCloseTo(1 - (23 + t * 9) / 64);
          }
        }
      });
    }
  }
});
