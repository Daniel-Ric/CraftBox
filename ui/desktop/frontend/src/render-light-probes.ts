import * as THREE from "three";
import type { SceneLight } from "./bridge";
import { sampleIndirectLight } from "./render-lighting";

type ProbeMesh = THREE.Mesh<THREE.BufferGeometry>;

const chunkKey = (x: number, z: number) => `${x},${z}`;
const cellKey = (x: number, y: number, z: number) => x + 16 * z + 256 * y;
const unitByte = (value: number) =>
  Math.round(255 * THREE.MathUtils.clamp(value, 0, 1));
const pause = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

export class RenderLightProbes {
  readonly power = { value: 0 };
  readonly skyColor = { value: new THREE.Color() };
  private chunks = new Map<
    string,
    {
      group: THREE.Group;
      lights: SceneLight[];
      meshes: ProbeMesh[];
      opaque: ProbeMesh[];
    }
  >();

  setChunk(
    group: THREE.Group,
    lights: SceneLight[],
    opaqueMaterial: THREE.Material,
  ) {
    const meshes = (group.children as ProbeMesh[]).filter((mesh) =>
      mesh.geometry.hasAttribute("indirectProbe"),
    );
    this.chunks.set(chunkKey(group.position.x / 16, group.position.z / 16), {
      group,
      meshes,
      opaque: meshes.filter((mesh) => mesh.material === opaqueMaterial),
      lights: lights.map((light) => ({
        ...light,
        position: [
          light.position[0] + group.position.x,
          light.position[1] + group.position.y,
          light.position[2] + group.position.z,
        ],
      })),
    });
  }

  clear() {
    this.chunks.clear();
  }

  async bake(cancelled: () => boolean, updated: () => void) {
    const cells = new Map<string, Set<number>>();
    const point = new THREE.Vector3();
    const chunks = Array.from(this.chunks.values());
    const address = (x: number, z: number) =>
      chunkKey(Math.floor(x / 16), Math.floor(z / 16));
    const local = (value: number) => value - Math.floor(value / 16) * 16;
    const occupied = (x: number, y: number, z: number) =>
      cells.get(address(x, z))?.has(cellKey(local(x), y, local(z))) ?? false;

    let work = 0;
    for (const { group, opaque } of chunks) {
      for (const { geometry } of opaque) {
        const positions = geometry.getAttribute("position");
        const normals = geometry.getAttribute("normal");
        const indices = geometry.index!;
        for (let i = 0; i < indices.count; i += 3) {
          const a = indices.getX(i),
            b = indices.getX(i + 1),
            c = indices.getX(i + 2);
          const x = Math.floor(
            group.position.x +
              (positions.getX(a) + positions.getX(b) + positions.getX(c)) / 3 -
              normals.getX(a) * 0.05,
          );
          const y = Math.floor(
            (positions.getY(a) + positions.getY(b) + positions.getY(c)) / 3 -
              normals.getY(a) * 0.05,
          );
          const z = Math.floor(
            group.position.z +
              (positions.getZ(a) + positions.getZ(b) + positions.getZ(c)) / 3 -
              normals.getZ(a) * 0.05,
          );
          const key = address(x, z);
          let occupiedCells = cells.get(key);
          if (!occupiedCells) cells.set(key, (occupiedCells = new Set()));
          occupiedCells.add(cellKey(local(x), y, local(z)));
          if (++work % 32768 === 0) {
            await pause();
            if (cancelled()) return;
          }
        }
      }
    }

    for (const { group, meshes } of chunks) {
      if (cancelled()) return;
      const cx = Math.floor(group.position.x / 16),
        cz = Math.floor(group.position.z / 16);
      const sources: SceneLight[] = [];
      for (let dz = -1; dz <= 1; dz++)
        for (let dx = -1; dx <= 1; dx++)
          for (const light of this.chunks.get(chunkKey(cx + dx, cz + dz))
            ?.lights ?? [])
            sources.push(light);
      const probes = new Map<string, ReturnType<typeof sampleIndirectLight>>();
      for (const { geometry } of meshes) {
        const positions = geometry.getAttribute("position");
        const normals = geometry.getAttribute("normal");
        const block = geometry.getAttribute("blockLight");
        const output = geometry.getAttribute(
          "indirectProbe",
        ) as THREE.BufferAttribute;
        const data = output.array as Uint8Array;
        for (let i = 0; i < positions.count; i++) {
          const x = Math.floor(positions.getX(i) + normals.getX(i) * 0.55);
          const y = Math.floor(positions.getY(i) + normals.getY(i) * 0.55);
          const z = Math.floor(positions.getZ(i) + normals.getZ(i) * 0.55);
          const level = block.getX(i);
          const key = `${x},${y},${z},${level}`;
          let probe = probes.get(key);
          if (!probe) {
            point.set(
              group.position.x + x + 0.5,
              y + 0.5,
              group.position.z + z + 0.5,
            );
            probe = sampleIndirectLight(point, level, occupied, sources);
            probes.set(key, probe);
          }
          data[i * 4] = unitByte(probe.block[0]);
          data[i * 4 + 1] = unitByte(probe.block[1]);
          data[i * 4 + 2] = unitByte(probe.block[2]);
          data[i * 4 + 3] = unitByte(probe.sky);
          if (++work % 16384 === 0) {
            await pause();
            if (cancelled()) return;
          }
        }
        output.needsUpdate = true;
      }
      updated();
    }
  }
}
