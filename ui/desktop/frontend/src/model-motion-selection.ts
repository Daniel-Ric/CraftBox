import type { BlockbenchModel } from "./blockbench-project";

type Data = Record<string, any>;
const entries = (value: unknown): [string, any][] => value && typeof value === "object" && !Array.isArray(value) ? Object.entries(value) : [];

export function selectModelMotion(model: BlockbenchModel, bones: Set<string>) {
  const documents = Object.values(model.sources ?? {}).filter((value): value is Data => !!value && typeof value === "object");
  const descriptions = documents.flatMap(doc => [doc["minecraft:client_entity"]?.description, doc["minecraft:attachable"]?.description].filter(Boolean)) as Data[];
  const aliases: Record<string, string> = Object.assign({}, ...descriptions.map(description => description.animations), model.motion?.aliases);
  const active = new Set<string>();
  const animations = (value: unknown) => Object.fromEntries(entries(value).flatMap(([name, source]) => {
    const animation = structuredClone(source) as Data;
    if (!animation || typeof animation !== "object") return [];
    const originalBones = entries(animation.bones);
    if (originalBones.length) {
      animation.bones = Object.fromEntries(originalBones.filter(([name]) => bones.has(name.toLowerCase())));
      if (!Object.keys(animation.bones).length) return [];
    } else if (!animation.particle_effects && !animation.sound_effects && !animation.timeline) return [];
    active.add(name.toLowerCase());
    return [[name, animation]];
  }));
  if (model.animations) model.animations = animations(model.animations);
  if (model.motion) model.motion.animations = animations(model.motion.animations);
  for (const doc of documents) if (doc.animations) doc.animations = animations(doc.animations);
  const controllers = Object.assign({}, ...documents.map(doc => doc.animation_controllers), model.motion?.controllers) as Record<string, Data>;
  const has = (name: unknown) => typeof name === "string" && active.has(name.toLowerCase());
  const refs = (value: unknown) => typeof value === "string" ? [value] : entries(value).map(([name]) => name);
  let changed = true;
  while (changed) {
    changed = false;
    const add = (name: string) => { if (!has(name)) { active.add(name.toLowerCase()); changed = true; } };
    for (const [name, target] of entries(aliases)) if (has(target)) add(name);
    for (const [name, controller] of Object.entries(controllers)) {
      const states = entries(controller.states).map(([, state]) => state as Data);
      if (states.some(state => (state.animations ?? []).some((value: unknown) => refs(value).some(has)) || state.on_entry?.length || state.on_exit?.length)) add(name);
    }
  }
  const filterRefs = (values: unknown) => (Array.isArray(values) ? values : []).flatMap<string | Data>(value => {
    if (typeof value === "string") return has(value) ? [value] : [];
    const retained = Object.fromEntries(entries(value).filter(([name]) => has(name)));
    return Object.keys(retained).length ? [retained] : [];
  });
  const filterControllers = (value: unknown) => Object.fromEntries(entries(value).filter(([name]) => has(name)).map(([name, source]) => {
    const controller = structuredClone(source) as Data;
    for (const [, state] of entries(controller.states)) if (state.animations) state.animations = filterRefs(state.animations);
    return [name, controller];
  }));
  const filterAliases = (value: unknown) => Object.fromEntries(entries(value).filter(([, target]) => has(target)));
  const filterScripts = (scripts: Data | undefined) => {
    if (scripts?.animate) scripts.animate = filterRefs(scripts.animate);
  };
  if (model.motion) {
    model.motion.controllers = filterControllers(model.motion.controllers);
    model.motion.aliases = filterAliases(model.motion.aliases);
    filterScripts(model.motion.scripts);
  }
  for (const doc of documents) if (doc.animation_controllers) doc.animation_controllers = filterControllers(doc.animation_controllers);
  for (const description of descriptions) {
    if (description.animations) description.animations = filterAliases(description.animations);
    if (description.animation_controllers) description.animation_controllers = description.animation_controllers.flatMap((value: unknown) => {
      const retained = Object.fromEntries(entries(value).filter(([, target]) => has(target)));
      return Object.keys(retained).length ? [retained] : [];
    });
    filterScripts(description.scripts);
  }
  if (model.object?.animations) model.object.animations = Object.fromEntries(entries(model.object.animations).filter(([name]) => has(name)));
}
