import FeedbackFlyout from "./FeedbackFlyout";
import {
  lazy,
  Suspense,
  useEffect,
  useState,
  useRef,
  type Dispatch,
  type SetStateAction,
} from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  api,
  type LibraryAsset,
  type PackObject,
  type PackObjectPreview,
} from "./bridge";
import type { SkinPose } from "./asset-geometry";
import type { SkinRigScene } from "./AssetViewport";
const SkinRigEditor = lazy(() => import("./SkinRigEditor"));

export default function SkinControls(p: {
  pose: SkinPose;
  setPose: (v: SkinPose) => void;
  walking: boolean;
  setWalking: (v: boolean) => void;
  packs: LibraryAsset[];
  attachments: PackObjectPreview[];
  setAttachments: Dispatch<SetStateAction<PackObjectPreview[]>>;
  geometry: unknown;
  preview: SkinRigScene;
  effects: boolean;
  setEffects: (value: boolean) => void;
}) {
  const request = useRef(0);
  useEffect(
    () => () => {
      request.current++;
    },
    [],
  );
  const [open, setOpen] = useState(false);
  const [pack, setPack] = useState(""),
    [objects, setObjects] = useState<PackObject[]>([]);
  const [chosen, setChosen] = useState(""),
    [search, setSearch] = useState("");
  const [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  useEffect(() => {
    request.current++;
    let dead = false;
    setObjects([]);
    setChosen("");
    setError("");
    setLoading(false);
    if (pack) {
      setLoading(true);
      api
        .GetAssetObjects(pack)
        .then((values) => {
          if (!dead)
            setObjects(values.filter((v) => v.geometryId && v.geometryFile));
        })
        .catch((e) => {
          if (!dead) setError(String(e));
        })
        .finally(() => {
          if (!dead) setLoading(false);
        });
    }
    return () => {
      dead = true;
    };
  }, [pack]);
  async function add() {
    const sequence = ++request.current;
    const object = objects.find((v) => v.id === chosen);
    if (!object) return;
    setLoading(true);
    setError("");
    try {
      const layers = objects.filter(
        (v) =>
          v.definition === object.definition &&
          (v.id === object.id ||
            (object.layers?.includes(object.geometryId) &&
              object.layers.includes(v.geometryId))),
      );
      if (layers.length > 8)
        throw new Error(
          "Choose a definition with up to eight geometry layers.",
        );
      if (
        p.attachments.filter((a) => a.object.definition !== object.definition)
          .length +
          layers.length >
        16
      )
        throw new Error(
          "Remove a cosmetic before adding more than sixteen layers.",
        );
      const previews: PackObjectPreview[] = [];
      for (const layer of layers) {
        const frame =
          /colou?r/i.test(layer.geometryId) && layer.textureOptions.frame_1
            ? "frame_1"
            : "";
        const value = await api.PreviewAssetObject(pack, layer.id, frame);
        if (!value.geometry || !value.image)
          throw new Error(
            "The model needs geometry and a readable texture. Restore the pack from a saved capture first.",
          );
        previews.push(value);
      }
      if (sequence !== request.current) return;
      p.setAttachments((old) => [
        ...old.filter((a) => a.object.definition !== object.definition),
        ...previews,
      ]);
    } catch (e) {
      if (sequence === request.current) setError(String(e));
    } finally {
      if (sequence === request.current) setLoading(false);
    }
  }
  return (
    <>
      <div className="skin-controls-summary">
        <Button variant="outline" onClick={() => setOpen(true)}>
          Pose & cosmetics…
        </Button>
        <span>
          {p.walking ? "Walking" : "Custom pose"} · {p.attachments.length}{" "}
          cosmetic layers
        </span>
      </div>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="skin-controls-dialog">
          <DialogHeader>
            <DialogTitle>Pose & cosmetics</DialogTitle>
            <DialogDescription>
              Adjust the captured model and add cosmetics from your resource
              packs. Changes are included in the model export.
            </DialogDescription>
          </DialogHeader>
          <Suspense fallback={<p role="status">Preparing character rig...</p>}>
            <SkinRigEditor
              scene={p.preview}
              pose={p.pose}
              setPose={p.setPose}
              walking={p.walking}
              setWalking={p.setWalking}
              attachments={p.attachments}
              effects={p.effects}
              setEffects={p.setEffects}
            >
              <div className="skin-rig-cosmetics">
                <h3>Pack cosmetics</h3>
                <p>
                  Captured equipment loads automatically when its resource pack
                  is available. For older captures, choose the cosmetic here.
                  Only layers used together by the model are added together.
                </p>
                <label>
                  Resource pack
                  <select
                    value={pack}
                    onChange={(e) => setPack(e.target.value)}
                  >
                    <option value="">Choose a readable pack</option>
                    {p.packs
                      .filter((v) => !v.encrypted)
                      .map((v) => (
                        <option key={v.id} value={v.id}>
                          {v.name}
                        </option>
                      ))}
                  </select>
                </label>
                <label>
                  Find a cosmetic
                  <input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Crown, ears, hat…"
                  />
                </label>
                <label>
                  Model
                  <select
                    value={chosen}
                    onChange={(e) => setChosen(e.target.value)}
                  >
                    <option value="">Choose a model</option>
                    {objects
                      .filter((v) =>
                        `${v.name} ${v.geometryId}`
                          .toLowerCase()
                          .includes(search.toLowerCase()),
                      )
                      .slice(0, 300)
                      .map((v) => (
                        <option value={v.id} key={v.id}>
                          {v.name}
                        </option>
                      ))}
                  </select>
                </label>
                <Button
                  disabled={loading || !chosen || p.attachments.length >= 16}
                  onClick={() => void add()}
                >
                  {loading ? "Loading…" : "Add cosmetic"}
                </Button>
                {error && <FeedbackFlyout title="Skin appearance">{error}</FeedbackFlyout>}
                {p.attachments.map((a, i) => (
                  <div
                    className="skin-cosmetic-row"
                    key={`${a.object.id}-${i}`}
                  >
                    <span>{a.object.name}</span>
                    <Button
                      variant="outline"
                      onClick={() =>
                        p.setAttachments(
                          p.attachments.filter((_, n) => n !== i),
                        )
                      }
                    >
                      Remove
                    </Button>
                  </div>
                ))}
                <p>
                  Cosmetics follow the body rig. Supported pack animations,
                  texture sequences and particle effects play with the preview.
                </p>
              </div>
            </SkinRigEditor>
          </Suspense>
        </DialogContent>
      </Dialog>
    </>
  );
}
