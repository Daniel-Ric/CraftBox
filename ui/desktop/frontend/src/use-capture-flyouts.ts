import { useEffect, useRef } from "react";
import { api, type Pack, type World } from "./bridge";
import { notify } from "./StudioFlyouts";
export function useCaptureFlyouts(
  worlds: World[],
  packs: Pack[],
  enabled = true,
) {
  const previous = useRef<{
    worlds: Map<string, string>;
    packs: Map<string, Pack>;
  }>({ worlds: new Map(), packs: new Map() });
  useEffect(() => {
    if (enabled) {
      for (const world of worlds)
        if (
          world.state === "finished" &&
          previous.current.worlds.has(world.id) &&
          previous.current.worlds.get(world.id) !== "finished"
        )
          notify({
            id: "world-finished:" + world.id,
            tone: "success",
            title: "World capture saved",
            description: world.name,
            action: world.path
              ? {
                  label: "Open folder",
                  onClick: () => api.OpenResultFolder(world.path),
                }
              : undefined,
          });
      for (const pack of packs) {
        const old = previous.current.packs.get(pack.id);
        if (!old) continue;
        if (pack.error && pack.error !== old.error)
          notify({
            id: "pack:" + pack.id,
            tone: "error",
            title: "Resource pack download failed",
            description: pack.name + ": " + pack.error,
          });
        else if (pack.done && !old.done && !pack.error)
          notify({
            id: "pack:" + pack.id,
            tone: "success",
            title: "Resource pack downloaded",
            description: pack.name,
            action: pack.path
              ? {
                  label: "Open folder",
                  onClick: () => api.OpenResultFolder(pack.path),
                }
              : undefined,
          });
      }
    }
    previous.current = {
      worlds: new Map(worlds.map((w) => [w.id, w.state])),
      packs: new Map(packs.map((p) => [p.id, { ...p }])),
    };
  }, [worlds, packs, enabled]);
}
