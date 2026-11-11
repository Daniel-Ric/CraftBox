import type { AssetFile, AssetFilePreview } from "./bridge";
import type { BlockbenchModel } from "./blockbench-project";

async function pngTexture(source: string) {
  if (source.startsWith("data:image/png;base64,")) return source;
  if (!/^data:image\/(?:jpeg|webp);base64,/.test(source))
    throw Error("Model texture must be an embedded image.");
  const image = new Image();
  image.src = source;
  await image.decode();
  if (image.naturalWidth * image.naturalHeight > 16 * 1024 * 1024)
    throw Error("Model texture exceeds 16 megapixels.");
  const canvas = document.createElement("canvas");
  canvas.width = image.naturalWidth;
  canvas.height = image.naturalHeight;
  const context = canvas.getContext("2d");
  if (!context) throw Error("Texture conversion is unavailable.");
  context.drawImage(image, 0, 0);
  return canvas.toDataURL("image/png");
}

export async function prepareBlockbenchSource(
  model: BlockbenchModel,
  files: AssetFile[],
  read: (path: string) => Promise<AssetFilePreview>,
): Promise<BlockbenchModel> {
  const warnings = new Set(model.warnings);
  const sources: Record<string, unknown> = { ...model.sources };
  const animations: Record<string, unknown> = { ...model.animations };
  const controllers: Record<string, any> = { ...model.motion?.controllers };
  const paths = new Set(
    [
      model.object?.definition,
      model.object?.geometryFile,
      ...Object.values(model.object?.animations ?? {}),
    ].filter((path): path is string => !!path && path.endsWith(".json")),
  );
  if (paths.size > 256)
    throw Error(
      "Model sources contain more than 256 animation and definition files.",
    );
  let bytes = 0;
  for (const path of paths) {
    if (sources[path] !== undefined) continue;
    const preview = await read(path);
    bytes += preview.data.length;
    if (bytes > 32 * 1024 * 1024)
      throw Error("Model source definitions exceed 32 MiB.");
    if (preview.truncated) throw Error(`Model source is truncated: ${path}`);
    const doc = JSON.parse(preview.data);
    sources[path] = doc;
  }
  const records = Object.values(sources).filter(
    (value): value is Record<string, any> =>
      !!value && typeof value === "object" && !Array.isArray(value),
  );
  for (const document of records) {
    Object.assign(animations, document.animations);
    Object.assign(controllers, document.animation_controllers);
  }
  Object.assign(animations, model.motion?.animations, model.animations);
  Object.assign(controllers, model.motion?.controllers);
  const definition = sources[model.object?.definition ?? ""] as
    Record<string, any> | undefined;
  const description =
    definition?.["minecraft:client_entity"]?.description ??
    definition?.["minecraft:attachable"]?.description ??
    {};
  const aliases: Record<string, string> = {
    ...description.animations,
    ...model.motion?.aliases,
  };
  const aliasIndex = new Map(
    Object.entries(aliases)
      .filter((entry) => typeof entry[1] === "string")
      .map(([name, ref]) => [name.toLowerCase(), ref]),
  );
  const sourceNames = new Map(
    [...Object.keys(animations), ...Object.keys(controllers)].map((name) => [
      name.toLowerCase(),
      name,
    ]),
  );
  const scripts = { ...description.scripts, ...model.motion?.scripts };
  const references = new Set<string>([
    ...Object.keys(model.object?.animations ?? {}),
    ...aliasIndex.values(),
    ...Object.keys(model.motion?.animations ?? {}),
    ...Object.keys(model.animations ?? {}),
  ]);
  for (const ref of scripts.animate ?? [])
    for (const name of typeof ref === "string" ? [ref] : Object.keys(ref))
      references.add(name);
  for (const ref of description.animation_controllers ?? [])
    for (const name of typeof ref === "string" ? [ref] : Object.values(ref))
      if (typeof name === "string") references.add(name);
  const visited = new Set<string>();
  const selected: Record<string, unknown> = {};
  const selectedControllers: Record<string, any> = {};
  const pending = [...references];
  while (pending.length) {
    const reference = pending.pop()!;
    let name = reference;
    const aliasChain = new Set<string>();
    while (true) {
      const key = name.toLowerCase(),
        target = aliasIndex.get(key);
      if (!target || target === name) break;
      if (aliasChain.has(key)) {
        warnings.add(`Cyclic animation alias: ${reference}`);
        break;
      }
      aliasChain.add(key);
      name = target;
    }
    name = sourceNames.get(name.toLowerCase()) ?? name;
    if (visited.has(name.toLowerCase())) continue;
    visited.add(name.toLowerCase());
    const animation = animations[name];
    if (animation) selected[name] = animation;
    const controller = controllers[name];
    if (!animation && !controller)
      warnings.add(
        `Animation source is not present in the captured pack: ${name}`,
      );
    if (!controller) continue;
    selectedControllers[name] = controller;
    for (const state of Object.values(controller.states ?? {}) as any[])
      for (const entry of state.animations ?? [])
        pending.push(
          ...(typeof entry === "string" ? [entry] : Object.keys(entry)),
        );
  }
  const textures = { ...model.textures };
  const textureNames = { ...model.textureNames };
  const loaded = new Map<string, string>();
  for (const [alias, path] of Object.entries(
    model.object?.textureOptions ?? {},
  )) {
    if (!path || !files.some((file) => file.path === path)) continue;
    let data = loaded.get(path);
    if (!data) {
      const preview = await read(path);
      if (preview.truncated) throw Error(`Model texture is truncated: ${path}`);
      data = preview.data;
      loaded.set(path, data);
    }
    const key = `variant:${alias}`;
    textures[key] = data;
    textureNames[key] = path.split("/").pop() || alias;
  }
  const converted = new Map<string, string>();
  const originals: Record<string, string> = {};
  for (const [alias, source] of Object.entries(textures)) {
    if (!source || source.startsWith("data:image/png;base64,")) continue;
    originals[alias] = source;
    let png = converted.get(source);
    if (!png) {
      png = await pngTexture(source);
      converted.set(source, png);
    }
    textures[alias] = png;
  }
  if (Object.keys(originals).length)
    sources["craftbox:original-textures"] = originals;
  return {
    ...model,
    textures,
    textureNames,
    sources: { ...model.sources, ...sources },
    warnings: [...warnings],
    animations: { ...selected, ...model.animations },
    motion: {
      particles: {},
      hideCape: false,
      ...model.motion,
      aliases,
      scripts,
      animations: selected,
      controllers: selectedControllers,
    },
  };
}

export function blockbenchProjectData(project: unknown) {
  const bytes = new TextEncoder().encode(JSON.stringify(project));
  let text = "";
  for (let i = 0; i < bytes.length; i += 8192)
    text += String.fromCharCode(...bytes.subarray(i, i + 8192));
  return btoa(text);
}
