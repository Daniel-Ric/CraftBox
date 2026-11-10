import HeaderActionIcon from "./HeaderActionIcon";
import CollectionToolbar from "./CollectionToolbar";
import {collectionSelection} from "./collection-selection";
import FeedbackFlyout from "./FeedbackFlyout";
import { DocumentScope, useDocumentScope } from "./document-scope";
import { useWorkspaceDocument } from "./use-workspace-document";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useWorkspaceActions } from "./workspace-actions";
import { openMarketplace, type MarketFavorite } from "./marketplace-model";
import { openPacketEvidence } from "./packet-navigation";
import {
  ArrowDown,
  ArrowUp,
  Box,
  CheckCircle2,
  Download,
  FolderOpen,
  Import,
  Layers,
  LoaderCircle,
  RefreshCw,
  Search,
  Shirt,
  Star,
  Trash2,
  TriangleAlert,
  Gamepad2,
  ImageDown,
  LayoutGrid,
  List,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  api,
  on,
  type AssetLibraryState,
  type LibraryAsset,
  type AssetDetail,
  type AssetFilePreview,
  type AssetStack,
  type AssetChange,
  type MinecraftInstallation,
  type PackObjectPreview,
} from "./bridge";
import type {
  AssetViewportHandle,
  MaterialMaps,
  SkinTextureAnimation,
} from "./AssetViewport";
import {
  neutralPose,
  selectSkinGeometry,
  type SkinPose,
} from "./asset-geometry";
import { notify, notifySaved, notifyFailure } from "./StudioFlyouts";
import TextureEditor from "./TextureEditor";
import PackContentPicker from "./PackContentPicker";
import { packObjectKind, packObjectLabel } from "./pack-content-model";
import { createBlockbenchProject } from "./blockbench-project";
import {
  blockbenchProjectData,
  prepareBlockbenchSource,
} from "./blockbench-source";
import { groupAudioFiles } from "./audio-model";
import SkinControls from "./SkinControls";
import WireframeControls from "./WireframeControls";
import type { WireframeSettings, WireframeStats } from "./wireframe-settings";
import AssetThumbnail from "./AssetThumbnail";
import {
  cachedAssetLibrary,
  invalidateAssetLibrary,
  refreshAssetLibrary,
} from "./asset-library-state";
import TextureViewport, { type TextureViewportHandle } from "./TextureViewport";
import "./asset-studio.css";
import "./audio-studio.css";
const AssetViewport = lazy(() => import("./AssetViewport"));
const AudioStudio = lazy(() => import("./AudioStudio"));
const bytes = (n: number) =>
  n >= 1073741824
    ? `${(n / 1073741824).toFixed(2)} GB`
    : n >= 1048576
      ? `${(n / 1048576).toFixed(1)} MB`
      : `${Math.ceil(n / 1024)} KB`;
const err = (e: unknown) => (e instanceof Error ? e.message : String(e));
const appearanceRequests = new Map<
  string,
  ReturnType<typeof api.GetSkinAppearance>
>();

type Tab = "preview" | "files" | "checks" | "compare" | "stack" | "edit";

export default function AssetStudio({
  kind,
  onSource,
}: {
  kind: "skins" | "packs";
  onSource: () => void;
}) {
  const scope = useDocumentScope();
  const cachedLibrary = cachedAssetLibrary();
  const initialView = scope.initial?.view;
  const restorePending = useRef(!!initialView);
  const [objectSelection, setObjectSelection] = useState({
    object: typeof initialView?.object === "string" ? initialView.object : "",
    variant:
      typeof initialView?.variant === "string" ? initialView.variant : "",
  });
  const [restored, setRestored] = useState(!initialView);
  const [audioStates, setAudioStates] = useState<
    Record<string, import("./AudioStudio").AudioWorkspaceState>
  >(
    (initialView?.audioStates as Record<
      string,
      import("./AudioStudio").AudioWorkspaceState
    >) || {},
  );
  const [audioLoading, setAudioLoading] = useState(false);
  const [textureDrafts, setTextureDrafts] = useState<Record<string, string>>(
    (initialView?.textureDrafts as Record<string, string>) || {},
  );
  const [cameraView, setCameraView] = useState<any>(initialView?.camera);
  const [multiple,setMultiple]=useState(false),[selectionAnchor,setSelectionAnchor]=useState("");
  const [marketMatches, setMarketMatches] = useState<MarketFavorite[]>([]);
  const [library, setLibrary] = useState<AssetLibraryState>(
      cachedLibrary ?? {
        items: [],
        bytes: 0,
        directory: "",
      },
    ),
    [selected, setSelected] = useState<string[]>(
      scope.initial?.source.id || scope.sourceId
        ? [scope.initial?.source.id || scope.sourceId!]
        : [],
    ),
    [active, setActive] = useState(
      scope.initial?.source.id || scope.sourceId || "",
    ),
    [detail, setDetail] = useState<AssetDetail>(),
    [query, setQuery] = useState(""),
    [favourites, setFavourites] = useState(false),
    [sort, setSort] = useState("newest"),
    [tab, setTab] = useState<Tab>("preview"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(""),
    [loading, setLoading] = useState(!cachedLibrary);
  useEffect(() => {
    if (error)
      notifyFailure(
        error,
        kind === "packs" ? "Resource pack error" : "Player skin error",
        `asset-error:${scope.id || kind}`,
      );
  }, [error, kind, scope.id]);
  const [fileQuery, setFileQuery] = useState(""),
    [fileKind, setFileKind] = useState("all"),
    [filePage, setFilePage] = useState(0),
    [file, setFile] = useState(""),
    [preview, setPreview] = useState<AssetFilePreview>(),
    [geometry, setGeometry] = useState<unknown>(),
    [cape, setCape] = useState(""),
    [texture, setTexture] = useState(""),
    [maps, setMaps] = useState<MaterialMaps>(),
    [shape, setShape] = useState<"cube" | "sphere">("cube");
  const [audioSelection, setAudioSelection] = useState<
    import("./bridge").AssetFile[]
  >([]);
  const [groupSounds, setGroupSounds] = useState(() => {
    try {
      return localStorage.getItem("craftbox:group-audio") !== "false";
    } catch {
      return true;
    }
  });
  const textureViewport = useRef<TextureViewportHandle>(null);
  const [thumbnails, setThumbnails] = useState<Record<string, string>>(() => {
    try {
      return JSON.parse(
        localStorage.getItem("craftbox:asset-thumbnails-v5") || "{}",
      );
    } catch {
      return {};
    }
  });
  const [pngAsSkin, setPngAsSkin] = useState(false);
  const [cosmeticsLoading, setCosmeticsLoading] = useState(false);
  const [modelReady, setModelReady] = useState(false);
  const [effects, setEffects] = useState(true);
  const [showDisplays, setShowDisplays] = useState(false);
  const [browserView, setBrowserView] = useState<"list" | "grid">("list");
  const [missingGeometry, setMissingGeometry] = useState("");
  const [face, setFace] = useState<{
    geometry: unknown;
    texture: string;
    animation?: SkinTextureAnimation;
  }>();
  const [animatedLayers, setAnimatedLayers] = useState<
    { geometry: unknown; animation: SkinTextureAnimation }[]
  >([]);
  const [attachments, setAttachments] = useState<PackObjectPreview[]>([]);
  const [pose, setPose] = useState<SkinPose>({ ...neutralPose }),
    [walking, setWalking] = useState(false),
    [outer, setOuter] = useState(true),
    [showCape, setShowCape] = useState(true),
    [rotate, setRotate] = useState(false),
    [wireframe, setWireframe] = useState(false),
    [light, setLight] = useState(3),
    [useGeometry, setUseGeometry] = useState(true),
    [arm, setArm] = useState("auto");
  const [animation, setAnimation] = useState<unknown>(),
    [animationName, setAnimationName] = useState(""),
    [propertiesOpen, setPropertiesOpen] = useState(false);
  const [manualPreview, setManualPreview] = useState(false);
  const [wireframeStats, setWireframeStats] = useState<WireframeStats>();
  const [wireframeOptions, setWireframeOptions] = useState<WireframeSettings>(
    {},
  );
  const [binding, setBinding] = useState<PackObjectPreview>(),
    [contentLoading, setContentLoading] = useState(false),
    [materialTextures, setMaterialTextures] =
      useState<Record<string, string>>();
  const [installations, setInstallations] = useState<MinecraftInstallation[]>(
      [],
    ),
    [installation, setInstallation] = useState(""),
    [minecraftWarning, setMinecraftWarning] = useState(""),
    [minecraftOpen, setMinecraftOpen] = useState(false),
    [confirmDelete, setConfirmDelete] = useState<string[] | null>(null),
    [exportOpen, setExportOpen] = useState(false),
    [exportFormat, setExportFormat] = useState("mcpack"),
    [exportName, setExportName] = useState(""),
    [modelFormat, setModelFormat] = useState("glb"),
    [blockbenchFormat, setBlockbenchFormat] = useState<
      "auto" | "entity" | "block" | "free"
    >("auto");
  const [name, setName] = useState(""),
    [tags, setTags] = useState(""),
    [compareID, setCompareID] = useState(""),
    [changes, setChanges] = useState<AssetChange[]>([]),
    [compareImage, setCompareImage] = useState(""),
    [stackIDs, setStackIDs] = useState<string[]>([]),
    [stack, setStack] = useState<AssetStack>(),
    [stackFilter, setStackFilter] = useState(""),
    [textureZoom, setTextureZoom] = useState(1),
    [dragging, setDragging] = useState(false),
    [visibleCount, setVisibleCount] = useState(100);
  const viewport = useRef<AssetViewportHandle>(null),
    detailRequest = useRef(0),
    fileRequest = useRef(0),
    mounted = useRef(true);
  const current = library.items.find((v) => v.id === active),
    skin = kind === "skins";
  const hasModelPreview = skin || !!geometry || !!maps;
  const hasVisiblePreview = hasModelPreview || !!texture;
  useEffect(() => {
    try {
      const target = JSON.parse(
        sessionStorage.getItem("craftbox:packet-content") || "null",
      );
      if (
        scope.active &&
        target?.kind === kind &&
        library.items.some((v) => v.id === target.id)
      ) {
        setActive(target.id);
        sessionStorage.removeItem("craftbox:packet-content");
      }
    } catch {}
  }, [kind, library.items]);
  useEffect(() => {
    if (
      !restorePending.current ||
      !detail ||
      detail.item.id !== active ||
      cosmeticsLoading ||
      contentLoading
    )
      return;
    restorePending.current = false;
    const v = initialView!;
    void (async () => {
      try {
        if (v.manualPreview && typeof v.file === "string" && v.file)
          await openFile(v.file);
        if (v.camera) setCameraView(v.camera);
        if (Array.isArray(v.audioSelection))
          setAudioSelection(
            v.audioSelection
              .map((path) => detail.files.find((f) => f.path === path))
              .filter((f): f is import("./bridge").AssetFile => !!f),
          );
        if (v.pose && typeof v.pose === "object") setPose(v.pose as SkinPose);
        for (const [key, set] of Object.entries({
          outer: setOuter,
          showCape: setShowCape,
          rotate: setRotate,
          walking: setWalking,
          groupSounds: setGroupSounds,
          wireframe: setWireframe,
          useGeometry: setUseGeometry,
          effects: setEffects,
        }))
          if (typeof v[key] === "boolean") set(v[key] as boolean);
        if (typeof v.fileQuery === "string") setFileQuery(v.fileQuery);
        if (typeof v.fileKind === "string") setFileKind(v.fileKind);
        if (typeof v.textureZoom === "number") setTextureZoom(v.textureZoom);
        if (v.shape === "cube" || v.shape === "sphere") setShape(v.shape);
        if (typeof v.animationName === "string")
          setAnimationName(v.animationName);
        if (typeof v.light === "number" && Number.isFinite(v.light))
          setLight(Math.max(0, Math.min(20, v.light)));
        if (
          typeof v.arm === "string" &&
          ["auto", "slim", "wide"].includes(v.arm)
        )
          setArm(v.arm);
        if (v.wireframeOptions && typeof v.wireframeOptions === "object")
          setWireframeOptions(v.wireframeOptions as WireframeSettings);
        if (
          typeof v.tab === "string" &&
          ["preview", "files", "compare", "edit"].includes(v.tab)
        )
          setTab(v.tab as Tab);
      } catch (e) {
        setError(err(e));
      } finally {
        setRestored(true);
      }
    })();
  }, [detail, active, cosmeticsLoading, contentLoading]);
  useWorkspaceDocument(
    kind,
    { id: active, name: current?.name || "" },
    {
      pose,
      textureDrafts,
      audioStates,
      audioSelection: audioSelection.map((f) => f.path),
      outer,
      showCape,
      rotate,
      walking,
      wireframe,
      wireframeOptions,
      light,
      useGeometry,
      arm,
      effects,
      tab,
      file,
      object: objectSelection.object || binding?.object.id || "",
      variant: objectSelection.variant,
      blockStates:objectSelection.object === initialView?.object ? initialView?.blockStates : undefined,
      camera: cameraView,
      fileQuery,
      fileKind,
      textureZoom,
      manualPreview,
      shape,
      animationName,
      groupSounds,
    },
    !!busy,
    !!detail &&
      !cosmeticsLoading &&
      !contentLoading &&
      !audioLoading &&
      restored,
  );
  const blockRefresh = useRef({ asset: "", revision: "" });
  useEffect(() => {
    const previous = blockRefresh.current,
      revision = library.blockRevisions?.[active] ?? "";
    if (previous.asset !== active) {
      blockRefresh.current = { asset: active, revision };
      return;
    }
    if (
      skin ||
      !active ||
      detail?.item.id !== active ||
      previous.revision === revision
    )
      return;
    blockRefresh.current = { asset: active, revision };
    const run = ++detailRequest.current;
    void api
      .GetAssetDetail(active)
      .then((value) => {
        if (run === detailRequest.current) setDetail(value);
      })
      .catch((e) => {
        if (run === detailRequest.current) setError(err(e));
      });
  }, [active, skin, library.blockRevisions?.[active], detail?.item.id]);
  const refresh = useCallback(async (fresh = true) => {
    if (fresh) invalidateAssetLibrary();
    const result = await refreshAssetLibrary();
    if (mounted.current) {
      setLibrary(result);
      const available = new Set(result.items.map((item) => item.id));
      setSelected((old) =>
        old.filter((id) => available.has(id)),
      );
      setLoading(false);
    }
    return result;
  }, []);
  useEffect(() => {
    mounted.current = true;
    void refresh(false).catch((e) => {
      setError(err(e));
      setLoading(false);
    });
    let timer = 0;
    const off = on("desktop:assets", () => {
      invalidateAssetLibrary();
      window.clearTimeout(timer);
      timer = window.setTimeout(
        () => void refresh(false).catch((e) => setError(err(e))),
        200,
      );
    });
    return () => {
      mounted.current = false;
      window.clearTimeout(timer);
      off();
    };
  }, [refresh]);
  const run = useCallback(
    async (label: string, job: () => Promise<unknown>) => {
      setError("");
      setBusy(label);
      try {
        await job();
      } catch (e) {
        if (mounted.current) setError(err(e));
      } finally {
        if (mounted.current) setBusy("");
      }
    },
    [],
  );
  useEffect(() => {
    let dead = false;
    setMarketMatches([]);
    if (active)
      void api
        .GetMarketplaceAssetMatches(active)
        .then((v) => {
          if (!dead) setMarketMatches(v);
        })
        .catch(() => {});
    return () => {
      dead = true;
    };
  }, [active]);
  async function minecraftRefresh() {
    const result = await api.GetMinecraftInstallations();
    if (!mounted.current) return;
    setInstallations(result.installations);
    setMinecraftWarning(result.warning);
    setInstallation((old) =>
      result.installations.some((v) => v.id === old)
        ? old
        : (result.installations[0]?.id ?? ""),
    );
  }
  useEffect(() => {
    if (minecraftOpen) void run("Finding Minecraft", minecraftRefresh);
  }, [minecraftOpen]);
  const representatives = useMemo(() => {
    const preferred = new Map<string, LibraryAsset>();
    for (const item of library.items) {
      if (
        item.kind !== "skins" ||
        !item.playerUUID ||
        item.captureType === "display"
      )
        continue;
      const previous = preferred.get(item.playerUUID);
      if (
        !previous ||
        (previous.persona && !item.persona) ||
        (previous.persona === item.persona && item.created > previous.created)
      )
        preferred.set(item.playerUUID, item);
    }
    return preferred;
  }, [library.items]);
  const variants = current?.playerUUID
    ? library.items
        .filter(
          (v) =>
            v.kind === "skins" &&
            v.playerUUID === current.playerUUID &&
            v.captureType !== "display",
        )
        .sort(
          (a, b) =>
            Number(a.persona) - Number(b.persona) ||
            b.created.localeCompare(a.created),
        )
    : [];
  const itemCount = library.items.filter(
    (v) =>
      v.kind === kind &&
      (!skin ||
        !v.playerUUID ||
        v.captureType === "display" ||
        representatives.get(v.playerUUID)?.id === v.id),
  ).length;
  const list = useMemo(
    () =>
      library.items
        .filter(
          (v) =>
            v.kind === kind &&
            (!skin ||
              !v.playerUUID ||
              v.captureType === "display" ||
              representatives.get(v.playerUUID)?.id === v.id) &&
            (!skin || showDisplays || v.captureType !== "display") &&
            (!favourites || v.favourite) &&
            `${v.name} ${v.source} ${v.version} ${v.tags.join(" ")}`
              .toLowerCase()
              .includes(query.toLowerCase()),
        )
        .sort((a, b) =>
          sort === "name"
            ? a.name.localeCompare(b.name)
            : sort === "size"
              ? b.bytes - a.bytes
              : b.created.localeCompare(a.created),
        ),
    [
      library.items,
      kind,
      favourites,
      query,
      sort,
      skin,
      showDisplays,
      representatives,
    ],
  );
  useEffect(() => {
    setVisibleCount(100);
  }, [query, sort, favourites]);
  useEffect(() => {
    if (!scope.onOpen && !active && list.length) setActive(list[0].id);
    if (!scope.onOpen && active && !library.items.some((v) => v.id === active))
      setActive(list[0]?.id ?? "");
  }, [list, active, library.items]);
  useEffect(() => {
    const sequence = ++detailRequest.current;
    setError("");
    ++fileRequest.current;
    setDetail(undefined);
    setPreview(undefined);
    setAudioSelection([]);
    setGeometry(undefined);
    setAttachments([]);
    setFace(undefined);
    setAnimatedLayers([]);
    setModelReady(false);
    setCosmeticsLoading(skin);
    setPngAsSkin(false);
    setMissingGeometry("");
    setBinding(undefined);
    setManualPreview(false);
    setMaterialTextures(undefined);
    setAnimation(undefined);
    setAnimationName("");
    setCape("");
    setTexture("");
    setMaps(undefined);
    setFile("");
    setPose({ ...neutralPose });
    setWalking(false);
    setUseGeometry(true);
    setArm("auto");
    setCompareID("");
    setChanges([]);
    setCompareImage("");
    if (!active) return;
    let dead = false;
    void (async () => {
      try {
        const d = await (initialView?.picked && !skin ? api.GetPreparedAssetDetail(active) : api.GetAssetDetail(active));
        if (dead || sequence !== detailRequest.current) return;
        setDetail(d);
        if(d.partial)setLibrary(old=>({...old,items:old.items.some(item=>item.id===d.item.id)?old.items:[...old.items,d.item]}));
        setName(d.item.name);
        setTags(d.item.tags.join(", "));
        setExportName(d.item.name);
        setTexture(d.item.kind === "skins" ? d.item.image : "");
        if (d.item.kind === "skins") {
          {
            try {
              const p = d.files.some((f) => f.path === "geometry.json")
                ? await api.ReadAssetFile(active, "geometry.json")
                : { data: "{}" };
              if (!dead) {
                const raw = JSON.parse(p.data);
                let skinAnimations: SkinTextureAnimation[] = [];
                if (d.files.some((f) => f.path === "animations.json"))
                  try {
                    skinAnimations = JSON.parse(
                      (await api.ReadAssetFile(active, "animations.json")).data,
                    );
                  } catch {}
                let identifier = d.metadata.geometryId;
                if (
                  !identifier &&
                  typeof d.metadata.resourcePatch === "string"
                ) {
                  try {
                    identifier = JSON.parse(atob(d.metadata.resourcePatch))
                      .geometry?.default;
                  } catch {}
                }
                let chosen = selectSkinGeometry(
                  raw,
                  typeof identifier === "string" ? identifier : undefined,
                );
                if (
                  !chosen &&
                  typeof identifier === "string" &&
                  !/^geometry\.humanoid\.(custom|customSlim)$/.test(identifier)
                ) {
                  setMissingGeometry(identifier);
                }
                if (!dead) setGeometry(chosen);
                if (typeof d.metadata.resourcePatch === "string") {
                  const faceID = JSON.parse(atob(d.metadata.resourcePatch))
                    .geometry?.animated_face;
                  if (typeof faceID === "string") {
                    const faceGeometry = selectSkinGeometry(raw, faceID);
                    if (
                      faceGeometry &&
                      d.files.some((v) => v.path === "face.png")
                    ) {
                      const image = await api.ReadAssetFile(active, "face.png");
                      if (!dead)
                        setFace({
                          geometry: faceGeometry,
                          texture: image.data,
                          animation: skinAnimations.find(
                            (a) => a.AnimationType === 1,
                          ),
                        });
                    } else if (d.metadata.persona && !dead)
                      setError(
                        "This Character Creator capture is missing its separate face texture. Capture it again with this build to include the face image.",
                      );
                  }
                  const patch = JSON.parse(atob(d.metadata.resourcePatch));
                  const layers: {
                    geometry: unknown;
                    animation: SkinTextureAnimation;
                  }[] = [];
                  for (const animation of skinAnimations) {
                    const key =
                      animation.AnimationType === 2
                        ? "animated_32x32"
                        : animation.AnimationType === 3
                          ? "animated_128x128"
                          : "";
                    if (!key || typeof patch.geometry?.[key] !== "string")
                      continue;
                    const geometry = selectSkinGeometry(
                      raw,
                      patch.geometry[key],
                    );
                    if (geometry) layers.push({ geometry, animation });
                  }
                  if (!dead) setAnimatedLayers(layers);
                }
              }
            } catch (e) {
              if (!dead)
                setError("Custom geometry could not be loaded: " + err(e));
            }
          }
          if (d.files.some((f) => f.path === "cape.png")) {
            const p = await api.ReadAssetFile(active, "cape.png");
            if (!dead) setCape(p.data);
          }
        }
        if (!dead) setModelReady(true);
      } catch (e) {
        if (!dead) setError(err(e));
      }
    })();
    return () => {
      dead = true;
    };
  }, [active]);
  useEffect(()=>{
    if(!detail?.partial || (!binding && tab!=="files"))return;
    let alive=true;
    void api.GetAssetDetail(active).then(value=>{if(alive&&value.item.id===active)setDetail(value)}).catch(e=>{if(alive)setError(err(e))});
    return()=>{alive=false};
  },[active,detail?.partial,binding,tab]);
  const readablePacksKey = library.items
    .filter((v) => v.kind === "packs" && !v.encrypted)
    .map((v) => v.id)
    .join(",");
  useEffect(() => {
    if (!missingGeometry) return;
    let cancelled = false;
    void (async () => {
      for (const id of readablePacksKey
        .split(",")
        .filter(Boolean)
        .slice(0, 64)) {
        if (cancelled) return;
        try {
          const objects = await api.GetAssetObjects(id);
          const match = objects.find(
            (v) => v.geometryId === missingGeometry && v.geometryFile,
          );
          if (!match) continue;
          const preview = await api.PreviewAssetObject(id, match.id, "");
          if (preview.geometry && !cancelled) {
            setGeometry(preview.geometry);
            setError("");
            return;
          }
        } catch {}
      }
      if (!cancelled)
        setError(
          "The skin references " +
            missingGeometry +
            ", but its model is missing. Import or restore the matching resource pack. A standard body is shown until the model is available.",
        );
    })();
    return () => {
      cancelled = true;
    };
  }, [active, missingGeometry, readablePacksKey]);
  useEffect(() => {
    if (!active || !skin || !detail || detail.item.id !== active) return;
    let cancelled = false;
    const key =
      (current?.playerUUID || active) +
      ":" +
      readablePacksKey +
      ":" +
      (library.equipmentRevision || "");
    let request = appearanceRequests.get(key);
    if (!request) {
      if (appearanceRequests.size > 32) appearanceRequests.clear();
      request = api.GetSkinAppearance(active);
      appearanceRequests.set(key, request);
      request.catch(() => appearanceRequests.delete(key));
    }
    setCosmeticsLoading(true);
    void request
      .then((result) => {
        if (cancelled) return;
        setAttachments((old) => [
          ...old.filter(
            (v) =>
              !v.capturedSlot &&
              !result.attachments.some((a) => a.object.id === v.object.id),
          ),
          ...result.attachments,
        ]);
        if (result.warnings.length) setError(result.warnings.join(". "));
      })
      .catch((e) => {
        if (!cancelled) setError(err(e));
      })
      .finally(() => {
        if (!cancelled) setCosmeticsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [active, skin, detail, readablePacksKey, library.equipmentRevision]);
  async function acceptImports(items: LibraryAsset[]) {
    await refresh();
    if (items.length) {
      const matching = items.filter((i) => i.kind === kind);
      if (matching.length) {
        if (scope.onOpen)
          scope.onOpen(matching[0].id, matching[0].name, true, true);
        else setActive(matching[0].id);
        setSelected(matching.map((i) => i.id));
      }
      notify({
        tone: "success",
        title: `${items.length} ${items.length === 1 ? "item" : "items"} added to the library`,
      });
    }
  }
  useWorkspaceActions(kind, (action) => {
    if (busy) {
      setError("Wait for the current library operation to finish.");
      return;
    }
    switch (action) {
      case "import-files":
        void run("Importing files", async () =>
          acceptImports(await api.ImportAssets(kind, [])),
        );
        break;
      case "import-folder":
        void run("Importing folder", async () =>
          acceptImports(await api.ImportAssetFolder(kind)),
        );
        break;
      case "browse-minecraft":
        setMinecraftOpen(true);
        break;
      case "export":
        if (selected.length || active) setExportOpen(true);
        else setError("Select a library item to export.");
        break;
      case "refresh":
        void run("Refreshing library", refresh);
        break;
      case "files":
        setTab("files");
        break;
      case "list":
        setBrowserView("list");
        break;
      case "grid":
        setBrowserView("grid");
        break;
      case "search":
        document
          .querySelector<HTMLInputElement>(".asset-library input")
          ?.focus();
        break;
    }
  });
  async function openFile(path: string) {
    setCameraView(undefined);
    setAudioSelection([]);
    setManualPreview(true);
    setContentLoading(false);
    const seq = ++fileRequest.current;
    setFile(path);
    setPreview(undefined);
    setError("");
    try {
      const p = await api.ReadAssetFile(active, path);
      if (seq !== fileRequest.current) return;
      setPreview(p);
      if (p.kind === "sound")
        setAudioSelection(detail?.files.filter((f) => f.path === path) ?? []);
      if (
        p.kind === "texture" &&
        (!skin || path === detail?.item.texturePath)
      ) {
        setTexture(p.data);
        setMaps(undefined);
        setTextureZoom(1);
        if (!skin) {
          setGeometry(undefined);
          setBinding(undefined);
          setMaterialTextures(undefined);
        }
      }
      if (p.kind === "model") {
        setGeometry(JSON.parse(p.data));
        if (!skin) {
          const objects = await api.GetAssetObjects(active);
          const match =
            objects.find(
              (v) =>
                v.geometryFile === path && v.geometryId === binding?.geometryId,
            ) ??
            objects.find(
              (v) => v.geometryFile === path && v.kind !== "geometry",
            ) ??
            objects.find((v) => v.geometryFile === path);
          const value = match
            ? await api.PreviewAssetObject(active, match.id, "")
            : undefined;
          if (seq !== fileRequest.current) return;
          setBinding(value);
          setTexture(value?.image ?? "");
          setMaterialTextures(value?.textures);
          setMaps(undefined);
          if (value?.geometry) setGeometry(value.geometry);
        }
      }
      if (p.kind === "animation") {
        const raw = JSON.parse(p.data);
        setAnimation(raw);
        setAnimationName(Object.keys(raw.animations ?? {})[0] ?? "");
        if (!skin) {
          const objects = await api.GetAssetObjects(active);
          const matches = objects.filter((v) =>
            Object.values(v.animations).includes(path),
          );
          const match =
            matches.find((v) => v.id === binding?.object.id) ?? matches[0];
          if (match) {
            const value = await api.PreviewAssetObject(active, match.id, "");
            if (seq !== fileRequest.current) return;
            setBinding(value);
            setGeometry(value.geometry);
            setTexture(value.image);
            setMaterialTextures(value.textures);
            setMaps(undefined);
          }
        }
      }
    } catch (e) {
      if (seq === fileRequest.current) setError(err(e));
    }
  }
  async function loadMaterial(path: string) {
    setManualPreview(true);
    setContentLoading(false);
    const seq = ++fileRequest.current;
    setFile(path);
    setError("");
    try {
      const p = await api.ReadAssetFile(active, path);
      const raw = JSON.parse(p.data);
      const set = raw["minecraft:texture_set"];
      if (!set) throw new Error("This file does not contain a texture set.");
      const load = async (value: unknown) => {
        if (
          Array.isArray(value) &&
          value.length >= 3 &&
          value.every((n) => typeof n === "number" && Number.isFinite(n))
        ) {
          const canvas = document.createElement("canvas");
          canvas.width = canvas.height = 1;
          const ctx = canvas.getContext("2d")!;
          const color = value
            .slice(0, 3)
            .map((n) => Math.round(Math.max(0, Math.min(255, n))));
          ctx.fillStyle = `rgb(${color.join(",")})`;
          ctx.fillRect(0, 0, 1, 1);
          return canvas.toDataURL("image/png");
        }
        if (
          typeof value === "string" &&
          /^#[\da-f]{6}([\da-f]{2})?$/i.test(value)
        ) {
          const canvas = document.createElement("canvas");
          canvas.width = canvas.height = 1;
          const ctx = canvas.getContext("2d")!;
          ctx.fillStyle = value;
          ctx.fillRect(0, 0, 1, 1);
          return canvas.toDataURL("image/png");
        }
        if (typeof value !== "string") return undefined;
        const directory = path.includes("/")
          ? path.slice(0, path.lastIndexOf("/") + 1)
          : "";
        const refs = value.startsWith("textures/")
          ? [value]
          : [directory + value, value];
        const candidates = refs.flatMap((ref) => [
          ref,
          ref + ".png",
          ref + ".jpg",
          ref + ".tga",
        ]);
        const f = detail?.files.find((f) => candidates.includes(f.path));
        if (!f) throw new Error("Material texture not found: " + value);
        return (await api.ReadAssetFile(active, f.path)).data;
      };
      const result: MaterialMaps = {
        color: await load(set.color),
        normal: await load(set.normal),
        mer: await load(
          set.metalness_emissive_roughness_subsurface ??
            set.metalness_emissive_roughness,
        ),
        height: await load(set.heightmap),
      };
      if (seq !== fileRequest.current) return;
      setMaps(result);
      setGeometry(undefined);
      setTab("preview");
    } catch (e) {
      if (seq === fileRequest.current) setError(err(e));
    }
  }
  async function compare(id: string) {
    setCompareID(id);
    setChanges([]);
    setCompareImage("");
    if (!id) return;
    const result = await api.CompareAssets(active, id);
    setChanges(result);
    setCompareImage(library.items.find((v) => v.id === id)?.image ?? "");
  }
  useEffect(() => {
    let dead = false;
    if (!stackIDs.length) {
      setStack(undefined);
      return;
    }
    void api
      .InspectAssetStack(stackIDs)
      .then((v) => {
        if (!dead) setStack(v);
      })
      .catch((e) => {
        if (!dead) setError(err(e));
      });
    return () => {
      dead = true;
    };
  }, [stackIDs, library.items]);
  const ids = selected.length ? selected : active ? [active] : [];
  const selectedBytes = library.items
    .filter((v) => ids.includes(v.id))
    .reduce((n, v) => n + v.bytes, 0);
  const deleteItems = library.items.filter((v) =>
    confirmDelete?.includes(v.id),
  );
  const deleteBytes = deleteItems.reduce((n, v) => n + v.bytes, 0);
  
  
  const files = useMemo(
    () =>
      detail?.files.filter(
        (f) =>
          (fileKind === "all" || f.kind === fileKind) &&
          f.path.toLowerCase().includes(fileQuery.toLowerCase()),
      ) ?? [],
    [detail, fileKind, fileQuery],
  );
  const Icon = skin ? Shirt : Layers;
  const fileEntries = useMemo(
    () => groupAudioFiles(files, groupSounds),
    [files, groupSounds],
  );
  const selectedInstallation = installations.find((v) => v.id === installation);
  
  function move(index: number, delta: number) {
    setStackIDs((old) => {
      const next = [...old];
      [next[index], next[index + delta]] = [next[index + delta], next[index]];
      return next;
    });
  }
  async function drop(files: FileList) {
    for (const file of Array.from(files)) {
      if (file.size > 128 << 20)
        throw new Error(
          "Drop files up to 128 MiB. Use Import files for larger packs.",
        );
      const data = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result).split(",")[1]);
        reader.onerror = () =>
          reject(new Error("Dropped file could not be read."));
        reader.readAsDataURL(file);
      });
      await acceptImports(await api.ImportAssetBytes(kind, file.name, data));
    }
  }
  return (
    <DocumentScope.Provider value={{ ...scope, busy: !!busy || scope.busy }}>
      <section
        className={`asset-studio ${dragging ? "is-dragging" : ""} ${propertiesOpen ? "properties-open" : ""}`}
        aria-label={skin ? "Skin library" : "Resource pack library"}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={(e) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node))
            setDragging(false);
        }}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          if (e.dataTransfer.files.length)
            void run("Importing dropped files", () =>
              drop(e.dataTransfer.files),
            );
        }}
      >
        <header className="asset-main-toolbar library-heading studio-view-heading">
          <div>
            <Button title="Properties" aria-label="Properties"
              className="asset-properties-toggle toolbar-icon-action"
              variant="outline"
              onClick={() => setPropertiesOpen((v) => !v)}
            ><HeaderActionIcon label="Properties"/>
              <span className="toolbar-action-label">Properties</span></Button>
            <Icon size={18} />
            <strong>{skin ? "Player skins" : "Resource packs"}</strong>
            <span>
              {itemCount} {itemCount === 1 ? "item" : "items"}
            </span>
          </div>
          <div>
            <Button title="Capture" aria-label="Capture" className="toolbar-icon-action" variant="ghost" onClick={onSource}><HeaderActionIcon label="Capture"/>
              <span className="toolbar-action-label">Capture</span></Button>
            {kind === "packs" && <Button title="Aim at an entity or custom block in Minecraft and press F8 to open its captured model here" aria-label="Pick model from Minecraft" className="toolbar-icon-action" variant="ghost" onClick={() => void (api as unknown as {PickMinecraftModel:()=>Promise<unknown>}).PickMinecraftModel().catch(e=>setError(err(e)))}><HeaderActionIcon label="Pick model"/><span className="toolbar-action-label">Pick model · F8</span></Button>}
            <Button title="Import files" aria-label="Import files" className="toolbar-icon-action"
              variant="outline"
              disabled={!!busy}
              onClick={() =>
                void run("Importing files", async () =>
                  acceptImports(await api.ImportAssets(kind, [])),
                )
              }
            >
              <Import />
              <span className="toolbar-action-label">Import files</span></Button>
            <Button title="Folder" aria-label="Folder" className="toolbar-icon-action"
              variant="ghost"
              disabled={!!busy}
              onClick={() =>
                void run("Importing folder", async () =>
                  acceptImports(await api.ImportAssetFolder(kind)),
                )
              }
            >
              <FolderOpen />
              <span className="toolbar-action-label">Folder</span></Button>
            <Button title="Minecraft" aria-label="Minecraft" className="toolbar-icon-action" variant="outline" onClick={() => setMinecraftOpen(true)}>
              <Gamepad2 />
              <span className="toolbar-action-label">Minecraft</span></Button>
            <Button
              disabled={!ids.length || !!busy}
              onClick={() => setExportOpen(true)}
            >
              <Download />
              Export{ids.length > 1 ? ` ${ids.length} items` : ""}
            </Button>
          </div>
        </header>
        {busy && (
          <div className="asset-operation" role="status">
            <LoaderCircle className="animate-spin" size={15} />
            {busy}…
          </div>
        )}
        {error && (
          <FeedbackFlyout title={skin ? "Player skins" : "Resource packs"}>
            <TriangleAlert size={16} />
            <span>{error}</span>
            <Button variant="ghost" onClick={() => setError("")}>
              Dismiss
            </Button>
          </FeedbackFlyout>
        )}
        <div className="asset-layout">
          <aside
            className={`studio-sidebar asset-browser view-${browserView}`}
            aria-label="Asset library"
          >
            <div className="asset-search">
              <Search size={15} />
              <input
                aria-label="Search assets"
                placeholder="Search name, source or tag"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            </div>
            <div className="asset-browser-tools">
              <Button
                variant={favourites ? "secondary" : "ghost"}
                aria-pressed={favourites}
                onClick={() => setFavourites((v) => !v)}
              >
                <Star />
                Favourites
              </Button>
              <select
                aria-label="Sort assets"
                value={sort}
                onChange={(e) => setSort(e.target.value)}
              >
                <option value="newest">Newest first</option>
                <option value="name">Name</option>
                <option value="size">Largest first</option>
              </select>
            </div>
            {skin && library.items.some((v) => v.captureType === "display") && (
              <Button
                variant="ghost"
                aria-pressed={showDisplays}
                onClick={() => setShowDisplays((v) => !v)}
              >
                {showDisplays
                  ? "Hide server display models"
                  : "Show server display models"}
              </Button>
            )}
            <div
              className="asset-view-toggle library-toolbar"
              role="group"
              aria-label="Library view"
            >
              <button
                aria-label="List view"
                aria-pressed={browserView === "list"}
                onClick={() => setBrowserView("list")}
              >
                <List size={15} />
              </button>
              <button
                aria-label="Grid view"
                aria-pressed={browserView === "grid"}
                onClick={() => setBrowserView("grid")}
              >
                <LayoutGrid size={15} />
              </button>
              <span>
                {list.length} {list.length === 1 ? "result" : "results"}
              </span>
            </div>
            <CollectionToolbar ids={list.map(v=>v.id)} selected={selected} multiple={multiple} onMultiple={setMultiple} onSelected={setSelected} onDelete={value=>setConfirmDelete(value)} busy={!!busy}/>
            <div
              className={`asset-list collection-items${!loading && !list.length ? " is-empty" : ""}`}
              data-layout={browserView}
              data-content={kind}
            >
              {loading ? (
                <p role="status">Loading library…</p>
              ) : (
                list.slice(0, visibleCount).map((v) => (
                  <div
                    key={v.id}
                    className={`asset-row collection-item ${active === v.id || (v.playerUUID && v.playerUUID === current?.playerUUID) ? "active" : ""}`}
                    data-selected={selected.includes(v.id)}
                  >
                    
                    <button
                      className="asset-row-main collection-select"
                      aria-label={v.name}
                      aria-pressed={selected.includes(v.id)}
                      title={`${v.name}\nClick to select · Ctrl-click to add · Shift-click for a range`}
                      onDoubleClick={()=>{if(!multiple)scope.onOpen?.(v.id,v.name,true);}}
                      onClick={e=>{
    const next=collectionSelection(list.map(value=>value.id),selected,selectionAnchor||active,v.id,e,multiple);
    setSelected(next.selected);setMultiple(next.multiple);setSelectionAnchor(next.anchor);
    if(!next.multiple){if(scope.onOpen)scope.onOpen(v.id,v.name,e.detail>=2);else setActive(v.id);}
   }}
                    >
                      <span
                        className={`asset-thumb collection-media ${skin ? "skin" : ""}`}
                      >
                        {thumbnails[v.id] || v.image ? (
                          <AssetThumbnail
                            src={v.image}
                            skin={skin && v.width <= 128}
                            rendered={thumbnails[v.id]}
                          />
                        ) : (
                          <Icon size={25} />
                        )}
                      </span>
                      <span className="collection-copy">
                        <strong>{v.name}</strong>
                        <small>
                          {skin
                            ? `${v.width} × ${v.height}${v.hasGeometry ? " · Custom model" : ""}`
                            : `${v.version || "No version"} · ${bytes(v.bytes)}`}
                        </small>
                        <small>
                          {v.captureType === "display"
                            ? "Server display model"
                            : v.source}
                        </small>
                      </span>
                      {v.favourite && (
                        <Star
                          className="collection-favourite"
                          size={12}
                          fill="currentColor"
                          aria-label="Favourite"
                        />
                      )}
                    </button>
                    
                  </div>
                ))
              )}
              {list.length > visibleCount && (
                <Button
                  className="collection-more"
                  variant="ghost"
                  onClick={() => setVisibleCount((n) => n + 100)}
                >
                  Show 100 more
                </Button>
              )}
              {!loading && !list.length && (
                <div className="asset-empty">
                  <Icon size={34} />
                  <h3>
                    {query || favourites
                      ? "No matching items"
                      : skin
                        ? "Add your first skin"
                        : "Add your first resource pack"}
                  </h3>
                  <p>
                    {query || favourites
                      ? "Change the search or turn off the favourites filter."
                      : "Import files, drag them here, or browse your installed Minecraft content."}
                  </p>
                  <Button variant="outline" onClick={onSource}>
                    Capture from a server
                  </Button>
                </div>
              )}
            </div>
            <footer>
              <span>
                {selected.length
                  ? `${selected.length} selected · ${bytes(selectedBytes)}`
                  : `${bytes(library.bytes)} in library`}
              </span>
              <div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Refresh library"
                  disabled={!!busy}
                  onClick={() => void run("Refreshing library", refresh)}
                >
                  <RefreshCw />
                </Button>
                
              </div>
            </footer>
          </aside>
          <main className="asset-studio-stage">
            <div data-editor-tab-host />
            <nav className="asset-tabs" aria-label="Asset tools">
              {(
                [
                  ["preview", "Preview"],
                  ["files", "Files"],
                  ["compare", skin ? "Variants" : "Compare"],
                ] as [Tab, string][]
              ).map(([value, label]) => (
                <button
                  key={value}
                  aria-pressed={tab === value}
                  onClick={() => setTab(value)}
                >
                  {label}
                </button>
              ))}
            </nav>
            <div hidden={tab !== "preview"}>
              {!skin && detail?.item.id === active && (
                <PackContentPicker
                  prepared={initialView?.picked === true}
                  liveStates={objectSelection.object === initialView?.object ? initialView?.blockStates as Record<string,unknown> | undefined : undefined}
                  initialSelection={objectSelection}
                  onSelection={setObjectSelection}
                  key={active}
                  assetID={active}
                  revision={library.blockRevisions?.[active]??""}
                  encrypted={!!detail.item.encrypted}
                  manual={manualPreview}
                  onChoose={() => {
                    setCameraView(undefined);
                    ++fileRequest.current;
                    setManualPreview(false);
                    setPreview(undefined);
                  }}
                  onError={setError}
                  onLoading={setContentLoading}
                  onBinding={(value) => {
                    setBinding(value);
                    setGeometry(value?.geometry);
                    setMaterialTextures(value?.textures);
                    setTexture(value?.image ?? "");
                    setMaps(undefined);
                    if (value) setFile(value.object.definition);
                  }}
                />
              )}
            </div>
            {!current ? (
              <div className="asset-empty">
                <Icon size={48} />
                <h3>{skin ? "Your skin collection" : "Your resource packs"}</h3>
                <p>Choose an item to preview it and see its files.</p>
              </div>
            ) : !detail ? (
              <div className="asset-empty" role="status">
                <LoaderCircle className="animate-spin" />
                Reading asset…
              </div>
            ) : (
              <>
                {(tab === "preview" || tab === "edit") && (
                  <>
                    <div className="asset-preview-tools">
                      <strong>
                        {file && preview?.kind === "texture" && !skin
                          ? file
                          : current.name}
                      </strong>
                      {hasVisiblePreview && (
                        <div>
                          {!skin && !!maps && (
                            <>
                              <Button
                                variant="ghost"
                                aria-pressed={shape === "cube"}
                                onClick={() => setShape("cube")}
                              >
                                Cube
                              </Button>
                              <Button
                                variant="ghost"
                                aria-pressed={shape === "sphere"}
                                onClick={() => setShape("sphere")}
                              >
                                Sphere
                              </Button>
                            </>
                          )}
                          <Button
                            variant="ghost"
                            onClick={() => {
                              viewport.current?.reset();
                              textureViewport.current?.reset();
                            }}
                          >
                            Fit view
                          </Button>
                          <Button
                            variant="outline"
                            onClick={() =>
                              void run("Saving preview", async () => {
                                const data =
                                  !skin && !geometry && !maps
                                    ? texture
                                    : viewport.current?.capture();
                                if (!data)
                                  throw new Error("Preview is still loading.");
                                notifySaved(
                                  await api.SaveImage({
                                    name: current.name,
                                    format: "png",
                                    data,
                                    quality: 1,
                                    background: "transparent",
                                  }),
                                );
                              })
                            }
                          >
                            <ImageDown />
                            Save image
                          </Button>
                        </div>
                      )}
                    </div>
                    {!skin && binding && (
                      <div className="pack-binding-info">
                        <strong>{packObjectLabel(binding.object).name}</strong>
                        <span>
                          {[
                            packObjectKind(binding.object.kind),
                            packObjectLabel(binding.object).context,
                          ]
                            .filter(Boolean)
                            .join(" - ")}
                        </span>
                        <details>
                          <summary>Identifiers and source</summary>
                          <code>{binding.object.name}</code>
                          {binding.object.geometryId && (
                            <code>{binding.object.geometryId}</code>
                          )}
                          <code>{binding.object.definition}</code>
                        </details>
                        {!!binding.warnings.length && <FeedbackFlyout title="Resource pack preview" tone="warning">
                          {binding.warnings.map((message, i) => <p key={i}>{message}</p>)}
                        </FeedbackFlyout>}
                      </div>
                    )}
                    {tab === "edit" ? (
                      <TextureEditor
                        key={
                          preview?.kind === "texture"
                            ? preview.path
                            : detail.item.texturePath || "skin.png"
                        }
                        initialDraft={
                          textureDrafts[
                            preview?.kind === "texture"
                              ? preview.path
                              : detail.item.texturePath || "skin.png"
                          ]
                        }
                        onDraftChange={(png) => {
                          const key =
                            preview?.kind === "texture"
                              ? preview.path
                              : detail.item.texturePath || "skin.png";
                          setTextureDrafts((old) =>
                            old[key] === png ? old : { ...old, [key]: png },
                          );
                        }}
                        image={
                          preview?.kind === "texture"
                            ? preview.data
                            : current.image
                        }
                        busy={!!busy}
                        onSave={(data) =>
                          void run("Saving edited copy", async () => {
                            const path =
                              preview?.kind === "texture"
                                ? preview.path
                                : detail.item.texturePath;
                            if (!path)
                              throw new Error("Choose a PNG texture first.");
                            await acceptImports(
                              await api.SaveAssetTexture(
                                active,
                                path,
                                current.name + " edited",
                                data,
                              ),
                            );
                            setTab("preview");
                          })
                        }
                      />
                    ) : (
                      <div
                        className={`asset-preview-area${!hasVisiblePreview ? " is-empty" : ""}`}
                      >
                        {skin && variants.length > 1 && (
                          <label className="asset-preview-note">
                            Appearance{" "}
                            <select
                              aria-label="Captured appearance"
                              value={active}
                              onChange={(e) =>
                                scope.onOpen
                                  ? scope.onOpen(
                                      e.target.value,
                                      library.items.find(
                                        (v) => v.id === e.target.value,
                                      )?.name || "Player skin",
                                      true,
                                    )
                                  : setActive(e.target.value)
                              }
                            >
                              {variants.map((v) => (
                                <option key={v.id} value={v.id}>
                                  {v.persona
                                    ? "Original character"
                                    : "Captured skin"}{" "}
                                  - {new Date(v.created).toLocaleString()}
                                </option>
                              ))}
                            </select>
                          </label>
                        )}
                        {skin && cosmeticsLoading && (
                          <div className="asset-preview-note" role="status">
                            Loading equipped cosmetics...
                          </div>
                        )}
                        {!skin && detail.item.encrypted ? (
                          <div className="asset-empty">
                            <Layers size={42} />
                            <h3>Encrypted pack content</h3>
                            <p>
                              The files were saved without their readable
                              session copy. Capture this server again with this
                              version of CraftBox to load its actual models and
                              textures.
                            </p>
                            <Button variant="outline" onClick={onSource}>
                              Capture readable content
                            </Button>
                            <Button
                              disabled={!!busy}
                              onClick={() =>
                                void run(
                                  "Restoring saved pack content",
                                  async () => {
                                    const result =
                                      await api.RecoverCapturedAssets();
                                    await refresh();
                                    notify({
                                      tone: result.recovered
                                        ? "success"
                                        : "warning",
                                      title: result.recovered
                                        ? `${result.recovered} packs restored`
                                        : "No matching saved handshake found",
                                      description: result.remaining
                                        ? `${result.remaining} packs still need a new capture. ${result.warnings.join(" ")}`
                                        : "Models and textures are ready to preview.",
                                    });
                                  },
                                )
                              }
                            >
                              Restore from saved captures
                            </Button>
                          </div>
                        ) : !skin && contentLoading ? (
                          <div className="asset-empty" role="status">
                            Resolving models and textures…
                          </div>
                        ) : (!skin && !geometry && !maps) ||
                          (skin &&
                            !geometry &&
                            !current.hasGeometry &&
                            (current.width > 128 ||
                              (current.width !== current.height &&
                                current.width !== current.height * 2)) &&
                            !pngAsSkin) ? (
                          <div
                            className={`asset-flat-preview${!texture ? " is-empty" : ""}`}
                          >
                            {texture ? (
                              <TextureViewport
                                ref={textureViewport}
                                src={texture}
                                name={binding?.object.name ?? current.name}
                              />
                            ) : (
                              <div className="asset-empty">
                                <h3>
                                  {!skin &&
                                  detail?.files.some(
                                    (f) => f.kind === "sound",
                                  ) &&
                                  !detail.files.some(
                                    (f) =>
                                      f.kind === "model" ||
                                      (f.kind === "texture" &&
                                        f.path !== "pack_icon.png"),
                                  )
                                    ? "Audio resource pack"
                                    : "No renderable object selected"}
                                </h3>
                                <p>
                                  {!skin &&
                                  detail?.files.some(
                                    (f) => f.kind === "sound",
                                  ) &&
                                  !detail.files.some(
                                    (f) =>
                                      f.kind === "model" ||
                                      (f.kind === "texture" &&
                                        f.path !== "pack_icon.png"),
                                  )
                                    ? "This pack contains audio instead of models or textures. Open Files to inspect its contents."
                                    : "Select a model, item or texture above. Missing references are listed with the selected object."}
                                </p>
                                {!skin && (
                                  <Button
                                    variant="outline"
                                    onClick={() => setTab("files")}
                                  >
                                    Open files
                                  </Button>
                                )}
                              </div>
                            )}
                          </div>
                        ) : (
                          <Suspense
                            fallback={
                              <p role="status">Preparing 3D preview…</p>
                            }
                          >
                            {skin && (!modelReady || cosmeticsLoading) ? (
                              <p role="status">
                                Preparing skin and equipped cosmetics…
                              </p>
                            ) : (
                              <AssetViewport
                                cameraState={cameraView}
                                onCameraChange={setCameraView}
                                ref={viewport}
                                onThumbnail={(data) =>
                                  setThumbnails((old) => {
                                    if (old[active] === data) return old;
                                    const next = Object.fromEntries([
                                      ...Object.entries(old).slice(-127),
                                      [active, data],
                                    ]);
                                    try {
                                      localStorage.setItem(
                                        "craftbox:asset-thumbnails-v5",
                                        JSON.stringify(next),
                                      );
                                    } catch {}
                                    return next;
                                  })
                                }
                                skin={skin}
                                effects={effects}
                                texture={texture || undefined}
                                cape={cape || undefined}
                                face={face}
                                animatedLayers={animatedLayers}
                                modelOnly={current.captureType === "display"}
                                geometry={useGeometry ? geometry : undefined}
                                attachments={skin ? attachments : undefined}
                                animation={animation}
                                animationName={animationName}
                                maps={maps}
                                materialTextures={materialTextures}
                                materialMode={binding?.object.material}
                                object={binding?.object}
                                initialPose={binding?.initialPose}
                                motion={binding?.motion}
                                textureAnimation={binding?.textureAnimation}
                                shape={shape}
                                slim={
                                  arm === "auto"
                                    ? current.armModel === "slim"
                                    : arm === "slim"
                                }
                                legacy={current.width === current.height * 2}
                                pose={pose}
                                walking={walking}
                                outer={outer}
                                showCape={showCape}
                                rotate={rotate}
                                wireframe={wireframe}
                                wireframeOptions={wireframeOptions}
                                onWireframeStats={setWireframeStats}
                                light={light}
                                onError={setError}
                              />
                            )}
                          </Suspense>
                        )}
                        <div className="asset-viewport-hint">
                          {(!skin && !geometry && !maps) ||
                          (skin &&
                            !geometry &&
                            !current.hasGeometry &&
                            current.width > 128 &&
                            !pngAsSkin)
                            ? "Drag to pan · Scroll to zoom · 0 to fit"
                            : "Drag to orbit · Right-drag to pan · Scroll to zoom"}
                        </div>
                        {skin &&
                          !geometry &&
                          !current.hasGeometry &&
                          current.width > 128 &&
                          !pngAsSkin && (
                            <div className="skin-png-message">
                              <p>
                                This PNG contains a texture, but no model. A
                                texture atlas needs matching geometry. Import
                                the original skin pack or a folder containing
                                skin.png and geometry.json.
                              </p>
                              <Button
                                variant="outline"
                                onClick={() => setPngAsSkin(true)}
                              >
                                Use standard skin layout
                              </Button>
                            </div>
                          )}
                        {maps && (
                          <div className="asset-preview-note">
                            Material preview · Minecraft lighting and subsurface
                            scattering may look different.
                          </div>
                        )}
                      </div>
                    )}
                    {skin && tab === "preview" && (
                      <SkinControls
                        key={active}
                        pose={pose}
                        setPose={setPose}
                        walking={walking}
                        setWalking={setWalking}
                        packs={library.items.filter((v) => v.kind === "packs")}
                        attachments={attachments}
                        setAttachments={setAttachments}
                        geometry={geometry}
                        preview={{
                          texture: texture || undefined,
                          geometry: useGeometry ? geometry : undefined,
                          face,
                          animatedLayers,
                          cape: cape || undefined,
                          slim:
                            arm === "auto"
                              ? current?.armModel === "slim"
                              : arm === "slim",
                          legacy:
                            !!current && current.width === current.height * 2,
                          outer,
                          showCape,
                          light,
                        }}
                        effects={effects}
                        setEffects={setEffects}
                      />
                    )}
                  </>
                )}
                {tab === "files" && (
                  <div className="asset-files-layout">
                    <div className="asset-file-list">
                      <div className="asset-search">
                        <Search size={14} />
                        <input
                          aria-label="Search files"
                          placeholder="Search files"
                          value={fileQuery}
                          onChange={(e) => {
                            setFileQuery(e.target.value);
                            setFilePage(0);
                          }}
                        />
                      </div>
                      <select
                        aria-label="File type"
                        value={fileKind}
                        onChange={(e) => {
                          setFileKind(e.target.value);
                          setFilePage(0);
                        }}
                      >
                        {[
                          "all",
                          "texture",
                          "model",
                          "sound",
                          "animation",
                          "json",
                          "text",
                        ].map((t) => (
                          <option value={t} key={t}>
                            {t === "all"
                              ? "All files"
                              : t[0].toUpperCase() + t.slice(1)}
                          </option>
                        ))}
                      </select>
                      {detail.files.some((f) => f.kind === "sound") && (
                        <label className="asset-audio-group-controls">
                          <input
                            type="checkbox"
                            checked={groupSounds}
                            onChange={(e) => {
                              setGroupSounds(e.target.checked);
                              setFilePage(0);
                              try {
                                localStorage.setItem(
                                  "craftbox:group-audio",
                                  String(e.target.checked),
                                );
                              } catch {}
                            }}
                          />
                          Group audio by folder
                        </label>
                      )}
                      {fileEntries
                        .slice(filePage * 150, (filePage + 1) * 150)
                        .map((entry) =>
                          entry.grouped ? (
                            <details
                              className="asset-audio-group"
                              key={entry.id}
                            >
                              <summary>
                                {entry.name}
                                <small>
                                  {entry.files.length} sound files ·{" "}
                                  {bytes(
                                    entry.files.reduce(
                                      (n, f) => n + f.bytes,
                                      0,
                                    ),
                                  )}
                                </small>
                              </summary>
                              <button
                                onClick={() => {
                                  ++fileRequest.current;
                                  setPreview(undefined);
                                  setFile("");
                                  setAudioSelection(entry.files);
                                }}
                              >
                                Open as audio sequence
                              </button>
                              <div>
                                {entry.files.slice(0, 150).map((f) => (
                                  <button
                                    key={f.path}
                                    aria-pressed={file === f.path}
                                    onClick={() => void openFile(f.path)}
                                  >
                                    <span>{f.path.split("/").at(-1)}</span>
                                    <small>{bytes(f.bytes)}</small>
                                  </button>
                                ))}
                                {entry.files.length > 150 && (
                                  <small>
                                    Search files to narrow this group.
                                  </small>
                                )}
                              </div>
                            </details>
                          ) : (
                            <button
                              key={entry.id}
                              aria-pressed={file === entry.id}
                              onClick={() => void openFile(entry.id)}
                            >
                              <span>{entry.name}</span>
                              <small>{bytes(entry.files[0].bytes)}</small>
                            </button>
                          ),
                        )}
                      <div className="asset-tools">
                        <Button
                          variant="ghost"
                          disabled={!filePage}
                          onClick={() => setFilePage((n) => n - 1)}
                        >
                          Previous
                        </Button>
                        <span>{files.length} files</span>
                        <Button
                          variant="ghost"
                          disabled={(filePage + 1) * 150 >= fileEntries.length}
                          onClick={() => setFilePage((n) => n + 1)}
                        >
                          Next
                        </Button>
                      </div>
                    </div>
                    <div className="asset-file-preview">
                      {audioSelection.length ? (
                        <Suspense
                          fallback={
                            <p role="status">Preparing audio studio…</p>
                          }
                        >
                          <AudioStudio
                            key={
                              active +
                              JSON.stringify(audioSelection.map((f) => f.path))
                            }
                            initialState={
                              audioStates[
                                JSON.stringify(
                                  audioSelection.map((f) => f.path),
                                )
                              ]
                            }
                            onLoading={setAudioLoading}
                            onStateChange={(value) => {
                              const key = JSON.stringify(
                                audioSelection.map((f) => f.path),
                              );
                              setAudioStates((old) =>
                                JSON.stringify(old[key]) ===
                                JSON.stringify(value)
                                  ? old
                                  : { ...old, [key]: value },
                              );
                            }}
                            assetID={active}
                            files={audioSelection}
                            available={detail.files}
                            initial={preview}
                          />
                        </Suspense>
                      ) : preview?.kind === "texture" ? (
                        <>
                          <div className="asset-tools">
                            <label>
                              Zoom
                              <input
                                type="range"
                                min={1}
                                max={8}
                                step={0.5}
                                value={textureZoom}
                                onChange={(e) =>
                                  setTextureZoom(+e.target.value)
                                }
                              />
                            </label>
                            <Button
                              variant="outline"
                              disabled={!file.endsWith(".png")}
                              onClick={() => setTab("edit")}
                            >
                              Edit texture
                            </Button>
                            <Button
                              variant="outline"
                              onClick={() => setTab("preview")}
                            >
                              Preview on model
                            </Button>
                          </div>
                          <div className="asset-image-scroll">
                            <img
                              src={preview.data}
                              alt={preview.path}
                              style={{
                                transform: `scale(${textureZoom})`,
                                transformOrigin: "top left",
                              }}
                            />
                          </div>
                        </>
                      ) : preview ? (
                        <>
                          <div className="asset-tools">
                            <strong>{file}</strong>
                            {file.endsWith(".texture_set.json") && (
                              <Button onClick={() => void loadMaterial(file)}>
                                Preview material
                              </Button>
                            )}
                            {(preview.kind === "model" ||
                              preview.kind === "animation") && (
                              <Button onClick={() => setTab("preview")}>
                                {preview.kind === "animation"
                                  ? "Preview animation"
                                  : "Preview model"}
                              </Button>
                            )}
                          </div>
                          <pre>{preview.data}</pre>
                        </>
                      ) : (
                        <div className="asset-empty">
                          <FolderOpen size={36} />
                          <p>
                            Choose a texture, model, animation, sound or text
                            file.
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                )}
                {tab === "checks" && (
                  <div className="asset-report">
                    <h3>
                      {detail.issues.length
                        ? `${detail.issues.length} checks need attention`
                        : "No issues found in the checked files"}
                    </h3>
                    <p>
                      Checks cover this asset. A resource supplied by Minecraft
                      or another active pack can resolve a missing reference.
                    </p>
                    <Button
                      variant="outline"
                      disabled={!!busy}
                      onClick={() =>
                        void run("Checking asset", async () =>
                          setDetail(await api.GetAssetDetail(active)),
                        )
                      }
                    >
                      <RefreshCw />
                      Check again
                    </Button>
                    {detail.issues.map((issue, i) => (
                      <button
                        className={`asset-issue ${issue.severity}`}
                        key={i}
                        onClick={() => {
                          setTab("files");
                          setFileQuery(issue.path);
                          void openFile(issue.path);
                        }}
                      >
                        {issue.severity === "error" ? (
                          <TriangleAlert size={17} />
                        ) : issue.severity === "warning" ? (
                          <TriangleAlert size={17} />
                        ) : (
                          <CheckCircle2 size={17} />
                        )}
                        <span>
                          <strong>{issue.message}</strong>
                          <small>{issue.path}</small>
                          {issue.reference && <code>{issue.reference}</code>}
                        </span>
                      </button>
                    ))}
                  </div>
                )}
                {tab === "compare" && (
                  <div className="asset-report">
                    <h3>
                      {skin
                        ? "Compare captured variants"
                        : "Compare pack versions"}
                    </h3>
                    <label>
                      Compare with
                      <select
                        value={compareID}
                        onChange={(e) =>
                          void run("Comparing files", () =>
                            compare(e.target.value),
                          )
                        }
                      >
                        <option value="">Choose an item</option>
                        {library.items
                          .filter((v) => v.kind === kind && v.id !== active)
                          .map((v) => (
                            <option key={v.id} value={v.id}>
                              {v.name} {v.version} ·{" "}
                              {new Date(v.created).toLocaleString()}
                            </option>
                          ))}
                      </select>
                    </label>
                    {compareID && (
                      <>
                        <div className="asset-comparison-images">
                          {[current.image, compareImage].map(
                            (image, i) =>
                              image && (
                                <figure key={i}>
                                  <img
                                    src={image}
                                    alt={i ? "Compared item" : "Selected item"}
                                  />
                                  <figcaption>
                                    {i
                                      ? library.items.find(
                                          (v) => v.id === compareID,
                                        )?.name
                                      : current.name}
                                  </figcaption>
                                </figure>
                              ),
                          )}
                        </div>
                        <p>
                          {changes.filter((v) => v.status === "added").length}{" "}
                          added ·{" "}
                          {changes.filter((v) => v.status === "removed").length}{" "}
                          removed ·{" "}
                          {
                            changes.filter((v) => v.status === "modified")
                              .length
                          }{" "}
                          changed
                        </p>
                        {!changes.length && <p>The files are identical.</p>}
                        {changes.slice(0, 1000).map((v) => (
                          <div className="asset-change" key={v.path}>
                            <span className={v.status}>{v.status}</span>
                            <code>{v.path}</code>
                          </div>
                        ))}
                        {changes.length > 1000 && (
                          <p>Showing the first 1,000 changed files.</p>
                        )}
                      </>
                    )}
                  </div>
                )}
                {tab === "stack" && (
                  <div className="asset-report">
                    <h3>Pack order</h3>
                    <p>
                      The first pack has priority. This order is used for
                      CraftBox previews; it does not change Minecraft's global
                      settings.
                    </p>
                    <Button
                      variant="outline"
                      onClick={() =>
                        setStackIDs((old) => [
                          ...old,
                          ...ids.filter(
                            (id) =>
                              !old.includes(id) &&
                              library.items.some(
                                (v) => v.id === id && v.kind === "packs",
                              ),
                          ),
                        ])
                      }
                    >
                      Add selected packs
                    </Button>
                    {stackIDs.map((id, i) => (
                      <div className="asset-stack-row" key={id}>
                        <span>{i + 1}</span>
                        <strong>
                          {library.items.find((v) => v.id === id)?.name ??
                            "Unavailable pack"}
                        </strong>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="Move pack up"
                          disabled={!i}
                          onClick={() => move(i, -1)}
                        >
                          <ArrowUp />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="Move pack down"
                          disabled={i === stackIDs.length - 1}
                          onClick={() => move(i, 1)}
                        >
                          <ArrowDown />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="Remove pack from stack"
                          onClick={() =>
                            setStackIDs((old) => old.filter((v) => v !== id))
                          }
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    ))}
                    {stack && (
                      <>
                        <h3>{stack.conflicts.length} overlapping files</h3>
                        <div className="asset-search">
                          <Search size={14} />
                          <input
                            placeholder="Find a resource path"
                            aria-label="Find a resource in the stack"
                            value={stackFilter}
                            onChange={(e) => setStackFilter(e.target.value)}
                          />
                        </div>
                        {!!stack.issues.length && <FeedbackFlyout title="Resource pack stack" tone="warning">
                          {stack.issues.map((v, i) => <p key={i}>{v.message}: {v.reference}</p>)}
                        </FeedbackFlyout>}
                        {Object.entries(stack.resolved)
                          .filter(([path]) =>
                            path
                              .toLowerCase()
                              .includes(stackFilter.toLowerCase()),
                          )
                          .slice(0, 200)
                          .map(([path, winner]) => (
                            <div className="asset-resolved" key={path}>
                              <button
                                onClick={() => {
                                  if (scope.onOpen)
                                    scope.onOpen(
                                      winner,
                                      library.items.find((v) => v.id === winner)
                                        ?.name || winner,
                                      true,
                                    );
                                  else setActive(winner);
                                  setTab("files");
                                  setFileQuery(path);
                                }}
                              >
                                {path}
                              </button>
                              <span>
                                {
                                  library.items.find((v) => v.id === winner)
                                    ?.name
                                }
                              </span>
                              {stack.conflicts.some((c) => c.path === path) && (
                                <small>Overrides another pack</small>
                              )}
                            </div>
                          ))}
                        <p>
                          Showing up to 200 resources. Use the search to find a
                          texture or model. Apply this order from Resource packs
                          in the Render editor.
                        </p>
                        <Button
                          variant="outline"
                          onClick={() => {
                            localStorage.setItem(
                              "craftbox.asset-stack",
                              JSON.stringify(stackIDs),
                            );
                            notify({
                              tone: "success",
                              title: "Pack order saved for the Render editor",
                            });
                          }}
                        >
                          Use this order in Render editor
                        </Button>
                      </>
                    )}
                  </div>
                )}
              </>
            )}
          </main>
          <aside
            className="asset-inspector library-properties"
            aria-label="Selection properties"
          >
            <h3 className="asset-properties-heading">Properties</h3>
            {current ? (
              <>
                <h3>{current.name}</h3>
                <Button
                  variant={current.favourite ? "secondary" : "outline"}
                  disabled={!!busy}
                  onClick={() =>
                    void run("Saving favourite", async () => {
                      await api.UpdateAsset(
                        active,
                        current.name,
                        !current.favourite,
                        current.tags,
                      );
                      await refresh();
                    })
                  }
                >
                  <Star fill={current.favourite ? "currentColor" : "none"} />
                  {current.favourite ? "In favourites" : "Add to favourites"}
                </Button>
                <dl>
                  <dt>Source</dt>
                  <dd>{current.source}</dd>
                  <dt>Added</dt>
                  <dd>{new Date(current.created).toLocaleString()}</dd>
                  <dt>Size</dt>
                  <dd>
                    {bytes(current.bytes)} · {current.files} files
                  </dd>
                  {skin ? (
                    <>
                      <dt>Texture</dt>
                      <dd>
                        {current.width} × {current.height}
                      </dd>
                      <dt>Model</dt>
                      <dd>
                        {current.persona
                          ? "Character Creator"
                          : current.hasGeometry
                            ? "Custom geometry"
                            : current.armModel === "slim"
                              ? "Slim arms"
                              : "Wide arms"}
                      </dd>
                      <dt>Extra data</dt>
                      <dd>
                        {[
                          current.hasCape && "Cape",
                          current.hasAnimations && "Animations",
                        ]
                          .filter(Boolean)
                          .join(", ") || "None"}
                      </dd>
                    </>
                  ) : (
                    <>
                      <dt>Version</dt>
                      <dd>{current.version || "Not specified"}</dd>
                      <dt>Minimum Minecraft</dt>
                      <dd>{current.minimumVersion || "Not specified"}</dd>
                      <dt>Pack ID</dt>
                      <dd className="asset-mono">
                        {current.uuid || "Not specified"}
                      </dd>
                    </>
                  )}
                </dl>
                <details className="asset-metadata-editor">
                  <summary>Edit name &amp; tags</summary>
                  <label>
                    Name
                    <input
                      value={name}
                      maxLength={160}
                      onChange={(e) => setName(e.target.value)}
                    />
                  </label>
                  <label>
                    Tags
                    <input
                      value={tags}
                      placeholder="Building, favourites, server…"
                      onChange={(e) => setTags(e.target.value)}
                    />
                  </label>
                  <Button
                    variant="outline"
                    disabled={!!busy || !name.trim()}
                    onClick={() =>
                      void run("Saving details", async () => {
                        if (skin && arm !== "auto")
                          await api.SetAssetArmModel(active, arm);
                        await api.UpdateAsset(
                          active,
                          name,
                          current.favourite,
                          tags
                            .split(",")
                            .map((t) => t.trim())
                            .filter(Boolean),
                        );
                        await refresh();
                      })
                    }
                  >
                    Save details
                  </Button>
                </details>
                {tab === "preview" && hasModelPreview && (
                  <>
                    <hr />
                    <h4>Preview</h4>
                    {!!animation && (
                      <>
                        <label>
                          Animation
                          <select
                            value={animationName}
                            onChange={(e) => setAnimationName(e.target.value)}
                          >
                            <option value="">Paused</option>
                            {Object.keys(
                              (
                                animation as {
                                  animations?: Record<string, unknown>;
                                }
                              ).animations ?? {},
                            ).map((name) => (
                              <option key={name} value={name}>
                                {name}
                              </option>
                            ))}
                          </select>
                        </label>
                        <small>
                          Numeric keyframes are previewed. Molang expressions
                          and controller state transitions require Minecraft.
                        </small>
                      </>
                    )}
                    {skin && (
                      <>
                        <label>
                          Arm model
                          <select
                            aria-label="Arm model"
                            value={arm}
                            onChange={(e) => setArm(e.target.value)}
                          >
                            <option value="auto">
                              Detected model (
                              {current.armModel === "slim" ? "slim" : "wide"})
                            </option>
                            <option value="wide">Wide arms</option>
                            <option value="slim">Slim arms</option>
                          </select>
                        </label>
                        {arm !== "auto" && (
                          <Button
                            variant="outline"
                            disabled={!!busy}
                            onClick={() =>
                              void run("Saving arm model", async () => {
                                await api.SetAssetArmModel(active, arm);
                                await refresh();
                              })
                            }
                          >
                            Save arm model
                          </Button>
                        )}
                        {current.hasGeometry && (
                          <label className="asset-check">
                            <input
                              type="checkbox"
                              checked={useGeometry}
                              onChange={(e) => setUseGeometry(e.target.checked)}
                            />
                            Use captured geometry
                          </label>
                        )}
                        <label className="asset-check">
                          <input
                            type="checkbox"
                            checked={outer}
                            onChange={(e) => setOuter(e.target.checked)}
                          />
                          Outer clothing layer
                        </label>
                        {current.hasCape && (
                          <label className="asset-check">
                            <input
                              type="checkbox"
                              checked={showCape}
                              onChange={(e) => setShowCape(e.target.checked)}
                            />
                            Show cape
                          </label>
                        )}
                      </>
                    )}
                    <label className="asset-check">
                      <input
                        type="checkbox"
                        checked={rotate}
                        onChange={(e) => setRotate(e.target.checked)}
                      />
                      Turntable
                    </label>
                    <WireframeControls
                      enabled={wireframe}
                      settings={wireframeOptions}
                      stats={wireframeStats}
                      onToggle={setWireframe}
                      onChange={setWireframeOptions}
                    />
                    <label>
                      Light intensity
                      <input
                        type="range"
                        min={0}
                        max={8}
                        step={0.1}
                        value={light}
                        onChange={(e) => setLight(+e.target.value)}
                      />
                    </label>
                    {skin && (
                      <Button variant="outline" onClick={() => setTab("edit")}>
                        Edit skin texture
                      </Button>
                    )}
                    <hr />
                    <h4>3D model export</h4>
                    <select
                      aria-label="Model export format"
                      value={modelFormat}
                      onChange={(e) => setModelFormat(e.target.value)}
                    >
                      {["bbmodel", "glb", "gltf", "obj", "fbx", "stl"].map((f) => (
                        <option key={f} value={f}>
                          {f === "bbmodel"
                            ? "Blockbench project · Editable"
                            : f.toUpperCase()}
                          {f === "stl" ? " · Geometry only" : ""}
                        </option>
                      ))}
                    </select>
                    {modelFormat === "bbmodel" && (
                      <label>
                        Blockbench model type
                        <select
                          aria-label="Blockbench model type"
                          value={blockbenchFormat}
                          onChange={(e) => setBlockbenchFormat(e.target.value as typeof blockbenchFormat)}
                        >
                          <option value="auto">
                            Auto · {binding?.object.kind === "block" ? "Bedrock Block" : "Bedrock Entity"}
                          </option>
                          <option value="entity">Bedrock Entity</option>
                          <option value="block">Bedrock Block</option>
                          <option value="free">
                            Generic Model · Multiple materials / meshes
                          </option>
                        </select>
                      </label>
                    )}
                    <Button
                      variant="outline"
                      disabled={!!busy || (!skin && !geometry && !maps)}
                      onClick={() =>
                        void run("Exporting 3D model", async () => {
                          if (modelFormat === "bbmodel") {
                            if (!geometry || !detail)
                              throw Error("Select authored model geometry before exporting a Blockbench project.");
                            const selectedBinding =
                              binding?.geometry === geometry ? binding : undefined;
                            const original = await api.GetAssetModelSources(
                              active,
                              selectedBinding?.object.id ?? "",
                            );
                            const model = await prepareBlockbenchSource({
                              name: current.name,
                              geometry,
                              geometryId: selectedBinding?.geometryId,
                              textures: materialTextures ?? (texture ? { "*": texture } : {}),
                              textureNames: selectedBinding?.object.textures,
                              object: selectedBinding?.object,
                              sources: original.documents,
                              motion: {
                                aliases: original.aliases,
                                scripts: original.scripts,
                                animations: original.animations,
                                controllers: original.controllers,
                                particles: selectedBinding?.motion?.particles ?? {},
                                hideCape: selectedBinding?.motion?.hideCape ?? false,
                              },
                              animations: {
                                ...original.animations,
                                ...(animation && animationName ? {
                                  [animationName]: (animation as { animations?: Record<string, unknown> }).animations?.[animationName] ?? animation,
                                } : {}),
                              },
                            }, detail.files, path => api.ReadAssetFile(active, path));
                            const result = createBlockbenchProject({
                              name: current.name,
                              models: [model],
                              format: blockbenchFormat,
                            });
                            const path = await api.SaveAssetModel(
                              current.name,
                              blockbenchProjectData(result.project),
                              "bbmodel",
                            );
                            if (path) {
                              notifySaved(path);
                              if (result.warnings.length)
                                notify({
                                  tone: "warning",
                                  title: "Blockbench project saved",
                                  description: result.warnings.join(" "),
                                });
                            }
                            return;
                          }
                          const data = await viewport.current?.model();
                          if (!data)
                            throw new Error("Preview is still loading.");
                          notifySaved(
                            await api.SaveAssetModel(
                              current.name,
                              data,
                              modelFormat,
                            ),
                          );
                        })
                      }
                    >
                      <Box />
                      {modelFormat === "bbmodel"
                        ? "Export Blockbench project"
                        : "Export displayed model"}
                    </Button>
                    <small>
                      {modelFormat === "bbmodel"
                        ? "Keeps source bones, individual elements, hidden parts, UVs, embedded textures and animation tracks editable. Preview poses are not baked."
                        : "The current pose and visible layers are included."}
                    </small>
                  </>
                )}
                <hr />
                {marketMatches.slice(0, 4).map((match) => (
                  <button
                    key={`${match.environment}:${match.source}:${match.item}`}
                    className="studio-button"
                    onClick={() => openMarketplace(match)}
                  >
                    View Marketplace offer /{" "}
                    {match.environment === "staging" ? "Staging" : "Production"}
                  </button>
                ))}
                <button
                  className="studio-button"
                  onClick={() =>
                    openPacketEvidence({
                      capture:
                        typeof detail?.metadata.packetCapture === "string"
                          ? detail.metadata.packetCapture
                          : undefined,
                      packet:
                        typeof detail?.metadata.packetSequence === "string"
                          ? detail.metadata.packetSequence
                          : undefined,
                      search:
                        current.playerUUID || current.uuid || current.name,
                    })
                  }
                >
                  Inspect related packets
                </button>
                <Button
                  variant="ghost"
                  onClick={() =>
                    void run("Opening folder", () =>
                      api.OpenAssetFolder(active),
                    )
                  }
                >
                  <FolderOpen />
                  Open library copy
                </Button>
              </>
            ) : (
              <p>Select an item to see its properties.</p>
            )}
          </aside>
        </div>
        {dragging && (
          <div className="asset-drop-overlay">
            <Import size={35} />
            <strong>Drop files to add them to the library</strong>
            <span>PNG skins, mcpack and ZIP files</span>
          </div>
        )}
        <Dialog
          lockWhileBusy={!!busy}
          open={confirmDelete !== null}
          onOpenChange={(open) => {
            if (!open) setConfirmDelete(null);
          }}
        >
          <DialogContent resizable={false} tone="confirmation">
            <DialogHeader>
              <DialogTitle>
                Delete{" "}
                {confirmDelete?.length === 1
                  ? "this item"
                  : `${confirmDelete?.length ?? 0} items`}{" "}
                from the library?
              </DialogTitle>
              <DialogDescription>
                This removes {bytes(deleteBytes)} of CraftBox copies. Original
                files and installed Minecraft packs stay available.
              </DialogDescription>
            </DialogHeader>
            <ul className="asset-delete-targets">
              {deleteItems.slice(0, 4).map((item) => (
                <li key={item.id}>{item.name}</li>
              ))}
              {deleteItems.length > 4 && (
                <li>and {deleteItems.length - 4} more</li>
              )}
            </ul>
            <DialogFooter>
              <Button
                variant="outline"
                disabled={!!busy}
                onClick={() => setConfirmDelete(null)}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                disabled={!!busy || !confirmDelete?.length}
                onClick={() =>
                  void run("Deleting library copies", async () => {
                    const targets = [...(confirmDelete ?? [])];
                    await api.DeleteAssets(targets);
                    setConfirmDelete(null);
                    setSelected((old) =>
                      old.filter((id) => !targets.includes(id)),
                    );
                    setStackIDs((old) =>
                      old.filter((id) => !targets.includes(id)),
                    );
                    if (targets.includes(active)) setActive("");
                    await refresh();
                  })
                }
              >
                Delete library copies
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Dialog
          lockWhileBusy={!!busy}
          open={exportOpen}
          onOpenChange={setExportOpen}
        >
          <DialogContent className="asset-export-dialog">
            <DialogHeader>
              <DialogTitle>
                Export {skin ? "skins" : "resource packs"}
              </DialogTitle>
              <DialogDescription>
                {ids.length} selected · {bytes(selectedBytes)} of source files
              </DialogDescription>
            </DialogHeader>
            <div className="asset-dialog-body">
              <label>
                Export name
                <input
                  value={exportName}
                  onChange={(e) => setExportName(e.target.value)}
                  maxLength={120}
                />
              </label>
              <label>
                Format
                <select
                  value={exportFormat}
                  onChange={(e) => setExportFormat(e.target.value)}
                >
                  {skin && ids.length === 1 && (
                    <option value="png">PNG · Skin texture</option>
                  )}
                  <option value="mcpack" disabled={!skin && ids.length > 1}>
                    MCPACK · Import into Minecraft
                  </option>
                  <option value="zip">ZIP · Original files and metadata</option>
                </select>
              </label>
              <p>
                {exportFormat === "mcpack"
                  ? skin
                    ? "Creates a named skin pack using classic wide or slim models. Custom geometry, capes and Character Creator parts remain available in the original-file ZIP."
                    : "Keeps the resource pack files and manifest for Minecraft import."
                  : exportFormat === "png"
                    ? "Exports the skin texture with transparency."
                    : "Preserves original textures, geometry and captured metadata. Multiple items use separate folders."}
              </p>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                disabled={!!busy}
                onClick={() => setExportOpen(false)}
              >
                Cancel
              </Button>
              <Button
                disabled={
                  !!busy ||
                  !exportName.trim() ||
                  (!skin && ids.length > 1 && exportFormat === "mcpack")
                }
                onClick={() =>
                  void run("Exporting assets", async () => {
                    const path = await api.ExportAssets(
                      ids,
                      exportName,
                      exportFormat,
                    );
                    if (path) {
                      setExportOpen(false);
                      notifySaved(path);
                    }
                  })
                }
              >
                <Download />
                Export
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Dialog open={minecraftOpen} onOpenChange={setMinecraftOpen}>
          <DialogContent className="asset-minecraft-dialog">
            <DialogHeader>
              <DialogTitle>Minecraft on this computer</DialogTitle>
              <DialogDescription>
                Browse installed content or add your library selection to
                Minecraft.
              </DialogDescription>
            </DialogHeader>
            <div className="asset-dialog-body">
              <div className="asset-tools">
                <label>
                  Installation
                  <select
                    value={installation}
                    disabled={!!busy}
                    onChange={(e) => setInstallation(e.target.value)}
                  >
                    {!installations.length && (
                      <option>No Minecraft storage found</option>
                    )}
                    {installations.map((v) => (
                      <option key={v.id} value={v.id}>
                        {v.name} · {v.layout}
                      </option>
                    ))}
                  </select>
                </label>
                <Button
                  variant="ghost"
                  disabled={!!busy}
                  onClick={() =>
                    void run("Finding Minecraft", minecraftRefresh)
                  }
                >
                  <RefreshCw />
                  Refresh
                </Button>
              </div>
              {selectedInstallation ? (
                <>
                  <dl>
                    <dt>Minecraft version</dt>
                    <dd>
                      {selectedInstallation.gameVersion ||
                        "Version unavailable"}
                    </dd>
                    <dt>Installed package version</dt>
                    <dd>
                      {selectedInstallation.version || "Version unavailable"}
                    </dd>
                    {!selectedInstallation.storageAvailable && (
                      <>
                        <dt>Storage status</dt>
                        <dd>
                          Start Minecraft once, or check access to its data
                          folder, then refresh.
                        </dd>
                      </>
                    )}
                    <dt>Storage folder</dt>
                    <dd className="asset-mono">
                      {selectedInstallation.storage}
                    </dd>
                    <dt>Channel</dt>
                    <dd>
                      {selectedInstallation.channel === "preview"
                        ? "Minecraft Preview"
                        : "Minecraft for Windows"}
                    </dd>
                  </dl>
                  <p>
                    Added resource packs appear in Minecraft's resource pack
                    settings. Added skin packs appear under Classic Skins.
                    Reopen the world or restart Minecraft to refresh its
                    content.
                  </p>
                  <p>
                    Existing packs are kept. CraftBox does not change your
                    selected skin or global pack order.
                  </p>
                  {skin && (
                    <p>
                      Custom models and Character Creator parts are retained in
                      CraftBox; installation uses classic skin models.
                    </p>
                  )}
                </>
              ) : (
                <p>
                  Install and start Minecraft for Windows once so its content
                  folder exists. Both GDK and older UWP locations are checked.
                </p>
              )}
              {minecraftWarning && (
                <FeedbackFlyout title={skin ? "Player skins" : "Resource packs"} tone="warning">{minecraftWarning}</FeedbackFlyout>
              )}
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                disabled={
                  !!busy ||
                  !installation ||
                  !selectedInstallation?.storageAvailable
                }
                onClick={() =>
                  void run("Reading Minecraft content", async () => {
                    await acceptImports(
                      await api.ImportMinecraftAssets(installation),
                    );
                  })
                }
              >
                Browse installed content
              </Button>
              <Button
                variant="outline"
                disabled={!!busy || !selectedInstallation?.application}
                onClick={() =>
                  void run("Opening Minecraft", () =>
                    api.OpenMinecraftInstallation(installation),
                  )
                }
              >
                Open Minecraft
              </Button>
              <Button
                disabled={
                  !!busy ||
                  !installation ||
                  !ids.length ||
                  (!skin && ids.length > 1)
                }
                onClick={() =>
                  void run("Adding selection to Minecraft", async () => {
                    const path = await api.InstallAssetsInMinecraft(
                      ids,
                      installation,
                    );
                    notify({
                      tone: "success",
                      title: "Added to Minecraft",
                      description: path,
                    });
                    setMinecraftOpen(false);
                  })
                }
              >
                Add selection to Minecraft
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </section>
    </DocumentScope.Provider>
  );
}
