import { useEffect, useRef, useState } from "react";
import {
  Box,
  FolderOpen,
  FolderPlus,
  RefreshCw,
  Search,
  Download,
  LoaderCircle,
  CheckCircle2,
  Package,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./components/ui/dialog";
import { Button } from "./components/ui/button";
import { Switch } from "./components/ui/switch";
import { api } from "./bridge";
import FeedbackFlyout from "./FeedbackFlyout";
import { notify, notifySaved } from "./StudioFlyouts";
import type {
  LocalWorld,
  LocalWorldCatalog,
  LocalWorldPlan,
} from "./local-worlds-model";
import "./local-worlds.css";
const size = (value: number) =>
  value >= 1073741824
    ? `${(value / 1073741824).toFixed(2)} GB`
    : value >= 1048576
      ? `${(value / 1048576).toFixed(1)} MB`
      : `${Math.ceil(value / 1024)} KB`;
const statusNames = {
  included: "Included",
  missing: "Not found",
  ambiguous: "Multiple copies",
  protected: "Protected",
};
export default function LocalWorldsDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (value: boolean) => void;
}) {
  const [catalog, setCatalog] = useState<LocalWorldCatalog>({ worlds: [] }),
    [loading, setLoading] = useState(true),
    [checking, setChecking] = useState(false),
    [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState(""),
    [query, setQuery] = useState(""),
    [profile, setProfile] = useState("all"),
    [includeGlobal, setIncludeGlobal] = useState(false),
    [backupOnly, setBackupOnly] = useState(false);
  const [plan, setPlan] = useState<LocalWorldPlan>(),
    [error, setError] = useState(""),
    [revision, setRevision] = useState(0);
  const sequence = useRef(0),
    mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const accept = (value: LocalWorldCatalog) => {
    setCatalog(value);
    setSelected((old) =>
      value.worlds.some((world) => world.id === old) ? old : "",
    );
    if (value.warning)
      notify({
        id: "local-world-discovery",
        tone: "warning",
        title: "Minecraft storage",
        description: value.warning,
      });
  };
  const load = async () => {
    setLoading(true);
    setError("");
    try {
      const value = await api.GetLocalWorlds();
      if (mounted.current) {
        accept(value);
        setRevision((value) => value + 1);
      }
    } catch (e) {
      if (mounted.current) setError(String(e));
    } finally {
      if (mounted.current) setLoading(false);
    }
  };
  useEffect(() => {
    if (open) void load();
  }, [open]);
  useEffect(() => {
    const run = ++sequence.current;
    setPlan(undefined);
    setError("");
    if (!selected) {
      setChecking(false);
      return;
    }
    setChecking(true);
    void api
      .InspectLocalWorld({
        worldId: selected,
        options: { includeGlobal, backupOnly: false },
      })
      .then((value) => {
        if (!mounted.current || run !== sequence.current) return;
        setPlan(value);
        if (!value.portable)
          notify({
            id: "local-world-dependencies",
            tone: "warning",
            title: "Some content cannot be shared as a complete world",
            description:
              "Review the pack report. Missing packs can be located with Find packs; protected content can only be preserved in an unchanged backup.",
          });
        else if (value.world.template || value.world.marketplace)
          notify({
            id: "local-world-template",
            tone: "info",
            title: "World template content",
            description:
              "Original template settings are preserved. The recipient may need to own its Marketplace content.",
          });
      })
      .catch((e) => {
        if (mounted.current && run === sequence.current) setError(String(e));
      })
      .finally(() => {
        if (mounted.current && run === sequence.current) setChecking(false);
      });
    return () => {
      sequence.current++;
    };
  }, [selected, includeGlobal, revision]);
  const run = async (work: () => Promise<unknown>) => {
    setError("");
    try {
      await work();
    } catch (e) {
      setError(String(e));
    }
  };
  const pick = () =>
    run(async () => {
      setLoading(true);
      try {
        accept(await api.PickLocalWorldFolder());
      } finally {
        setLoading(false);
      }
    });
  const exportWorld = async () => {
    if (!plan || checking || busy || (!plan.portable && !backupOnly)) return;
    setBusy(true);
    setError("");
    try {
      const result = await api.ExportLocalWorld({
        worldId: selected,
        fingerprint: plan.fingerprint,
        options: { includeGlobal, backupOnly },
      });
      if (result?.path) {
        notifySaved(result.path);
        if (!result.portable)
          notify({
            tone: "warning",
            title: "Backup saved",
            description:
              "This copy preserves the available files and protection metadata. It is not a complete, unlocked world for another account.",
          });
      }
    } catch (e) {
      if (!/cancelled|canceled|context cancel/i.test(String(e)))
        setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const shown = catalog.worlds.filter(
    (world) =>
      (profile === "all" || world.installation === profile) &&
      `${world.name} ${world.installation}`
        .toLocaleLowerCase()
        .includes(query.toLocaleLowerCase()),
  );
  const world = catalog.worlds.find((value) => value.id === selected);
  const choose = (value: LocalWorld) => {
    setBackupOnly(false);
    setSelected(value.id);
  };
  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!busy) onOpenChange(value);
      }}
    >
      <DialogContent className="local-world-dialog" showCloseButton={!busy}>
        <DialogHeader>
          <DialogTitle>Copy singleplayer world</DialogTitle>
          <DialogDescription>
            Export the original world, its add-ons and texture packs as a
            .mcworld file.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <FeedbackFlyout title="Singleplayer world copy">
            {error}
          </FeedbackFlyout>
        )}
        <div className="local-world-layout">
          <aside
            className="studio-sidebar local-world-browser"
            aria-label="Local singleplayer worlds"
          >
            <div className="local-world-library-heading">
              <strong>Local worlds</strong>
              <span>{catalog.worlds.length}</span>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Refresh local worlds"
                title="Refresh"
                disabled={busy || loading}
                onClick={() => void load()}
              >
                <RefreshCw size={14} />
              </Button>
            </div>
            <label className="local-world-search">
              <Search size={14} />
              <input
                type="search"
                aria-label="Search local worlds"
                placeholder="Search world or profile"
                value={query}
                disabled={busy}
                onChange={(e) => setQuery(e.target.value)}
              />
            </label>
            <select
              aria-label="Minecraft profile"
              value={profile}
              disabled={busy}
              onChange={(e) => setProfile(e.target.value)}
            >
              <option value="all">All Minecraft profiles</option>
              {[
                ...new Set(catalog.worlds.map((value) => value.installation)),
              ].map((name) => (
                <option key={name} value={name}>
                  {name}
                </option>
              ))}
            </select>
            <div className="local-world-list">
              {loading ? (
                <div className="local-world-loading" role="status">
                  <LoaderCircle className="spin" size={20} />
                  Finding installed worlds...
                </div>
              ) : (
                shown.map((value) => (
                  <button
                    key={value.id}
                    aria-label={value.name}
                    aria-pressed={selected === value.id}
                    className="local-world-row"
                    title={`${value.name}\n${value.folder}`}
                    disabled={busy}
                    onClick={() => choose(value)}
                  >
                    {value.icon ? (
                      <img src={value.icon} alt="" />
                    ) : (
                      <Box size={28} />
                    )}
                    <span>
                      <strong>{value.name}</strong>
                      <small>{value.installation}</small>
                      <small>
                        {value.template || value.marketplace
                          ? "Template world"
                          : new Date(value.modified).toLocaleDateString()}
                      </small>
                    </span>
                  </button>
                ))
              )}
              {!loading && !shown.length && (
                <div className="local-world-loading">
                  <Box size={28} />
                  <strong>
                    {catalog.worlds.length
                      ? "No matching worlds"
                      : "No local worlds found"}
                  </strong>
                  <p>
                    Choose a saved world folder, minecraftWorlds or com.mojang.
                  </p>
                </div>
              )}
            </div>
            <footer>
              <Button
                variant="ghost"
                disabled={busy || loading}
                onClick={() => void pick()}
              >
                <FolderPlus size={14} />
                Choose world folder...
              </Button>
            </footer>
          </aside>
          <main className="local-world-details" aria-label="World copy details">
            {!world ? (
              <div className="local-world-empty">
                <Box size={36} />
                <strong>Select a singleplayer world</strong>
                <p>
                  World data, player inventories, settings and available packs
                  are copied together. The installed world stays unchanged.
                </p>
              </div>
            ) : (
              <>
                <div className="local-world-title">
                  <span>
                    <strong>{world.name}</strong>
                    <small>{world.installation}</small>
                  </span>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={busy}
                    title="Open original world folder"
                    aria-label="Open original world folder"
                    onClick={() =>
                      void run(() => api.OpenLocalWorldFolder(world.id))
                    }
                  >
                    <FolderOpen size={15} />
                  </Button>
                </div>
                <div className="local-world-options">
                  <label>
                    <span>
                      <strong>Include global texture packs</strong>
                      <small>
                        Also activate the resource packs enabled globally in
                        this Minecraft profile.
                      </small>
                    </span>
                    <Switch
                      aria-label="Include global texture packs"
                      checked={includeGlobal}
                      disabled={busy}
                      onCheckedChange={setIncludeGlobal}
                    />
                  </label>
                  <label>
                    <span>
                      <strong>Backup only</strong>
                      <small>
                        Preserve available files even when packs are missing or
                        protected. Protection and account requirements remain.
                      </small>
                    </span>
                    <Switch
                      aria-label="Backup only"
                      checked={backupOnly}
                      disabled={busy}
                      onCheckedChange={setBackupOnly}
                    />
                  </label>
                </div>
                {checking ? (
                  <div className="local-world-loading" role="status">
                    <LoaderCircle className="spin" size={22} />
                    <strong>Checking world and pack dependencies...</strong>
                    <p>Matching pack IDs, versions and add-on dependencies.</p>
                  </div>
                ) : plan ? (
                  <>
                    <div className="local-world-summary">
                      <CheckCircle2 size={17} />
                      <strong>
                        {plan.portable
                          ? "All required files found"
                          : "Review pack dependencies"}
                      </strong>
                      <span>
                        {size(plan.bytes)} · {plan.files.toLocaleString()} files
                      </span>
                    </div>
                    <div className="local-world-pack-heading">
                      <strong>Packs and add-ons ({plan.packs.length})</strong>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        onClick={() =>
                          void run(async () => {
                            await api.PickLocalWorldPackFolder(selected);
                            setRevision((value) => value + 1);
                          })
                        }
                      >
                        <FolderPlus size={14} />
                        Find packs...
                      </Button>
                    </div>
                    <div className="local-world-packs">
                      {plan.packs.map((pack, index) => (
                        <div
                          key={`${pack.id}:${pack.version}:${index}`}
                          className="local-world-pack"
                        >
                          <Package size={18} />
                          <span>
                            <strong>{pack.name}</strong>
                            <small>
                              {pack.kind === "behavior"
                                ? "Behavior pack"
                                : "Resource pack"}{" "}
                              · {pack.version} ·{" "}
                              {pack.embedded ? "In world" : "Dependency"}
                            </small>
                            <small>Required by: {pack.requiredBy}</small>
                          </span>
                          <span
                            className="local-world-pack-status"
                            data-status={pack.status}
                          >
                            {statusNames[pack.status]}
                          </span>
                        </div>
                      ))}
                      {!plan.packs.length && (
                        <p>This world does not require additional packs.</p>
                      )}
                    </div>
                    {!!plan.issues.length && (
                      <details className="local-world-report">
                        <summary>
                          Dependency report ({plan.issues.length})
                        </summary>
                        <ul>
                          {plan.issues.map((issue, index) => (
                            <li key={index}>{issue.message}</li>
                          ))}
                        </ul>
                      </details>
                    )}
                  </>
                ) : (
                  <div className="local-world-loading">
                    <p>The world could not be inspected.</p>
                    <Button
                      variant="outline"
                      onClick={() => setRevision((value) => value + 1)}
                    >
                      Retry inspection
                    </Button>
                  </div>
                )}
              </>
            )}
          </main>
        </div>
        <DialogFooter>
          <span className="local-world-footer-note">
            Close the selected world in Minecraft before exporting. Open the
            .mcworld file to import it on the receiving device.
          </span>
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => onOpenChange(false)}
          >
            Close
          </Button>
          <Button
            disabled={
              busy ||
              checking ||
              loading ||
              !plan ||
              (!plan.portable && !backupOnly)
            }
            onClick={() => void exportWorld()}
          >
            <Download size={14} />
            {busy
              ? "Exporting..."
              : backupOnly
                ? "Export backup"
                : "Export .mcworld"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
