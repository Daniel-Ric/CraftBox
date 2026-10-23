import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Play,
  Pause,
  SkipBack,
  SkipForward,
  Plus,
  Trash2,
  Undo2,
  Redo2,
  Download,
  Upload,
  Video,
  Camera,
  Copy,
} from "lucide-react";
import RenderCanvas, {
  type RenderCanvasHandle,
  type RenderCanvasProps,
} from "./RenderCanvas";
import {
  createSequence,
  sampleSequence,
  sampleCameraSequence,
  parseSequence,
  timecode,
  type CameraKey,
  type Sequence,
  type Vec3,
} from "./cinematic";
import { MediaExportDialog, type MediaFormat } from "./MediaDialogs";
import PanelResizeHandle from "./PanelResizeHandle";
import "./cinematic.css";
import "./cinematic-audio.css";
import { exportCinematic, saveCinematic } from "./cinematic-export";
import { useEditorCommands, dispatchEditorCommand } from "./editor-commands";
import CinematicTimeline from "./CinematicTimeline";
import {
  independentSequence,
  editPoseKey,
  addChannelKey,
  changeSequenceFPS,
  insertPoseKey,
  removePoseKey,
  minimumSequenceDuration,
} from "./cinematic-tracks";
import CinematicLensControls from "./CinematicLensControls";
import CinematicShots from "./CinematicShots";
import {
  cameraSequence,
  applyCameraEdit,
  resolveShot,
} from "./cinematic-shots";
import CinematicTracking from "./CinematicTracking";
import {
  targetChannels,
  targetSequence,
  editTargetPosition,
} from "./cinematic-tracking";
import { motionDefaults, type MotionBlur } from "./cinematic-motion";
import CinematicAudio from "./CinematicAudio";
import CinematicAudioTimeline from "./CinematicAudioTimeline";
import { MarkerInspector, MarkerLane } from "./CinematicMarkers";
import { addMarker } from "./cinematic-markers";
import {
  audioDefaults,
  hasAudibleAudio,
  parseAudioTimeline,
} from "./cinematic-audio";
import {
  CinematicAudioPlayer,
  discardAudioAsset,
  importAudioAsset,
  releaseAudioCache,
} from "./cinematic-audio-runtime";
import {
  hasAudioDraft,
  isAudioDraft,
  loadAudioDraft,
  saveAudioDraft,
} from "./cinematic-draft";

export default forwardRef<RenderCanvasHandle, RenderCanvasProps>(
  function CinematicStudio(props, ref) {
    const [exportDialog, setExportDialog] = useState<MediaFormat | null>(null);
    const [timelineHeight, setTimelineHeight] = useState(260);
    const [exportPreview, setExportPreview] = useState<string>();
    useEffect(() => {
      if (exportDialog) {
        setPlaying(false);
        setExportPreview(canvas.current?.preview?.());
      }
    }, [exportDialog]);
    const storageKey = `craftbox-cinematic:${props.projectId ? "project:" + props.projectId : props.worldId}`;
    const [project, setProject] = useState<Sequence>(() => {
      const lens = {
        enabled: props.settings.dofEnabled,
        distance: props.settings.dofDistance,
        aperture: props.settings.dofAperture,
      };
      if (props.initialCinematic)
        return independentSequence(props.initialCinematic.sequence, lens);
      try {
        const saved = localStorage.getItem(storageKey);
        if (saved && !isAudioDraft(saved))
          return independentSequence(parseSequence(saved), lens);
        const created = createSequence();
        return independentSequence(
          {
            ...created,
            name: props.projectName || created.name,
            fps: props.projectFps || created.fps,
            duration: props.projectDuration || created.duration,
            keys: created.keys.map((key) => ({
              ...key,
              time:
                (key.time / created.duration) *
                (props.projectDuration || created.duration),
            })),
          },
          lens,
        );
      } catch {
        return independentSequence(createSequence(), lens);
      }
    });
    const [draftLoading, setDraftLoading] = useState(
      () => !props.initialCinematic && hasAudioDraft(storageKey),
    );
    const [draftFailed, setDraftFailed] = useState(false);
    const [importingAudio, setImportingAudio] = useState(false);
    const [preparingAudio, setPreparingAudio] = useState(false);
    const [audioPeak, setAudioPeak] = useState(0);
    const [audioPlayer] = useState(() => new CinematicAudioPlayer());
    const [audioTimelineShown, setAudioTimelineShown] = useState(false);
    const [audioClipId, setAudioClipId] = useState<string>();
    const [markerId, setMarkerId] = useState<string>();
    const importGeneration = useRef(0);
    const [cameraId, setCameraId] = useState<string | undefined>(
      props.initialCinematic?.cameraId,
    );
    const [program, setProgram] = useState(
      props.initialCinematic?.program ?? true,
    );
    const sequence = useMemo(
      () => cameraSequence(project, cameraId),
      [project, cameraId],
    );
    const [past, setPast] = useState<Sequence[]>([]),
      [future, setFuture] = useState<Sequence[]>([]);
    const [time, setTime] = useState(props.initialCinematic?.time ?? 0),
      [playing, setPlaying] = useState(false),
      [loop, setLoop] = useState(props.initialCinematic?.loop ?? true),
      [speed, setSpeed] = useState(props.initialCinematic?.speed ?? 1);
    const [selected, setSelected] = useState(
        props.initialCinematic?.selected ?? sequence.keys[0].id,
      ),
      [preview, setPreview] = useState(props.initialCinematic?.preview ?? true),
      [showPath, setShowPath] = useState(
        props.initialCinematic?.showPath ?? true,
      );
    const [zoom, setZoom] = useState(props.initialCinematic?.zoom ?? 1),
      [snap, setSnap] = useState(props.initialCinematic?.snap ?? true),
      [guides, setGuides] = useState(props.initialCinematic?.guides ?? true),
      [plane, setPlane] = useState("top");
    const [recording, setRecording] = useState(false),
      [message, setMessage] = useState("");
    const [cameraTab, setCameraTab] = useState("camera");
    const [pickingFocus, setPickingFocus] = useState(false);
    const [targetId, setTargetId] = useState<string>();
    const [editingTarget, setEditingTarget] = useState(false);
    const [pickingTarget, setPickingTarget] = useState(false);
    const selectedTarget =
      project.targets?.find((t) => t.id === targetId) ?? project.targets?.[0];
    const targetTimeline =
      editingTarget && selectedTarget
        ? targetSequence(project, selectedTarget)
        : undefined;
    const minimumDuration = Math.max(
      minimumSequenceDuration(sequence),
      ...(project.shots?.map((s) => s.end) ?? [0]),
      ...(project.markers?.map((m) => m.time + m.duration) ?? [0]),
      ...(props.settings.atmosphere.cloudLayers?.flatMap((l) =>
        l.tracks.flatMap((t) => t.keys.map((k) => k.time)),
      ) ?? [0]),
      ...(project.targets?.flatMap((t) =>
        t.tracks.flatMap((track) => track.keys.map((k) => k.time)),
      ) ?? [0]),
    );
    const [navigation, setNavigation] = useState<"orbit" | "fly">(
      props.initialCinematic?.navigation ?? "orbit",
    );
    const [navigationSpeed, setNavigationSpeed] = useState(
      props.initialCinematic?.navigationSpeed ?? 1,
    );
    const studioState = useRef({
      sequence: project,
      cameraId,
      program,
      time,
      selected,
      preview,
      showPath,
      zoom,
      snap,
      guides,
      loop,
      speed,
      navigation,
      navigationSpeed,
    });
    const draftSnapshot = useRef({
      project,
      key: storageKey,
      loading: draftLoading,
      failed: draftFailed,
    });
    draftSnapshot.current = {
      project,
      key: storageKey,
      loading: draftLoading,
      failed: draftFailed,
    };
    studioState.current = {
      sequence: project,
      cameraId,
      program,
      time,
      selected,
      preview,
      showPath,
      zoom,
      snap,
      guides,
      loop,
      speed,
      navigation,
      navigationSpeed,
    };
    useEffect(() => {
      props.onCinematicChange?.({
        sequence: project,
        cameraId,
        program,
        time,
        selected,
        preview,
        showPath,
        zoom,
        snap,
        guides,
        loop,
        speed,
        navigation,
        navigationSpeed,
      });
    }, [
      project,
      cameraId,
      program,
      time,
      selected,
      preview,
      showPath,
      zoom,
      snap,
      guides,
      loop,
      speed,
      navigation,
      navigationSpeed,
    ]);
    const [encoding, setEncoding] = useState(false),
      [exportPercent, setExportPercent] = useState(0),
      [exportEdge, setExportEdge] = useState(props.settings.edge),
      [gifEdge, setGifEdge] = useState(640),
      [gifFps, setGifFps] = useState(12),
      [ready, setReady] = useState(false);
    const abort = useRef<AbortController | null>(null);
    const canvas = useRef<RenderCanvasHandle>(null),
      root = useRef<HTMLDivElement>(null);
    const file = useRef<HTMLInputElement>(null);
    const selectedPose =
      sequence.keys.find((k) => k.id === selected) ?? sequence.keys[0];
    const key = {
      ...selectedPose,
      ...sampleCameraSequence(
        { ...sequence, tracking: undefined },
        selectedPose.time,
      ),
    };
    const sourcePose = useMemo(
      () => sampleCameraSequence(sequence, time),
      [sequence, time],
    );
    const pose = useMemo(
      () => (program ? sampleSequence(project, time) : sourcePose),
      [project, time, program, sourcePose],
    );
    useEffect(() => {
      props.onHistoryChange?.({
        undo: past.length > 0,
        redo: future.length > 0,
      });
    }, [past.length, future.length, props.onHistoryChange]);
    useEditorCommands((command) => {
      if (recording) return;
      if (command === "undo") undo();
      else if (command === "redo") redo();
      else if (command === "save-project" && !props.onCinematicChange)
        void saveCinematic(
          new Blob([JSON.stringify(project, null, 2)], {
            type: "application/json",
          }),
          sequence.name + ".cinematic",
          "json",
        )
          .then((path) =>
            setMessage(path ? `Saved: ${path}` : "Save canceled"),
          )
          .catch((e) => setMessage(String(e)));
      else if (command === "play") {
        setPreview(true);
        setProgram(true);
        setPlaying((v) => !v);
      } else if (command === "add-key" && ready) addKey();
      else if (command === "toggle-path") {
        setShowPath((v) => !v);
        setPreview(false);
        setPlaying(false);
      } else if (command === "export-gif" && ready) setExportDialog("gif");
      else if (command === "export-mp4" && ready) setExportDialog("mp4");
      else if (command === "export-webm" && ready) setExportDialog("webm");
      else if (["fit", "isometric", "top", "inspect-view"].includes(command)) {
        setPreview(false);
        setPlaying(false);
      }
    });
    useImperativeHandle(
      ref,
      () => ({
        studioState: () => {
          setPlaying(false);
          return studioState.current;
        },
        seekTime: (t) => {
          const s = studioState.current;
          setPlaying(false);
          setProgram(true);
          setTime(
            Math.max(
              0,
              Math.min(
                s.sequence.duration,
                s.snap ? Math.round(t * s.sequence.fps) / s.sequence.fps : t,
              ),
            ),
          );
        },
        analyse: (progress) => canvas.current!.analyse(progress),
        capture: () => canvas.current!.capture(),
        fit: () => canvas.current?.fit(),
        view: (a, t) => canvas.current?.view(a, t),
        snapshot: () => canvas.current?.snapshot(),
        preview: () => canvas.current?.preview?.(),
        renderFrame: (pose, w, h, t, samples, signal) =>
          canvas.current!.renderFrame(pose, w, h, t, samples, signal),
        finishExport: () => canvas.current?.finishExport(),
      }),
      [],
    );
    useEffect(() => {
      setExportEdge(props.settings.edge);
    }, [props.settings.edge]);
    useEffect(() => {
      props.onExporting?.(recording || draftLoading || importingAudio);
    }, [recording, draftLoading, importingAudio, props.onExporting]);
    useEffect(() => () => props.onExporting?.(false), [props.onExporting]);
    useEffect(() => {
      if (!draftLoading) return;
      let current = true;
      void loadAudioDraft(storageKey)
        .then((s) => {
          if (current) {
            setProject(independentSequence(s));
            setDraftLoading(false);
          }
        })
        .catch((error) => {
          if (current) {
            setMessage(String(error));
            setDraftFailed(true);
            setDraftLoading(false);
          }
        });
      return () => {
        current = false;
      };
    }, [storageKey, draftLoading]);
    useEffect(() => {
      releaseAudioCache(project.audio);
    }, [project.audio?.assets]);
    useEffect(
      () => () => {
        importGeneration.current++;
        audioPlayer.dispose();
        releaseAudioCache();
        const draft = draftSnapshot.current;
        if (
          !draft.loading &&
          !draft.failed &&
          draft.project.audio?.assets.length
        )
          void saveAudioDraft(draft.key, draft.project).catch(() => {});
      },
      [audioPlayer],
    );
    useEffect(() => {
      if (draftLoading || draftFailed) return;
      if (project.audio?.assets.length) {
        let current = true;
        const timer = window.setTimeout(() => {
          void saveAudioDraft(storageKey, project, () => current).catch(
            (error) => {
              if (current)
                setMessage(
                  `Could not save audio draft: ${String(error)}. Save the project to a file.`,
                );
            },
          );
        }, 250);
        return () => {
          current = false;
          window.clearTimeout(timer);
        };
      }
      try {
        localStorage.setItem(storageKey, JSON.stringify(project));
      } catch {
        setMessage(
          "Local project storage is full. Save the project to a file.",
        );
      }
    }, [project, storageKey, draftLoading, draftFailed]);
    useEffect(
      () => () => {
        abort.current?.abort();
      },
      [],
    );
    useEffect(() => {
      if (!playing) {
        audioPlayer.stop();
        setAudioPeak(0);
        setPreparingAudio(false);
        return;
      }
      let frame = 0,
        current = true;
      const start = studioState.current.time;
      const begin = async () => {
        const audible = hasAudibleAudio(project.audio);
        setPreparingAudio(audible);
        const audioClock = audible
          ? await audioPlayer.play(
              project.audio!,
              start,
              sequence.duration,
              speed,
              loop,
            )
          : false;
        if (!current) return;
        setPreparingAudio(false);
        const anchor = performance.now();
        const tick = (now: number) => {
          if (!current) return;
          const next = audioClock
            ? audioPlayer.time()
            : start + ((now - anchor) / 1000) * speed;
          if (next >= sequence.duration && !loop) {
            setTime(sequence.duration);
            setPlaying(false);
            return;
          }
          setTime(
            loop ? next % sequence.duration : Math.min(sequence.duration, next),
          );
          setAudioPeak(audioClock ? audioPlayer.peak() : 0);
          frame = requestAnimationFrame(tick);
        };
        frame = requestAnimationFrame(tick);
      };
      void begin().catch((error) => {
        if (current) {
          setMessage(String(error));
          setPreparingAudio(false);
          setPlaying(false);
        }
      });
      return () => {
        current = false;
        cancelAnimationFrame(frame);
        audioPlayer.stop();
      };
    }, [playing, project.audio, sequence.duration, loop, speed, audioPlayer]);
    function commit(next: Sequence, baseline = sequence) {
      setDraftFailed(false);
      setPast((p) => [
        ...p.slice(-49),
        applyCameraEdit(project, baseline, cameraId),
      ]);
      setFuture([]);
      setProject(applyCameraEdit(project, next, cameraId));
      setProgram(false);
    }
    function commitProject(next: Sequence, baseline = project) {
      setDraftFailed(false);
      setPast((p) => [...p.slice(-49), baseline]);
      setFuture([]);
      setProject(next);
    }
    function undo() {
      if (!past.length) return;
      setFuture((f) => [project, ...f]);
      setProject(past[past.length - 1]);
      setPast((p) => p.slice(0, -1));
    }
    function redo() {
      if (!future.length) return;
      setPast((p) => [...p, project]);
      setProject(future[0]);
      setFuture((f) => f.slice(1));
    }
    function updateKey(change: Partial<CameraKey>) {
      commit(editPoseKey(sequence, key.id, change));
    }
    function addKey(fromView = false) {
      const value = fromView
        ? (canvas.current?.snapshot() ?? sourcePose)
        : sourcePose;
      const k: CameraKey = {
        ...value,
        id: crypto.randomUUID(),
        time: Math.min(
          sequence.duration,
          Math.round(time * sequence.fps) / sequence.fps,
        ),
        ease: "smooth",
      };
      commit(insertPoseKey(sequence, k));
      setSelected(k.id);
    }
    function preset(kind: string) {
      const s = createSequence();
      s.duration = sequence.duration;
      s.name = sequence.name;
      s.fps = sequence.fps;
      s.keys = Array.from({ length: 5 }, (_, i) => {
        const t = i / 4,
          angle = t * Math.PI * 1.7;
        const position: Vec3 =
          kind === "orbit"
            ? [Math.sin(angle) * 2.4, 0.65, Math.cos(angle) * 2.4]
            : kind === "dolly"
              ? [0, 0.3, 3.2 - t * 1.8]
              : [1.7 - t * 0.5, 0.15 + t * 2.2, 2.2];
        return {
          ...s.keys[0],
          id: crypto.randomUUID(),
          time: t * s.duration,
          position,
          ease: "linear",
        };
      });
      commit({ ...sequence, ...independentSequence(s, sequence.lens) });
      setTime(0);
      setSelected(s.keys[0].id);
      setPreview(true);
    }
    function addSequenceMarker() {
      const next = addMarker(project, time),
        created = next.markers?.find(
          (m) => !project.markers?.some((old) => old.id === m.id),
        );
      commitProject(next);
      if (created) setMarkerId(created.id);
      setCameraTab("markers");
    }
    async function importAudio(file: File) {
      const generation = ++importGeneration.current;
      setImportingAudio(true);
      setPlaying(false);
      setMessage("");
      try {
        const asset = await importAudioAsset(file);
        if (generation !== importGeneration.current) {
          discardAudioAsset(asset.id);
          return;
        }
        const latest = studioState.current.sequence,
          base = latest.audio ?? audioDefaults();
        if (base.clips.length >= 128)
          throw new Error("You can add up to 128 audio clips.");
        const existing = base.assets.find((a) => a.data === asset.data),
          material = existing ?? asset;
        const tracks = base.tracks.length
          ? base.tracks
          : audioDefaults().tracks;
        const clip = {
          id: crypto.randomUUID(),
          name: material.name,
          assetId: material.id,
          trackId: tracks[0].id,
          start: studioState.current.time,
          in: 0,
          out: material.duration,
          rate: 1,
          gainDb: 0,
          pan: 0,
          fadeIn: 0,
          fadeOut: 0,
          muted: false,
        };
        const audio = {
          ...base,
          tracks,
          assets: existing ? base.assets : [...base.assets, asset],
          clips: [...base.clips, clip],
        };
        parseAudioTimeline(audio);


        releaseAudioCache(audio);
        commitProject({ ...latest, audio }, latest);
        setAudioClipId(clip.id);
        setAudioTimelineShown(true);
        setEditingTarget(false);
        setCameraTab("audio");
      } catch (error) {
        if (generation === importGeneration.current) {
          releaseAudioCache(studioState.current.sequence.audio);
          setMessage(String(error));
        }
      } finally {
        if (generation === importGeneration.current) setImportingAudio(false);
      }
    }
    async function encode(
      kind: "gif" | "mp4" | "webm",
      filename = sequence.name,
    ) {
      const controller = new AbortController();
      abort.current = controller;
      setEncoding(true);
      setRecording(true);
      setPlaying(false);
      setPreview(true);
      setExportPercent(0);
      setMessage("");
      try {
        const blob = await exportCinematic({
          kind,
          sequence: project,
          edge: kind === "gif" ? gifEdge : exportEdge,
          aspect: props.settings.aspect,
          gifFps,
          signal: controller.signal,
          render: (pose, w, h, t, samples) =>
            canvas.current!.renderFrame(
              pose,
              w,
              h,
              t,
              samples,
              controller.signal,
            ),
          progress: (t, p) => {
            setTime(t);
            setExportPercent(p);
          },
        });
        controller.signal.throwIfAborted();
        const path = await saveCinematic(blob, filename, kind);
        setMessage(path ? `Saved: ${path}` : "Save canceled");
      } catch (e) {
        setMessage(
          controller.signal.aborted ? "Export canceled" : String(e),
        );
      } finally {
        canvas.current?.finishExport();
        abort.current = null;
        setEncoding(false);
        setRecording(false);
      }
    }
    const selectKey = (k: CameraKey) => {
      setSelected(k.id);
      setTime(k.time);
      setPlaying(false);
    };
    const scrub = (t: number) => {
      setPlaying(false);
      setTime(
        Math.max(
          0,
          Math.min(
            sequence.duration,
            snap ? Math.round(t * sequence.fps) / sequence.fps : t,
          ),
        ),
      );
    };
    const path = useMemo(
      () =>
        Array.from(
          { length: 161 },
          (_, i) =>
            sampleSequence(sequence, (i / 160) * sequence.duration).position,
        ),
      [sequence],
    );
    const axes = plane === "top" ? [0, 2] : [0, 1];
    const all = [...path, ...sequence.keys.map((k) => k.target)];
    const low = axes.map((axis) => Math.min(...all.map((p) => p[axis])) - 0.3),
      high = axes.map((axis) => Math.max(...all.map((p) => p[axis])) + 0.3);
    const xy = (v: Vec3) => [
      20 + ((v[axes[0]] - low[0]) / (high[0] - low[0])) * 360,
      160 - ((v[axes[1]] - low[1]) / (high[1] - low[1])) * 140,
    ];
    return (
      <div
        className="cinema-studio"
        inert={draftLoading}
        aria-busy={draftLoading || importingAudio || preparingAudio}
        ref={root}
        onKeyDown={(e) => {
          if (
            (e.target as HTMLElement).matches("input,select,textarea") ||
            recording
          )
            return;
          if (e.code === "Space") {
            e.preventDefault();
            setPreview(true);
            setProgram(true);
            setPlaying((p) => !p);
          }
          if (
            e.key.toLowerCase() === "m" &&
            !e.ctrlKey &&
            !e.metaKey &&
            !e.altKey
          ) {
            e.preventDefault();
            addSequenceMarker();
          }
          if ((e.ctrlKey || e.metaKey) && e.key === "z") {
            e.preventDefault();
            if (e.shiftKey) redo();
            else undo();
          }
          if (e.key === "ArrowRight") {
            e.preventDefault();
            scrub(time + 1 / sequence.fps);
          }
          if (e.key === "ArrowLeft") {
            e.preventDefault();
            scrub(time - 1 / sequence.fps);
          }
        }}
        tabIndex={0}
      >
        {draftLoading && (
          <div className="cinematic-draft-loading" role="status">
            Loading audio project from local storage …
          </div>
        )}
        {preparingAudio && (
          <div className="cinematic-audio-preparing" role="status">
            Preparing audio for playback …
          </div>
        )}
        <MediaExportDialog
          open={exportDialog !== null}
          onOpenChange={(open) => {
            if (!open) setExportDialog(null);
          }}
          format={exportDialog || "mp4"}
          onFormat={setExportDialog}
          edge={exportDialog === "gif" ? gifEdge : exportEdge}
          onEdge={exportDialog === "gif" ? setGifEdge : setExportEdge}
          aspect={props.settings.aspect}
          name={sequence.name}
          preview={exportPreview}
          duration={sequence.duration}
          fps={sequence.fps}
          gifFps={gifFps}
          onGifFps={setGifFps}
          onExport={(name) => {
            if (
              exportDialog === "webm" ||
              exportDialog === "gif" ||
              exportDialog === "mp4"
            )
              void encode(exportDialog, name);
          }}
        />
        <div className="cinema-project">
          <input
            aria-label="Project name"
            value={sequence.name}
            disabled={recording}
            onChange={(e) =>
              commit({ ...sequence, name: e.target.value.slice(0, 200) })
            }
          />
          <button
            title="Undo"
            aria-label="Undo"
            disabled={!past.length || recording}
            onClick={undo}
          >
            <Undo2 size={15} />
          </button>
          <button
            title="Redo"
            aria-label="Redo"
            disabled={!future.length || recording}
            onClick={redo}
          >
            <Redo2 size={15} />
          </button>
          <button disabled={recording} onClick={() => file.current?.click()}>
            <Upload size={14} /> Open camera path
          </button>
          <button
            disabled={recording}
            onClick={() => {
              if (props.onCinematicChange) {
                dispatchEditorCommand("save-project");
                return;
              }
              void saveCinematic(
                new Blob([JSON.stringify(project, null, 2)], {
                  type: "application/json",
                }),
                sequence.name + ".cinematic",
                "json",
              )
                .then((path) =>
                  setMessage(
                    path ? `Saved: ${path}` : "Save canceled",
                  ),
                )
                .catch((e) => setMessage(String(e)));
            }}
          >
            <Download size={14} /> Project
          </button>
          <button
            disabled={recording || !ready}
            onClick={() => setExportDialog("webm")}
          >
            <Video size={14} /> WebM
          </button>
          <button
            disabled={recording || !ready}
            onClick={() => setExportDialog("gif")}
          >
            GIF
          </button>
          <button
            disabled={recording || !ready}
            onClick={() => setExportDialog("mp4")}
          >
            MP4
          </button>
          <input
            ref={file}
            type="file"
            accept=".json"
            hidden
            onChange={async (e) => {
              const f = e.target.files?.[0];
              if (!f) return;
              try {
                if (f.size > 64 * 1024 * 1024)
                  throw new Error("Project file is too large");
                const s = independentSequence(
                  parseSequence(await f.text()),
                  sequence.lens,
                );
                commitProject(s);
                setCameraId(undefined);
                setProgram(true);
                setTime(0);
                setSelected(s.keys[0].id);
                setPlaying(false);
              } catch (error) {
                setMessage(String(error));
              }
              e.target.value = "";
            }}
          />
        </div>
        <div className="cinema-editing">
          <div className="cinema-monitor">
            <div className="cinema-monitor-bar">
              <span>
                {preview
                  ? program
                    ? `Program · ${resolveShot(project, time)?.camera.name ?? "Camera"}`
                    : `Source · ${project.cameras?.find((c) => c.id === cameraId)?.name ?? project.cameras?.[0]?.name ?? "Camera"}`
                  : navigation === "fly"
                    ? "Scene · Free camera"
                    : "Scene · Edit path"}
              </span>
              {!!project.cameras?.length && (
                <button
                  aria-label="Show shot sequence"
                  aria-pressed={program}
                  disabled={recording}
                  onClick={() => {
                    setProgram((value) => !value);
                    setPreview(true);
                  }}
                >
                  {program ? "Edit camera" : "Show shot sequence"}
                </button>
              )}
              <span>
                {timecode(time, sequence.fps)} /{" "}
                {timecode(sequence.duration, sequence.fps)}
              </span>
            </div>
            <div className="cinema-viewport">
              <RenderCanvas
                {...props}
                targetEditor={{
                  selected: selectedTarget?.id,
                  picking: pickingTarget,
                  onSelect: (id) => {
                    setTargetId(id);
                    setCameraTab("tracking");
                  },
                  onPosition: (position) => {
                    if (selectedTarget)
                      commitProject(
                        editTargetPosition(
                          project,
                          selectedTarget.id,
                          time,
                          position,
                        ),
                      );
                    setPickingTarget(false);
                  },
                }}
                pickingFocus={pickingFocus || props.pickingFocus}
                onFocusPick={(distance) => {
                  commit(
                    addChannelKey(
                      {
                        ...sequence,
                        lens: {
                          enabled: true,
                          distance,
                          aperture:
                            sequence.lens?.aperture ??
                            props.settings.dofAperture,
                        },
                      },
                      "focusDistance",
                      time,
                      distance,
                    ),
                  );
                  setPickingFocus(false);
                  props.onFocusPick?.(distance);
                }}
                ref={canvas}
                cinematic
                onReady={() => {
                  setReady(true);
                  props.onReady();
                }}
                shot={{
                  pose,
                  sequence: program ? project : sequence,
                  preview,
                  showPath,
                  time,
                  navigation,
                  navigationSpeed,
                  onNavigationExit: () => setNavigation("orbit"),
                }}
              />
              {guides && preview && (
                <div
                  className="cinema-guides"
                  aria-hidden="true"
                  style={{
                    aspectRatio: props.settings.aspect,
                    width: `min(80cqw, ${80 * props.settings.aspect}cqh)`,
                  }}
                >
                  <i />
                  <i />
                  <i />
                  <i />
                </div>
              )}
              <div className="cinema-view-tools">
                {props.viewMenu}
                <button
                  disabled={recording}
                  aria-pressed={preview}
                  onClick={() => {
                    setNavigation("orbit");
                    setPreview((p) => !p);
                    setPlaying(false);
                  }}
                >
                  {preview ? "Edit 3D path" : "Camera preview"}
                </button>
                <button
                  disabled={recording}
                  aria-pressed={!preview && navigation === "fly"}
                  onClick={() => {
                    setPlaying(false);
                    setPreview(false);
                    setNavigation((n) =>
                      !preview && n === "fly" ? "orbit" : "fly",
                    );
                  }}
                >
                  Free camera
                </button>
                {!preview && (
                  <>
                    <button onClick={() => canvas.current?.fit()}>
                      Fit to view
                    </button>
                    <button onClick={() => addKey(true)}>
                      <Camera size={13} /> Add keyframe from view
                    </button>
                    <button
                      onClick={() => {
                        const value = canvas.current?.snapshot();
                        if (value) {
                          updateKey(value);
                          setTime(key.time);
                          setMessage(
                            "View applied to the selected keyframe.",
                          );
                        }
                      }}
                    >
                      Update keyframe
                    </button>
                  </>
                )}
              </div>
              {!preview && navigation === "fly" && (
                <div className="cinema-navigation-help">
                  <span>
                    Click the view · WASD to move · Q/E for height · Right-drag to look · Shift for speed · Alt for precision · Esc to orbit
                  </span>
                  <label>
                    Speed
                    <select
                      aria-label="Flight speed"
                      value={navigationSpeed}
                      onChange={(e) => setNavigationSpeed(+e.target.value)}
                    >
                      {[0.1, 0.25, 0.5, 1, 2, 4].map((v) => (
                        <option key={v} value={v}>
                          {v}×
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              )}
              {recording && (
                <div className="cinema-record">
                  {encoding ? "Export" : "Capture"}{" "}
                  {Math.round(
                    encoding ? exportPercent : (time / sequence.duration) * 100,
                  )}{" "}
                  %{" "}
                  <button
                    onClick={() =>
                      encoding ? abort.current?.abort() : abort.current?.abort()
                    }
                  >
                    {encoding ? "Cancel" : "Finish & save"}
                  </button>
                </div>
              )}
            </div>
            <div className="cinema-transport">
              <button
                aria-label="Go to start"
                disabled={recording}
                onClick={() => scrub(0)}
              >
                <SkipBack size={15} />
              </button>
              <button
                aria-label={playing ? "Pause" : "Play"}
                disabled={recording}
                onClick={() => {
                  if (time >= sequence.duration) setTime(0);
                  setPreview(true);
                  setProgram(true);
                  setPlaying((p) => !p);
                }}
              >
                {playing ? <Pause size={17} /> : <Play size={17} />}
              </button>
              <button
                aria-label="Go to end"
                disabled={recording}
                onClick={() => scrub(sequence.duration)}
              >
                <SkipForward size={15} />
              </button>
              <strong>{timecode(time, sequence.fps)}</strong>
              <select
                aria-label="Playback speed"
                disabled={recording}
                value={speed}
                onChange={(e) => setSpeed(+e.target.value)}
              >
                {[0.25, 0.5, 1, 2].map((s) => (
                  <option key={s} value={s}>
                    {s}×
                  </option>
                ))}
              </select>
              <label>
                <input
                  type="checkbox"
                  checked={loop}
                  onChange={(e) => setLoop(e.target.checked)}
                />{" "}
                Loop
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={guides}
                  onChange={(e) => setGuides(e.target.checked)}
                />{" "}
                Composition guides
              </label>
            </div>
          </div>
          <aside className={`cinema-inspector camera-tab-${cameraTab}`}>
            <header className="cinema-inspector-title">
              <strong>Camera &amp; sequence</strong>
              <span>Perspective</span>
            </header>
            <nav
              className="cinema-inspector-tabs"
              role="tablist"
              aria-label="Camera controls"
            >
              {[
                ["camera", "Camera"],
                ["shots", "Shots"],
                ["tracking", "Target"],
                ["motion", "Motion"],
                ["audio", "Audio"],
                ["markers", "Marker"],
                ["sequence", "Sequence"],
                ["path", "Path"],
                ["export", "Export"],
              ].map(([id, label]) => (
                <button
                  role="tab"
                  aria-selected={cameraTab === id}
                  key={id}
                  onClick={() => setCameraTab(id)}
                >
                  {label}
                </button>
              ))}
            </nav>
            <fieldset disabled={recording}>
              <CinematicAudio
                audio={project.audio}
                selected={audioClipId}
                importing={importingAudio}
                disabled={recording || importingAudio}
                time={time}
                peak={audioPeak}
                onImport={(file) => {
                  void importAudio(file);
                }}
                onSelect={setAudioClipId}
                onChange={(audio) => commitProject({ ...project, audio })}
                onTimeline={() => {
                  setAudioTimelineShown(true);
                  setEditingTarget(false);
                }}
              />
              <MarkerInspector
                project={project}
                time={time}
                selected={markerId}
                disabled={recording}
                onSelect={setMarkerId}
                onScrub={scrub}
                onChange={commitProject}
              />
              <CinematicShots
                project={project}
                cameraId={cameraId}
                time={time}
                disabled={recording}
                onChange={commitProject}
                onCamera={(id) => {
                  setCameraId(id);
                  setProgram(false);
                  setPlaying(false);
                  setPreview(true);
                }}
                onScrub={(time) => {
                  scrub(time);
                  setProgram(true);
                }}
              />
              <section data-camera-panel="motion">
                <h3>Motion blur</h3>
                <label>
                  <input
                    aria-label="Enable motion blur"
                    type="checkbox"
                    checked={project.motionBlur?.enabled ?? false}
                    onChange={(e) =>
                      commitProject({
                        ...project,
                        motionBlur: {
                          ...motionDefaults,
                          ...project.motionBlur,
                          enabled: e.target.checked,
                        },
                      })
                    }
                  />
                  Temporal motion blur
                </label>
                {(
                  [
                    {
                      field: "angle",
                      label: "Shutter angle",
                      min: 0,
                      max: 720,
                      step: 1,
                    },
                    {
                      field: "phase",
                      label: "Shutter phase",
                      min: -720,
                      max: 720,
                      step: 1,
                    },
                    {
                      field: "samples",
                      label: "Temporal samples per frame",
                      min: 2,
                      max: 64,
                      step: 1,
                    },
                  ] as const
                ).map((c) => (
                  <label key={c.field}>
                    {c.label}
                    <input
                      aria-label={c.label}
                      type="number"
                      min={c.min}
                      max={c.max}
                      step={c.step}
                      value={(project.motionBlur ?? motionDefaults)[c.field]}
                      onChange={(e) => {
                        if (
                          !e.target.value ||
                          !Number.isFinite(+e.target.value)
                        )
                          return;
                        const value = Math.max(
                          c.min,
                          Math.min(c.max, +e.target.value),
                        );
                        commitProject({
                          ...project,
                          motionBlur: {
                            ...motionDefaults,
                            ...project.motionBlur,
                            [c.field]:
                              c.field === "samples" ? Math.round(value) : value,
                          } as MotionBlur,
                        });
                      }}
                    />
                  </label>
                ))}
                <label>
                  <input
                    aria-label="Motion blur preview"
                    type="checkbox"
                    checked={project.motionBlur?.preview ?? false}
                    onChange={(e) =>
                      commitProject({
                        ...project,
                        motionBlur: {
                          ...motionDefaults,
                          ...project.motionBlur,
                          preview: e.target.checked,
                        },
                      })
                    }
                  />
                  Enable in preview (up to 8 samples)
                </label>
                <p>
                  180° exposes half a frame; −90° centers that interval. GIF, MP4 and WebM use the selected sample count regardless of the preview toggle. Samples never cross shot cuts.
                </p>
              </section>
              <CinematicTracking
                project={project}
                source={sequence}
                pose={sourcePose}
                time={time}
                selected={selectedTarget?.id}
                editing={editingTarget}
                picking={pickingTarget}
                disabled={recording}
                onProject={(next) => {
                  commitProject(next);
                  setPlaying(false);
                }}
                onSource={commit}
                onSelect={(id) => {
                  setTargetId(id);
                  setPickingTarget(false);
                }}
                onEdit={(edit) => {
                  setAudioTimelineShown(false);
                  setEditingTarget(edit);
                  setPlaying(false);
                }}
                onPick={() => {
                  setPickingTarget((value) => !value);
                  setPickingFocus(false);
                  setPlaying(false);
                }}
              />
              <CinematicLensControls
                lens={
                  sequence.lens ?? {
                    enabled: props.settings.dofEnabled,
                    distance: props.settings.dofDistance,
                    aperture: props.settings.dofAperture,
                  }
                }
                pose={sourcePose}
                scale={props.settings.dofWorldScale}
                picking={pickingFocus}
                disabled={recording}
                onChange={(lens) => commit({ ...sequence, lens })}
                onKey={(field, value) =>
                  commit(
                    addChannelKey(
                      {
                        ...sequence,
                        lens: {
                          enabled: true,
                          distance:
                            sequence.lens?.distance ??
                            props.settings.dofDistance,
                          aperture:
                            sequence.lens?.aperture ??
                            props.settings.dofAperture,
                        },
                      },
                      field,
                      time,
                      value,
                    ),
                  )
                }
                onPick={() => {
                  setPickingFocus((p) => !p);
                  setPlaying(false);
                  setPreview(true);
                  setProgram(false);
                }}
                onScale={props.onLensScale}
              />
              <section data-camera-panel="sequence">
                <h3>Sequence</h3>
                <div className="cinema-pair">
                  <label>
                    Duration · s
                    <input
                      aria-label="Sequence duration"
                      type="number"
                      min={minimumDuration}
                      max={3600}
                      step={0.1}
                      value={sequence.duration}
                      onChange={(e) => {
                        const duration = +e.target.value;
                        if (duration >= minimumDuration && duration <= 3600)
                          commit({ ...sequence, duration });
                      }}
                    />
                  </label>
                  <label>
                    Frame rate
                    <select
                      aria-label="Frame rate"
                      value={sequence.fps}
                      onChange={(e) => {
                        const fps = +e.target.value;
                        commit(changeSequenceFPS(sequence, fps));
                      }}
                    >
                      {[24, 25, 30, 50, 60].map((f) => (
                        <option key={f}>{f}</option>
                      ))}
                    </select>
                  </label>
                </div>
                <div className="cinema-preset">
                  {[
                    ["orbit", "Orbit"],
                    ["dolly", "Dolly-in"],
                    ["crane", "Crane move"],
                  ].map(([id, label]) => (
                    <button key={id} onClick={() => preset(id)}>
                      {label}
                    </button>
                  ))}
                </div>
              </section>
              <section data-camera-panel="export">
                <h3>Export</h3>
                <label>
                  MP4 · Resolution
                  <select
                    aria-label="MP4 resolution"
                    value={exportEdge}
                    onChange={(e) => setExportEdge(+e.target.value)}
                  >
                    {[720, 1280, 1920, 2560, 3840].map((n) => (
                      <option key={n} value={n}>
                        {n} pixels
                      </option>
                    ))}
                  </select>
                </label>
                <div className="cinema-pair">
                  <label>
                    GIF · Size
                    <select
                      aria-label="GIF size"
                      value={gifEdge}
                      onChange={(e) => setGifEdge(+e.target.value)}
                    >
                      {[320, 480, 640, 960].map((n) => (
                        <option key={n} value={n}>
                          {n} px
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    GIF · Frame rate
                    <select
                      aria-label="GIF frame rate"
                      value={gifFps}
                      onChange={(e) => setGifFps(+e.target.value)}
                    >
                      {[10, 12, 15, 20, 25].map((n) => (
                        <option key={n}>{n}</option>
                      ))}
                    </select>
                  </label>
                </div>
                <p>
                  MP4: H.264, rendered frame by frame up to 4K. GIF: 256 colors. WebM: frame-accurate VP9/VP8 output.
                </p>
              </section>
              <section data-camera-panel="camera">
                <h3>
                  Keyframe{" "}
                  <span>
                    {sequence.keys.indexOf(selectedPose) + 1} /{" "}
                    {sequence.keys.length}
                  </span>
                </h3>
                <label>
                  Time · s
                  <input
                    aria-label="Keyframe time"
                    type="number"
                    min={0}
                    max={sequence.duration}
                    step={1 / sequence.fps}
                    value={key.time}
                    onChange={(e) => {
                      const time = +e.target.value;
                      if (time >= 0 && time <= sequence.duration)
                        updateKey({ time });
                    }}
                  />
                </label>
                {(["position", "target"] as const).map((field) => (
                  <div key={field}>
                    <h4>
                      {field === "position" ? "Camera position" : "Look-at target"}{" "}
                      <small>relative to world size</small>
                    </h4>
                    <div className="cinema-vector">
                      {["X", "Y", "Z"].map((axis, i) => (
                        <label key={axis}>
                          {axis}
                          <input
                            aria-label={`${field} ${axis}`}
                            type="number"
                            min={-10000}
                            max={10000}
                            step={0.05}
                            value={Number(key[field][i].toFixed(4))}
                            onChange={(e) => {
                              const n = +e.target.value;
                              if (!Number.isFinite(n) || Math.abs(n) > 10000)
                                return;
                              const v = [...key[field]] as Vec3;
                              v[i] = n;
                              updateKey({ [field]: v });
                            }}
                          />
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
                {(
                  [
                    {
                      field: "fov",
                      label: "Field of view",
                      min: 10,
                      max: 120,
                      step: 1,
                      unit: "°",
                    },
                    {
                      field: "roll",
                      label: "Roll angle",
                      min: Math.min(-180, key.roll - 180),
                      max: Math.max(180, key.roll + 180),
                      step: 1,
                      unit: "°",
                    },
                    {
                      field: "exposure",
                      label: "Exposure",
                      min: 0.1,
                      max: 4,
                      step: 0.05,
                      unit: "×",
                    },
                  ] as const
                ).map((control) => (
                  <label className="cinema-slider" key={control.field}>
                    <span>
                      {control.label}
                      <output>
                        {key[control.field].toFixed(
                          control.field === "exposure" ? 2 : 0,
                        )}
                        {control.unit}
                      </output>
                    </span>
                    <input
                      aria-label={control.label}
                      type="range"
                      min={control.min}
                      max={control.max}
                      step={control.step}
                      value={key[control.field]}
                      onChange={(e) =>
                        updateKey({ [control.field]: +e.target.value })
                      }
                    />
                  </label>
                ))}
                <label>
                  Interpolation
                  <select
                    aria-label="Interpolation"
                    value={key.ease}
                    onChange={(e) =>
                      updateKey({ ease: e.target.value as CameraKey["ease"] })
                    }
                  >
                    <option value="smooth">
                      Smooth · continuous camera motion
                    </option>
                    <option value="linear">Linear · constant rate</option>
                    <option value="hold">Hold · hard cut</option>
                  </select>
                </label>
                <div className="cinema-preset">
                  <button
                    aria-label="Duplicate keyframe"
                    onClick={() => {
                      const time = Math.min(
                        sequence.duration,
                        key.time + 1 / sequence.fps,
                      );
                      const k = {
                        ...key,
                        ...sampleSequence(sequence, key.time),
                        id: crypto.randomUUID(),
                        time,
                      };
                      commit(insertPoseKey(sequence, k));
                      setSelected(k.id);
                    }}
                  >
                    <Copy size={13} /> Copy
                  </button>
                  <button
                    aria-label="Delete keyframe"
                    disabled={sequence.keys.length < 2}
                    onClick={() => {
                      commit(removePoseKey(sequence, key.id));
                    }}
                  >
                    <Trash2 size={13} /> Delete
                  </button>
                </div>
              </section>
              <section data-camera-panel="path">
                <h3>Path view</h3>
                <div className="cinema-pair">
                  <select
                    aria-label="Path projection"
                    value={plane}
                    onChange={(e) => setPlane(e.target.value)}
                  >
                    <option value="top">Top · X/Z</option>
                    <option value="front">Front · X/Y</option>
                  </select>
                  <label>
                    <input
                      type="checkbox"
                      checked={showPath}
                      onChange={(e) => setShowPath(e.target.checked)}
                    />{" "}
                    3D path
                  </label>
                </div>
                <svg
                  className="cinema-path"
                  viewBox="0 0 400 180"
                  aria-label="Visual camera path"
                >
                  <path
                    d={path
                      .map((p, i) => `${i ? "L" : "M"}${xy(p).join(",")}`)
                      .join(" ")}
                    fill="none"
                    stroke="#b3a4ff"
                    strokeWidth="2"
                  />
                  {sequence.keys.map((k, i) => {
                    const [x, y] = xy(
                      sampleSequence(sequence, k.time).position,
                    );
                    return (
                      <g
                        key={k.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`Path point ${i + 1}`}
                        onClick={() => selectKey(k)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") selectKey(k);
                        }}
                      >
                        <circle
                          cx={x}
                          cy={y}
                          r={selected === k.id ? 8 : 6}
                          fill={selected === k.id ? "#ffd595" : "#9e95eb"}
                        />
                        <text x={x + 10} y={y + 4} fill="#d7d9e2" fontSize={12}>
                          {i + 1}
                        </text>
                      </g>
                    );
                  })}
                  <circle
                    cx={xy(pose.position)[0]}
                    cy={xy(pose.position)[1]}
                    r={4}
                    fill="white"
                  />
                </svg>
                <p>
                  Orbit the scene and use “Apply view” to save the current camera. Space: play · Arrow keys: one frame · Ctrl+Z: undo.
                </p>
              </section>
            </fieldset>
          </aside>
        </div>
        <PanelResizeHandle
          label="Timeline height"
          value={timelineHeight}
          min={160}
          max={Math.max(180, Math.floor(window.innerHeight * 0.5))}
          onChange={setTimelineHeight}
        />
        <div className="cinema-timeline" style={{ height: timelineHeight }}>
          <div className="cinema-timeline-toolbar">
            <strong>Timeline</strong>
            <span>{sequence.keys.length} Keyframes</span>
            <button
              aria-pressed={!audioTimelineShown}
              disabled={recording}
              onClick={() => setAudioTimelineShown(false)}
            >
              Animation
            </button>
            <button
              aria-pressed={audioTimelineShown}
              disabled={recording}
              onClick={() => {
                setAudioTimelineShown(true);
                setEditingTarget(false);
                setCameraTab("audio");
              }}
            >
              Audio
            </button>
            <button
              aria-label="Add sequence marker"
              disabled={recording}
              onClick={addSequenceMarker}
            >
              ⚑ Marker
            </button>
            <button
              aria-label="Add keyframe"
              disabled={recording}
              onClick={() => addKey()}
            >
              <Plus size={14} /> Keyframe
            </button>
            <label>
              <input
                type="checkbox"
                checked={snap}
                onChange={(e) => setSnap(e.target.checked)}
              />{" "}
              Snapping
            </label>
            <label>
              Zoom
              <input
                aria-label="Timeline zoom"
                type="range"
                min={1}
                max={8}
                step={0.5}
                value={zoom}
                onChange={(e) => setZoom(+e.target.value)}
              />
            </label>
          </div>
          <input
            className="cinema-scrub"
            aria-label="Playhead position"
            disabled={recording}
            type="range"
            min={0}
            max={sequence.duration}
            step={1 / sequence.fps}
            value={Math.min(time, sequence.duration)}
            onChange={(e) => scrub(+e.target.value)}
          />
          {!!project.shots?.length && (
            <div className="cinematic-shot-lane" aria-label="Shot track">
              {project.shots.map((shot) => (
                <button
                  key={shot.id}
                  disabled={recording}
                  style={{
                    left: `${(shot.start / project.duration) * 100}%`,
                    width: `${((shot.end - shot.start) / project.duration) * 100}%`,
                  }}
                  aria-pressed={
                    program && resolveShot(project, time)?.shot.id === shot.id
                  }
                  onClick={() => {
                    scrub(shot.start);
                    setProgram(true);
                    setCameraTab("shots");
                  }}
                >
                  {shot.name} ·{" "}
                  {project.cameras?.find((c) => c.id === shot.cameraId)?.name}
                </button>
              ))}
            </div>
          )}
          <MarkerLane
            project={project}
            selected={markerId}
            disabled={recording}
            onSelect={(id) => {
              setMarkerId(id);
              setCameraTab("markers");
            }}
            onScrub={scrub}
            onChange={commitProject}
            onPreview={(next) => {
              setPlaying(false);
              setProject(next);
            }}
          />
          {audioTimelineShown ? (
            <CinematicAudioTimeline
              project={project}
              time={time}
              selected={audioClipId}
              zoom={zoom}
              snap={snap}
              disabled={recording}
              onSelect={(id) => {
                setAudioClipId(id);
                setCameraTab("audio");
              }}
              onScrub={scrub}
              onChange={commitProject}
              onPreview={(next) => {
                setPlaying(false);
                setProject(next);
              }}
            />
          ) : targetTimeline && selectedTarget ? (
            <>
              <div className="animation-source-label">
                Target · {selectedTarget.name}
                <button onClick={() => setEditingTarget(false)}>
                  Camera tracks
                </button>
              </div>
              <CinematicTimeline
                key={selectedTarget.id}
                sequence={targetTimeline}
                channels={targetChannels}
                time={time}
                zoom={zoom}
                snap={snap}
                disabled={recording}
                onScrub={scrub}
                onChange={(next, before) => {
                  const replace = (s: Sequence) => ({
                    ...project,
                    targets: project.targets!.map((t) =>
                      t.id === selectedTarget.id
                        ? { ...t, tracks: s.tracks! }
                        : t,
                    ),
                  });
                  setPast((p) => [
                    ...p.slice(-49),
                    before ? replace(before) : project,
                  ]);
                  setFuture([]);
                  setProject(replace(next));
                }}
                onPreview={(next) => {
                  setPlaying(false);
                  setProject({
                    ...project,
                    targets: project.targets!.map((t) =>
                      t.id === selectedTarget.id
                        ? { ...t, tracks: next.tracks! }
                        : t,
                    ),
                  });
                }}
              />
            </>
          ) : (
            <CinematicTimeline
              sequence={sequence}
              time={time}
              zoom={zoom}
              snap={snap}
              disabled={recording}
              onScrub={(time) => {
                scrub(time);
                setProgram(false);
              }}
              onChange={commit}
              onPreview={(next) => {
                setPlaying(false);
                setProgram(false);
                setProject(applyCameraEdit(project, next, cameraId));
              }}
            />
          )}
        </div>
        <footer className="cinema-status">
          <span>
            {message ||
              "Project saved locally · GIF, MP4 and WebM are rendered frame by frame"}
          </span>
          <span>
            {sequence.fps} fps · {Math.round(sequence.duration * sequence.fps)}{" "}
            Frames
          </span>
        </footer>
      </div>
    );
  },
);
