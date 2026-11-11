import type { BlockbenchModel } from "./blockbench-project";

type Data = Record<string, any>;
const record = (value: unknown): Data =>
  value && typeof value === "object" && !Array.isArray(value)
    ? (value as Data)
    : {};
const list = (value: unknown): any[] => (Array.isArray(value) ? value : []);
const id = () => crypto.randomUUID();
const script = (value: unknown) =>
  Array.isArray(value) ? value.join("\n") : String(value ?? "");
const canonical = (value: string) =>
  value
    .toLowerCase()
    .replace(/^q\./, "query.")
    .replace(/^v\./, "variable.")
    .replace(/^c\./, "context.")
    .replace(/^t\./, "temp.");

function invert(value: unknown): unknown {
  if (typeof value === "number") return -value;
  if (typeof value === "string") {
    if (!value.trim()) return 0;
    if (Number.isFinite(Number(value))) return -Number(value);
    const tokens = /'[^']*'|"[^"]*"|\breturn\b/gi;
    let result = "",
      cursor = 0,
      returned = false;
    for (const token of value.matchAll(tokens)) {
      if (token[0].toLowerCase() !== "return") continue;
      const start = token.index! + token[0].length;
      let end = start,
        quote = "",
        depth = 0;
      for (; end < value.length; end++) {
        const character = value[end];
        if (quote) {
          if (character === quote) quote = "";
          continue;
        }
        if (character === "'" || character === '"') {
          quote = character;
          continue;
        }
        if (character === "(" || character === "[") depth++;
        if (character === ")" || character === "]") depth--;
        if (!depth && (character === ";" || character === "}")) break;
      }
      result +=
        value.slice(cursor, token.index) +
        `return -(${value.slice(start, end).trim() || "0"})`;
      cursor = end;
      returned = true;
    }
    if (returned) return result + value.slice(cursor);
    let boundary = -1,
      quote = "",
      depth = 0;
    for (let index = 0; index < value.length; index++) {
      const character = value[index];
      if (quote) {
        if (character === quote) quote = "";
        continue;
      }
      if (character === "'" || character === '"') {
        quote = character;
        continue;
      }
      if (character === "(" || character === "[" || character === "{") depth++;
      if (")]}".includes(character)) depth--;
      if (!depth && character === ";" && value.slice(index + 1).trim())
        boundary = index;
    }
    const expression = value
      .slice(boundary + 1)
      .replace(/;+\s*$/, "")
      .trim();
    if (/(?:^|[^=!<>])=(?!=)/.test(expression.replace(/'[^']*'|"[^"]*"/g, "")))
      return `${value.replace(/;+\s*$/, "")};0`;
    return value.slice(0, boundary + 1) + `-(${expression})`;
  }
  throw Error("Animation contains an invalid coordinate.");
}

function points(value: unknown, channel: string): Data[] {
  const data = record(value);
  if (data.pre !== undefined || data.post !== undefined) {
    const before = data.pre === undefined ? [] : points(data.pre, channel);
    const after = data.post === undefined ? [] : points(data.post, channel);
    return JSON.stringify(before) === JSON.stringify(after)
      ? before
      : [...before, ...after];
  }
  if (typeof value === "number" || typeof value === "string")
    return [{ x: value, y: value, z: value }];
  if (
    !Array.isArray(value) ||
    value.length !== 3 ||
    value.some(
      (coordinate) =>
        !(
          typeof coordinate === "string" ||
          (typeof coordinate === "number" && Number.isFinite(coordinate))
        ),
    )
  )
    throw Error("Animation contains an invalid keyframe vector.");
  const [x, y, z] = value;
  return [
    {
      x: channel === "scale" ? x : invert(x),
      y: channel === "rotation" ? invert(y) : y,
      z,
    },
  ];
}

function time(value: string, name: string) {
  const result = Number(value);
  if (!value.trim() || !Number.isFinite(result) || result < 0)
    throw Error(`Invalid animation time in ${name}.`);
  return result;
}

function snapping(times: number[]) {
  if (new Set(times).size < 2) return 20;
  for (let rate = 10; rate <= 100; rate++)
    if (
      times.every(
        (value) => Math.abs(value * rate - Math.round(value * rate)) < 0.01,
      )
    )
      return rate;
  return 100;
}

function previewInputs(model: BlockbenchModel) {
  const native = new Set(
    [
      "anim_time",
      "life_time",
      "time_stamp",
      "delta_time",
      "camera_rotation",
      "rotation_to_camera",
      "distance_from_camera",
      "lod_index",
      "camera_distance_range_lerp",
      "is_first_person",
      "get_default_bone_pivot",
      "state_time",
      "all_animations_finished",
      "any_animation_finished",
    ].map((name) => `query.${name}`),
  );
  native.add("context.is_first_person");
  const inputs = new Map<string, string>();
  const inspect = (value: unknown) => {
    if (Array.isArray(value)) value.forEach(inspect);
    else if (value && typeof value === "object")
      Object.values(value).forEach(inspect);
    else if (typeof value === "string") {
      const pattern =
        /'[^']*'|"[^"]*"|\b((?:query|q|variable|v|context|c|temp|t)\.[a-z_][\w]*)/gi;
      for (const match of value.matchAll(pattern)) {
        if (!match[1]) continue;
        const name = canonical(match[1]);
        if (
          native.has(name) ||
          name.startsWith("temp.") ||
          /^\s*\(/.test(value.slice(match.index! + match[0].length))
        )
          continue;
        inputs.set(
          name,
          /\.(?:is_|has_|can_)/.test(name)
            ? `toggle('${name}')`
            : `slider('${name}')`,
        );
      }
    }
  };
  inspect({
    animations: { ...model.motion?.animations, ...model.animations },
    controllers: model.motion?.controllers,
    scripts: model.motion?.scripts,
  });
  for (const [key, value] of Object.entries(
    model.motion?.previewQueries ?? {},
  )) {
    const name = canonical(key);
    if (
      !native.has(name) &&
      /^(query|variable|context)\.\w+$/.test(name) &&
      Number.isFinite(value)
    )
      inputs.set(name, String(value));
  }
  return [...inputs.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, value]) => `${name} = ${value}`);
}

export function createBlockbenchMotion(
  model: BlockbenchModel,
  groups: Map<string, Data>,
  warnings: Set<string>,
) {
  const sourceAnimations = { ...model.motion?.animations, ...model.animations };
  const sourceControllers = model.motion?.controllers ?? {};
  if (
    Object.keys(sourceAnimations).length > 4096 ||
    Object.keys(sourceControllers).length > 1024
  )
    throw Error("Model exceeds editable animation graph limits.");
  const animations: Data[] = [];
  const animationIDs = new Map(
    Object.keys(sourceAnimations).map((name) => [name.toLowerCase(), id()]),
  );
  let keyCount = 0;
  for (const [name, raw] of Object.entries(sourceAnimations)) {
    const animation = record(raw),
      animators: Data = {};
    let length = 0;
    const times: number[] = [];
    for (const [boneName, rawChannels] of Object.entries(
      record(animation.bones),
    )) {
      const group = groups.get(boneName.toLowerCase());
      if (!group)
        warnings.add(
          `Animation ${name} references missing bone ${boneName}; its track remains editable and the source is retained.`,
        );
      const channels = record(rawChannels),
        keyframes: Data[] = [];
      for (const channel of ["position", "rotation", "scale"]) {
        const source = channels[channel];
        if (source === undefined) continue;
        const data = record(source);
        const entries: [string, unknown][] =
          typeof source !== "object" ||
          Array.isArray(source) ||
          data.pre !== undefined ||
          data.post !== undefined
            ? [["0", source]]
            : Object.entries(data);
        const channelKeys = entries
          .map(([timestamp, value]) => {
            const at = time(timestamp, name),
              entry = record(value);
            const interpolation = entry.lerp_mode ?? "linear";
            if (!["linear", "catmullrom", "step"].includes(interpolation))
              throw Error(
                `Unsupported animation interpolation in ${name}: ${interpolation}`,
              );
            length = Math.max(length, at);
            times.push(at);
            const scalar = entry.post ?? entry.pre ?? value;
            return {
              uuid: id(),
              channel,
              time: at,
              interpolation,
              data_points: points(value, channel),
              uniform: typeof scalar === "string" || typeof scalar === "number",
            };
          })
          .sort((a, b) => a.time - b.time);
        let stepped = false;
        for (let index = 0; index < channelKeys.length; index++) {
          const current = channelKeys[index],
            next = channelKeys[index + 1];
          if (
            next?.data_points.length === 2 &&
            JSON.stringify(current.data_points.at(-1)) ===
              JSON.stringify(next.data_points[0])
          ) {
            next.data_points.shift();
            current.interpolation = "step";
            stepped = true;
          } else if (!next && stepped) current.interpolation = "step";
        }
        keyframes.push(...channelKeys);
      }
      animators[group?.uuid ?? id()] = {
        name: group?.name ?? boneName,
        type: "bone",
        rotation_global: channels.relative_to?.rotation === "entity",
        keyframes,
      };
      keyCount += keyframes.length;
    }
    const effectKeys: Data[] = [];
    for (const [field, channel] of [
      ["sound_effects", "sound"],
      ["particle_effects", "particle"],
      ["timeline", "timeline"],
    ]) {
      for (const [timestamp, rawEffect] of Object.entries(
        record(animation[field]),
      )) {
        const at = time(timestamp, name),
          values = Array.isArray(rawEffect) ? rawEffect : [rawEffect];
        effectKeys.push({
          uuid: id(),
          channel,
          time: at,
          data_points:
            channel === "timeline"
              ? [{ script: values.join("\n") }]
              : values.map((value) => ({
                  ...(typeof value === "string"
                    ? { effect: value }
                    : record(value)),
                  ...(channel === "particle"
                    ? { script: record(value).pre_effect_script ?? "" }
                    : {}),
                })),
        });
        times.push(at);
        length = Math.max(length, at);
      }
    }
    if (effectKeys.length)
      animators.effects = {
        type: "effect",
        keyframes: effectKeys.sort((a, b) => a.time - b.time),
      };
    keyCount += effectKeys.length;
    if (keyCount > 500000)
      throw Error("Model exceeds 500000 animation keyframes.");
    if (animation.animation_length !== undefined) {
      if (
        !Number.isFinite(animation.animation_length) ||
        animation.animation_length < 0
      )
        throw Error(`Invalid animation length in ${name}.`);
      length = animation.animation_length;
    }
    if (
      typeof animation.loop === "string" &&
      animation.loop !== "hold_on_last_frame"
    )
      warnings.add(
        `Animation ${name} has a conditional loop expression; Blockbench shows a looping track and retains the original expression in the source data.`,
      );
    animations.push({
      uuid: animationIDs.get(name.toLowerCase()),
      name,
      loop:
        animation.loop === "hold_on_last_frame"
          ? "hold"
          : animation.loop
            ? "loop"
            : "once",
      override: animation.override_previous_animation === true,
      length,
      snapping: snapping(times),
      anim_time_update: script(animation.anim_time_update),
      blend_weight: script(animation.blend_weight),
      start_delay: script(animation.start_delay),
      loop_delay: script(animation.loop_delay),
      animators,
    });
  }
  const aliases = new Map(
    Object.entries(model.motion?.aliases ?? {}).map(([name, ref]) => [
      name.toLowerCase(),
      ref,
    ]),
  );
  const resolve = (name: string) => {
    const seen = new Set<string>();
    let key = name.toLowerCase();
    while (aliases.has(key) && !seen.has(key)) {
      seen.add(key);
      key = aliases.get(key)!.toLowerCase();
    }
    return key;
  };
  const controllerNames = new Set(
    Object.keys(sourceControllers).map((name) => name.toLowerCase()),
  );
  let stateCount = 0;
  const controllers = Object.entries(sourceControllers).map(([name, raw]) => {
    const source = record(raw),
      states = record(source.states);
    const stateIDs = new Map(Object.keys(states).map((name) => [name, id()]));
    stateCount += stateIDs.size;
    if (stateCount > 16384)
      throw Error("Model exceeds 16384 animation controller states.");
    const converted = Object.entries(states).map(([stateName, rawState]) => {
      const state = record(rawState);
      let duration =
        typeof state.blend_transition === "number" ? state.blend_transition : 0;
      const curve: Data = {};
      if (
        state.blend_transition &&
        typeof state.blend_transition === "object"
      ) {
        const values = Object.entries(record(state.blend_transition)).map(
          ([at, value]) => [time(at, name), value] as const,
        );
        duration = Math.max(0, ...values.map(([at]) => at));
        if (duration > 0)
          for (const [at, value] of values)
            curve[String(at / duration)] = value;
      }
      if (!Number.isFinite(duration) || duration < 0)
        throw Error(`Invalid state blend duration in ${name}.`);
      if (Object.keys(record(state.variables)).length)
        warnings.add(
          `Controller ${name} uses state variables/remap curves that Blockbench does not execute; their original definitions are retained.`,
        );
      return {
        uuid: stateIDs.get(stateName),
        name: stateName,
        animations: list(state.animations).flatMap((entry) =>
          (typeof entry === "string"
            ? [[entry, ""]]
            : Object.entries(record(entry))
          ).map(([key, weight]) => {
            const target = resolve(key),
              animation = animationIDs.get(target) ?? "";
            if (controllerNames.has(target))
              warnings.add(
                `Controller ${name} references nested controller ${key}; both diagrams are editable, but Blockbench does not play nested controllers through animation links.`,
              );
            else if (!animation)
              warnings.add(
                `Controller ${name} references missing animation ${key}; the unresolved link is retained.`,
              );
            return {
              uuid: id(),
              key,
              animation,
              blend_value: String(weight ?? ""),
            };
          }),
        ),
        transitions: list(state.transitions).flatMap((entry) =>
          Object.entries(record(entry)).map(([target, condition]) => {
            if (!stateIDs.has(target))
              warnings.add(
                `Controller ${name} references missing state ${target}; its transition condition is retained.`,
              );
            return {
              uuid: id(),
              target: stateIDs.get(target) ?? "",
              condition: String(condition ?? ""),
            };
          }),
        ),
        on_entry: script(state.on_entry),
        on_exit: script(state.on_exit),
        blend_transition: duration,
        blend_transition_curve: curve,
        blend_via_shortest_path: state.blend_via_shortest_path === true,
        particles: list(state.particle_effects).map((effect) => ({
          uuid: id(),
          effect: effect.effect ?? "",
          locator: effect.locator ?? "",
          bind_to_actor: effect.bind_to_actor !== false,
          pre_effect_script: effect.pre_effect_script ?? "",
          file: "",
        })),
        sounds: list(state.sound_effects).map((effect) => ({
          uuid: id(),
          effect: typeof effect === "string" ? effect : (effect.effect ?? ""),
          file: "",
        })),
      };
    });
    const initial =
      stateIDs.get(source.initial_state ?? "default") ??
      converted[0]?.uuid ??
      "";
    if (source.initial_state && !stateIDs.has(source.initial_state))
      warnings.add(
        `Controller ${name} has a missing initial state; Blockbench selects its first state.`,
      );
    return {
      uuid: id(),
      type: "animation_controller",
      name,
      saved: false,
      initial_state: initial,
      selected_state: null,
      states: converted,
    };
  });
  if (
    model.motion?.scripts.initialize ||
    model.motion?.scripts.pre_animation ||
    model.motion?.scripts.parent_setup
  )
    warnings.add(
      "Entity initialization and pre-animation scripts are retained as source data. Blockbench preview inputs are editable controls and do not reproduce Minecraft's entity runtime.",
    );
  if (
    animations.some((animation) =>
      animation.animators.effects?.keyframes.some(
        (key: Data) => key.channel === "sound" || key.channel === "particle",
      ),
    ) ||
    controllers.some((controller) =>
      controller.states.some(
        (state) => state.particles.length || state.sounds.length,
      ),
    )
  )
    warnings.add(
      "Sound and particle event tracks are editable; their Minecraft resources and identifiers are retained in source data rather than linked to external editor files.",
    );
  return {
    animations,
    controllers,
    placeholders: previewInputs(model),
    states: stateCount,
    keyframes: keyCount,
  };
}
