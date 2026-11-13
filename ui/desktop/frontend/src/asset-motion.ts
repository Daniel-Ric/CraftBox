import * as T from "three";
import type { MotionRuntime } from "./bridge";
import { poseAnimation, bedrockVector } from "./asset-geometry";
import {
  molang,
  molangScripts,
  previewContext,
  type MolangContext,
} from "./asset-molang";

export function motionPlayer(
  root: T.Group,
  runtime: MotionRuntime,
  contextOverrides: MolangContext = {},
) {
  const visibility: {
    node: T.Mesh;
    expression: unknown;
    condition?: unknown;
  }[] = [];
  root.traverse((node) => {
    if (!(node instanceof T.Mesh)) return;
    for (const rule of runtime.visibility ?? []) {
      const pattern = new RegExp(
        "^" +
          rule.pattern
            .split("*")
            .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
            .join(".*") +
          "$",
        "i",
      );
      if (pattern.test(node.name))
        visibility.push({
          node,
          expression: rule.expression,
          condition: rule.condition,
        });
    }
  });
  const context = {
    ...previewContext(0),
    ...runtime.previewQueries,
    ...contextOverrides,
  };
  const states = new Map<string, { name: string; entered: number }>();
  let lastTime = 2;
  molangScripts(runtime.scripts.initialize, context);
  Object.assign(context, runtime.previewQueries, contextOverrides);
  return (time: number, enabled = true) => {
    if (enabled) lastTime = time;
    else time = lastTime;
    Object.assign(
      context,
      previewContext(time),
      runtime.previewQueries,
      contextOverrides,
    );
    molangScripts(runtime.scripts.pre_animation, context);
    const particles: { effect: string; locator?: string }[] = [];
    const visit = (alias: string, weight: number, depth: number) => {
      if (depth > 8 || weight <= 0) return;
      const name = runtime.aliases[alias] ?? alias;
      const animation = runtime.animations[name];
      if (animation) {
        poseAnimation(
          root,
          { animations: { [name]: animation } },
          name,
          time,
          context,
          weight,
          true,
        );
        return;
      }
      const controller = runtime.controllers[name];
      if (!controller) return;
      const firstEvaluation = !states.has(name);
      let current = states.get(name) ?? {
        name: controller.initial_state ?? "default",
        entered: 0,
      };
      let state = controller.states?.[current.name];
      if (!state) return;
      context["query.state_time"] = time - current.entered;
      // Select the correct initial branch even in a paused preview. Follow
      // immediate routing states once, then freeze until playback is enabled.
      const routed = new Set<string>([current.name]);
      for (let hop = 0; (enabled || firstEvaluation) && hop < 16; hop++) {
        let destination: string | undefined;
        for (const transition of state.transitions ?? []) {
          const next = Object.entries(transition).find(([, test]) =>
            molang(test, context),
          );
          if (next) {
            destination = next[0];
            break;
          }
        }
        if (!destination || (firstEvaluation && routed.has(destination))) break;
        routed.add(destination);
        current = { name: destination, entered: time };
        state = controller.states?.[current.name];
        context["query.state_time"] = 0;
        if (!state || !firstEvaluation) break;
      }
      states.set(name, current);
      if (!state) return;
      for (const entry of state.animations ?? []) {
        if (typeof entry === "string") visit(entry, weight, depth + 1);
        else
          for (const [alias, condition] of Object.entries(entry))
            visit(alias, weight * molang(condition, context), depth + 1);
      }
      for (const effect of state.particle_effects ?? [])
        if (effect.effect) particles.push(effect);
    };
    for (const entry of runtime.scripts.animate ?? []) {
      if (typeof entry === "string") visit(entry, 1, 0);
      else
        for (const [alias, test] of Object.entries(entry))
          visit(alias, molang(test, context), 0);
    }
    for (const { node } of visibility) {
      node.userData.controllerVisible = true;
      node.visible = true;
    }
    for (const { node, expression, condition } of visibility) {
      if (condition !== undefined && !molang(condition, context, 1)) continue;
      const visible = molang(expression, context, 1) !== 0;
      node.userData.controllerVisible = visible;
      node.visible = visible;
    }
    return { context, particles };
  };
}

export async function particlePlayer(
  root: T.Group,
  runtime: MotionRuntime,
  load: (src: string) => Promise<T.Texture>,
) {
  type Active = {
    sprite: T.Sprite;
    age: number;
    life: number;
    velocity: T.Vector3;
    origin: T.Vector3;
    seed: number;
  };
  const emitters = await Promise.all(
    Object.entries(runtime.particles)
      .slice(0, 8)
      .map(async ([alias, source]) => {
        const texture = await load(source.image);
        return {
          alias,
          components: source.definition.components ?? {},
          texture,
          particles: [] as Active[],
          pool: [] as T.Sprite[],
          credit: 0,
        };
      }),
  );
  const group = new T.Group();
  group.name = "particleEffects";
  root.add(group);
  group.userData.disposeParticles = () => {
    for (const emitter of emitters) {
      for (const sprite of emitter.pool) {
        sprite.material.map?.dispose();
        sprite.material.dispose();
      }
      emitter.pool.length = 0;
    }
  };
  let previous = 0,
    serial = 0;
  return (
    time: number,
    context: MolangContext,
    effects: { effect: string; locator?: string }[],
    enabled: boolean,
  ) => {
    const dt = Math.min(0.1, Math.max(0, time - previous));
    previous = time;
    for (const emitter of emitters) {
      const c = emitter.components,
        active = effects.find((v) => v.effect === emitter.alias);
      const steady = c["minecraft:emitter_rate_steady"] ?? {},
        lifetime = c["minecraft:particle_lifetime_expression"] ?? {};
      const rate = Math.max(
        0,
        Math.min(100, molang(steady.spawn_rate, context, 10)),
      );
      emitter.credit += active && enabled ? rate * dt : 0;
      while (
        emitter.credit >= 1 &&
        emitter.particles.length <
          Math.min(128, molang(steady.max_particles, context, 60))
      ) {
        emitter.credit--;
        serial++;
        const seed = (serial * 0.61803398875) % 1,
          angle = seed * Math.PI * 2;
        const shape =
          c["minecraft:emitter_shape_sphere"] ??
          c["minecraft:emitter_shape_point"] ??
          {};
        const localContext = { ...context, "variable.particle_random_1": seed };
        const offset = (shape.offset ?? [0, 0, 0]).map((v: unknown) =>
          molang(v, localContext),
        );
        const anchor =
          root.getObjectByName("locator:" + (active?.locator ?? "")) ??
          root.getObjectByName(active?.locator ?? "");
        const origin = new T.Vector3();
        if (anchor) {
          anchor.getWorldPosition(origin);
          root.worldToLocal(origin);
        }
        const radius = molang(shape.radius, context);
        origin.add(bedrockVector(offset).multiplyScalar(16));
        origin.add(
          new T.Vector3(
            Math.cos(angle),
            seed - 0.5,
            Math.sin(angle),
          ).multiplyScalar(radius * 16),
        );
        const velocity = new T.Vector3(Math.cos(angle), seed, Math.sin(angle))
          .normalize()
          .multiplyScalar(
            molang(c["minecraft:particle_initial_speed"], context, 0) * 16,
          );
        const sprite =
          emitter.pool.pop() ??
          new T.Sprite(
            new T.SpriteMaterial({
              map: emitter.texture.clone(),
              transparent: true,
              depthWrite: false,
              color: 0xffffff,
            }),
          );
        sprite.material.color.set(0xffffff);
        sprite.name = "effectParticle";
        group.add(sprite);
        emitter.particles.push({
          sprite,
          age: 0,
          life: Math.max(
            0.05,
            Math.min(20, molang(lifetime.max_lifetime, context, 1)),
          ),
          velocity,
          origin,
          seed,
        });
      }
      emitter.credit = Math.min(1, emitter.credit);
      for (let i = emitter.particles.length - 1; i >= 0; i--) {
        const p = emitter.particles[i];
        p.age += dt;
        if (!enabled || p.age > p.life) {
          group.remove(p.sprite);
          emitter.pool.push(p.sprite);
          emitter.particles.splice(i, 1);
          continue;
        }
        const ctx = {
          ...context,
          "variable.particle_age": p.age,
          "variable.particle_lifetime": p.life,
          "variable.particle_random_1": p.seed,
        };
        const dynamic = c["minecraft:particle_motion_dynamic"] ?? {};
        const acceleration = (dynamic.linear_acceleration ?? [0, 0, 0]).map(
          (v: unknown) => molang(v, ctx),
        );
        p.velocity
          .add(bedrockVector(acceleration).multiplyScalar(16 * dt))
          .multiplyScalar(
            Math.exp(-molang(dynamic.linear_drag_coefficient, ctx) * dt),
          );
        p.origin.addScaledVector(p.velocity, dt);
        p.sprite.position.copy(p.origin);
        const billboard = c["minecraft:particle_appearance_billboard"] ?? {};
        const size = billboard.size ?? [0.1, 0.1];
        p.sprite.scale.set(
          molang(size[0], ctx, 0.1) * 16,
          molang(size[1], ctx, 0.1) * 16,
          1,
        );
        const uv = billboard.uv,
          flip = uv?.flipbook,
          map = p.sprite.material.map!;
        if (flip) {
          const frames = Math.max(1, molang(flip.max_frame, ctx, 1)),
            frame = Math.min(
              frames - 1,
              Math.floor(
                (flip.stretch_to_lifetime
                  ? (p.age / p.life) * frames
                  : p.age * molang(flip.frames_per_second, ctx, 20)) % frames,
              ),
            );
          map.repeat.set(
            flip.size_UV[0] / uv.texture_width,
            flip.size_UV[1] / uv.texture_height,
          );
          map.offset.set(
            (flip.base_UV[0] + frame * flip.step_UV[0]) / uv.texture_width,
            1 -
              (flip.base_UV[1] + frame * flip.step_UV[1] + flip.size_UV[1]) /
                uv.texture_height,
          );
        }
        const color = c["minecraft:particle_appearance_tinting"]?.color;
        if (typeof color === "string")
          p.sprite.material.color.set(
            color.length === 9 ? "#" + color.slice(3) : color,
          );
        else if (color?.gradient) {
          const value = molang(color.interpolant, ctx),
            keys = Object.keys(color.gradient)
              .map(Number)
              .sort((a, b) => a - b),
            lo = [...keys].reverse().find((k) => k <= value) ?? keys[0],
            hi = keys.find((k) => k >= value) ?? keys[keys.length - 1];
          const parse = (k: number) => {
            const key = Object.keys(color.gradient).find(
              (key) => Number(key) === k,
            )!;
            const hex = color.gradient[key];
            return new T.Color(hex.length === 9 ? "#" + hex.slice(3) : hex);
          };
          p.sprite.material.color
            .copy(parse(lo))
            .lerp(parse(hi), hi === lo ? 0 : (value - lo) / (hi - lo));
        }
      }
    }
  };
}
