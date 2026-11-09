import FeedbackFlyout from "./FeedbackFlyout";
import StudioForm from "@/StudioForm";
import { useEffect, useRef, useState } from "react";
import { sourceAddressError } from "./input-rules";
import {
  Search,
  RefreshCw,
  LoaderCircle,
  Server,
  CheckCircle2,
  AlertCircle,
  Star,
  Trash2,
  FolderOpen,
  LockKeyhole,
} from "lucide-react";
import { SourceKindIcon } from "./studio-icons";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
  type Source,
  type SourceProbe,
  type MinecraftSourceProfile,
} from "./bridge";
import "./source-picker.css";

export const sourceNames: Record<string, string> = {
  custom: "Custom address",
  minecraft: "Saved Servers",
  featured: "Featured servers",
  realms: "Realms",
  events: "Experiences",
  worlds: "Friends' worlds",
};
const message = (e: unknown) => (e instanceof Error ? e.message : String(e));
const selectableSource = (source: Source, kind: string) => {
  if (!source.available) return false;
  if (source.onDemand || kind === "worlds") return true;
  if (!source.probe) return kind !== "minecraft";
  return source.probe.reachable || source.probe.discovery === "session";
};
const count = (s: Source) => {
  if (s.onDemand && s.probe?.players === undefined && s.players !== undefined)
    return String(s.players);
  const players = s.probe?.players ?? s.players,
    capacity = s.probe?.maxPlayers ?? s.maxPlayers;
  return capacity === undefined && players !== undefined
    ? String(players)
    : `${players ?? "?"} / ${capacity ?? "?"}`;
};
const status = (p?: SourceProbe) =>
  !p
    ? "Not checked"
    : p.discovery === "session"
      ? "Session found"
      : p.reachable
        ? "NetherNet reachable"
        : "Status unavailable";
function loadSaved(): string[] {
  try {
    const v: unknown = JSON.parse(
      localStorage.getItem("CraftBox-custom-sources") ?? "[]",
    );
    return Array.isArray(v)
      ? v
          .filter((s): s is string => typeof s === "string" && s.length <= 512)
          .slice(0, 12)
      : [];
  } catch {
    return [];
  }
}

export default function SourcePicker({
  open,
  onOpenChange,
  initialKind,
  loggedIn,
  minecraft,
  accountKey,
  onSignIn,
  onSelect,
  onOpenCapture,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  initialKind: string;
  loggedIn: boolean;
  minecraft?: string;
  accountKey?: string;
  onSignIn: () => void;
  onSelect: (s: Source) => void;
  onOpenCapture?: () => void;
}) {
  const [kind, setKind] = useState(initialKind),
    [sources, setSources] = useState<Source[]>([]),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [search, setSearch] = useState(""),
    [chosen, setChosen] = useState<string>(),
    [custom, setCustom] = useState(""),
    [customSource, setCustomSource] = useState<Source>(),
    [checking, setChecking] = useState<string[]>([]),
    [saved, setSaved] = useState(loadSaved),
    [revision, setRevision] = useState(0),
    [profiles, setProfiles] = useState<MinecraftSourceProfile[]>([]),
    [profile, setProfile] = useState("");
  const epoch = useRef(0),
    customEpoch = useRef(0);
  const inspections = useRef(new Set<string>());
  useEffect(() => {
    if (open) setKind(initialKind);
  }, [open, initialKind]);
  useEffect(() => {
    const sequence = ++epoch.current;
    customEpoch.current++;
    setSources([]);
    setChosen(undefined);
    setChecking([]);
    setError("");
    setSearch("");
    setCustomSource(undefined);
    setProfiles([]);
    if (!open || kind === "custom" || (!loggedIn && kind !== "minecraft")) {
      setLoading(false);
      return;
    }
    setLoading(true);
    const request =
      kind === "minecraft"
        ? api.GetMinecraftServerSources().then((catalog) => {
            if (sequence === epoch.current) {
              setProfiles(catalog.profiles);
              setProfile((old) =>
                catalog.profiles.some((p) => p.id === old) ? old : "",
              );
            }
            return catalog.sources;
          })
        : api.ListSources(kind);
    void request
      .then(async (result) => {
        if (sequence !== epoch.current) return;
        setSources(result);
        setLoading(false);

        if (kind === "featured" || kind === "minecraft") {
          const publicSources = result.filter((s) => !s.onDemand);
          let index = 0;
          await Promise.all(
            Array.from(
              { length: Math.min(3, publicSources.length) },
              async () => {
                while (
                  index < publicSources.length &&
                  sequence === epoch.current
                ) {
                  const source = publicSources[index++];
                  await inspect(source, false, sequence);
                }
              },
            ),
          );
        }
      })
      .catch((e) => {
        if (sequence === epoch.current) {
          setError(message(e));
          setLoading(false);
        }
      });
    return () => {
      epoch.current++;
      customEpoch.current++;
    };
  }, [open, kind, loggedIn, accountKey, revision]);
  async function inspect(
    source: Source,
    connection: boolean,
    sequence = epoch.current,
  ) {
    const key = `${sequence}:${source.id}`;
    if (inspections.current.has(key)) return;
    inspections.current.add(key);
    setChecking((v) => [...v, source.id]);
    try {
      const probe = await api.InspectSource(source.id, connection);
      if (sequence === epoch.current)
        setSources((v) =>
          v.map((s) => (s.id === source.id ? { ...s, probe } : s)),
        );
    } catch (e) {
      if (sequence === epoch.current)
        setSources((v) =>
          v.map((s) =>
            s.id === source.id
              ? {
                  ...s,
                  probe: {
                    endpoint: "",
                    reachable: false,
                    name: "",
                    latencyMs: 0,
                    transport: "not-tested",
                    checkedAt: new Date().toISOString(),
                    error: message(e),
                  },
                }
              : s,
          ),
        );
    } finally {
      inspections.current.delete(key);
      if (sequence === epoch.current)
        setChecking((v) => v.filter((id) => id !== source.id));
    }
  }
  async function checkCustom(connection = false, addressOverride?: string) {
    if (addressOverride === undefined && checking.includes("custom")) return;
    const sequence = ++customEpoch.current,
      address = (addressOverride ?? custom).trim();
    setChecking((v) => [...v, "custom"]);
    setError("");
    try {
      const probe = await api.InspectSource(address, connection);
      if (sequence === customEpoch.current)
        setCustomSource({
          id: probe.endpoint || address,
          name: probe.name || address,
          address: probe.endpoint || address,
          description: "Custom server",
          available: probe.reachable,
          route: "NetherNet · HTTP signaling",
          probe,
        });
    } catch (e) {
      if (sequence === customEpoch.current) setError(message(e));
    } finally {
      if (sequence === customEpoch.current)
        setChecking((v) => v.filter((id) => id !== "custom"));
    }
  }
  function editCustom(value: string) {
    customEpoch.current++;
    setCustom(value);
    setCustomSource(undefined);
    setError("");
    setChecking((v) => v.filter((id) => id !== "custom"));
  }
  function selectSaved(address: string) {
    if (checking.includes("custom") && custom.trim() === address) return;
    editCustom(address);
    void checkCustom(false, address);
  }
  function saveAddress(address: string, remove = false) {
    const next = remove
      ? saved.filter((v) => v !== address)
      : [address, ...saved.filter((v) => v !== address)].slice(0, 12);
    setSaved(next);
    try {
      localStorage.setItem("CraftBox-custom-sources", JSON.stringify(next));
    } catch {
      /* Preferences may be unavailable. */
    }
  }
  const selected =
      kind === "custom" ? customSource : sources.find((s) => s.id === chosen),
    p = selected?.probe,
    busy = checking.includes(
      kind === "custom" ? "custom" : (selected?.id ?? ""),
    ),
    friend = kind === "worlds";
  const visible = sources
    .filter(
      (s) =>
        (kind !== "minecraft" || !profile || s.profileId === profile) &&
        `${s.name} ${s.description} ${s.address} ${s.host ?? ""}`
          .toLowerCase()
          .includes(search.toLowerCase()),
    )
    .sort(
      (a, b) =>
        Number(selectableSource(b, kind)) - Number(selectableSource(a, kind)) ||
        (b.probe?.players ?? b.players ?? -1) -
          (a.probe?.players ?? a.players ?? -1) ||
        a.name.localeCompare(b.name),
    );
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="source-browser">
        <DialogHeader>
          <DialogTitle>Select source</DialogTitle>
          <DialogDescription>Servers and worlds</DialogDescription>
        </DialogHeader>
        <div
          className="source-browser-tabs"
          role="tablist"
          aria-label="Source type"
        >
          {Object.entries(sourceNames).map(([key, name]) => (
            <Button
              key={key}
              role="tab"
              aria-selected={kind === key}
              data-tooltip={
                !loggedIn && key !== "custom" && key !== "minecraft"
                  ? "Microsoft sign-in required"
                  : undefined
              }
              variant={kind === key ? "default" : "ghost"}
              onClick={() => setKind(key)}
            >
              <SourceKindIcon kind={key} size={14} />
              {name}
              {!loggedIn && key !== "custom" && key !== "minecraft" && (
                <LockKeyhole size={12} aria-hidden="true" />
              )}
            </Button>
          ))}
        </div>
        <div className="source-browser-body">
          <section
            className="source-browser-list"
            aria-label="Available sources"
          >
            {kind === "custom" ? (
              <>
                <StudioForm
                  className="custom-source-form"
                  onSubmit={(e) => {
                    e.preventDefault();
                    void checkCustom();
                  }}
                >
                  <label htmlFor="custom-source-address">
                    Custom server address
                  </label>
                  <Input
                    id="custom-source-address"
                    required
                    validate={sourceAddressError}
                    autoComplete="off"
                    maxLength={512}
                    placeholder="server.example:19132 or https://host:port"
                    value={custom}
                    onChange={(e) => editCustom(e.target.value)}
                  />
                  <p>
                    Host or [IPv6]:port. Default: 19132. HTTPS stays encrypted.
                  </p>
                  <Button type="submit" disabled={!custom.trim() || busy}>
                    {busy ? <LoaderCircle className="spin" /> : <Search />}Check
                    server
                  </Button>
                </StudioForm>
                <div className="source-browser-saved">
                  <h3>Saved addresses</h3>
                  {saved.length ? (
                    saved.map((address) => (
                      <div key={address}>
                        <button
                          aria-pressed={custom.trim() === address}
                          onClick={() => selectSaved(address)}
                        >
                          <Server size={15} />
                          {address}
                        </button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Remove ${address}`}
                          onClick={() => saveAddress(address, true)}
                        >
                          <Trash2 size={14} />
                        </Button>
                      </div>
                    ))
                  ) : (
                    <p>No saved addresses.</p>
                  )}
                </div>
                {onOpenCapture && (
                  <Button
                    className="source-browser-replay"
                    variant="outline"
                    onClick={onOpenCapture}
                  >
                    <FolderOpen />
                    Open captured session (.pcap2)
                  </Button>
                )}
              </>
            ) : (
              <>
                <div
                  className={`source-browser-toolbar ${kind === "minecraft" ? "has-profile" : ""}`}
                >
                  {kind === "minecraft" && (
                    <div className="source-browser-profile">
                      <label
                        className="sr-only"
                        htmlFor="minecraft-source-profile"
                      >
                        Minecraft profile
                      </label>
                      <select
                        id="minecraft-source-profile"
                        value={profile}
                        disabled={loading}
                        onChange={(e) => {
                          setProfile(e.target.value);
                          setChosen(undefined);
                        }}
                      >
                        <option value="">All profiles</option>
                        {profiles.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name} · {p.count} servers
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                  <div className="source-browser-search">
                    <Search size={15} />
                    <Input
                      aria-label="Browse sources"
                      placeholder="Search servers"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                    />
                  </div>
                  <Button
                    className="source-browser-refresh"
                    variant="ghost"
                    size="icon-sm"
                    aria-label="Refresh sources"
                    disabled={loading}
                    onClick={() => setRevision((v) => v + 1)}
                  >
                    <RefreshCw className={loading ? "spin" : ""} />
                  </Button>
                </div>
                {kind === "minecraft" &&
                  profiles
                    .filter((p) => (!profile || p.id === profile) && p.warning)
                    .map((p) => (
                      <FeedbackFlyout key={p.id} title={p.name} tone="warning">
                        {p.warning}
                      </FeedbackFlyout>
                    ))}
                <div className="source-browser-rows" aria-busy={loading}>
                  {!loggedIn && kind !== "minecraft" ? (
                    <div className="source-browser-empty">
                      <LockKeyhole />
                      <p>Microsoft sign-in required</p>
                      <Button onClick={onSignIn}>Sign in</Button>
                    </div>
                  ) : loading ? (
                    <div className="source-browser-empty">
                      <LoaderCircle className="spin" />
                      <p>Loading sources …</p>
                      <span>Timeout: 25 s.</span>
                    </div>
                  ) : error ? (
                    <div className="source-browser-empty">
                      <AlertCircle />
                      <FeedbackFlyout title="Connection source">
                        {error}
                      </FeedbackFlyout>
                      {!loggedIn ? (
                        <Button onClick={onSignIn}>Sign in</Button>
                      ) : (
                        <Button
                          variant="outline"
                          onClick={() => setRevision((v) => v + 1)}
                        >
                          Try again
                        </Button>
                      )}
                    </div>
                  ) : visible.length ? (
                    visible.map((s) => (
                      <button
                        className="source-browser-row"
                        key={s.id}
                        disabled={!selectableSource(s, kind)}
                        aria-pressed={chosen === s.id}
                        onClick={() => {
                          setChosen(s.id);
                          if (
                            !s.probe &&
                            kind !== "featured" &&
                            !friend &&
                            !s.onDemand
                          )
                            void inspect(s, false);
                        }}
                      >
                        <SourceKindIcon kind={kind} size={20} />
                        <div>
                          <strong>{s.name}</strong>
                          <span>{s.description || s.address}</span>
                          <small>
                            {kind === "minecraft" && !s.probe
                              ? checking.includes(s.id)
                                ? "Checking status…"
                                : "Waiting for status"
                              : s.available
                                ? !s.probe && s.onDemand
                                  ? "Session assigned on demand"
                                  : status(s.probe)
                                : "Unavailable"}
                            {checking.includes(s.id) ? " · Checking…" : ""}
                          </small>
                        </div>
                        <div className="source-browser-player-count">
                          <span>
                            {s.onDemand &&
                            s.probe?.players === undefined &&
                            s.players === undefined
                              ? s.playerCountError
                                ? "Count unavailable"
                                : "No public count"
                              : count(s)}
                          </span>
                          <small>
                            {s.onDemand
                              ? s.probe?.players === undefined
                                ? "experience players"
                                : "session players"
                              : "players"}
                          </small>
                        </div>
                      </button>
                    ))
                  ) : (
                    <div className="source-browser-empty">
                      <SourceKindIcon kind={kind} />
                      <p>
                        {search
                          ? "No matching sources."
                          : kind === "minecraft"
                            ? "No saved Minecraft servers found in this profile."
                            : "No sources available."}
                      </p>
                    </div>
                  )}
                </div>
              </>
            )}
          </section>
          <aside className="source-browser-details" aria-label="Source details">
            {selected ? (
              <>
                <div className="source-browser-detail-title">
                  <SourceKindIcon kind={kind} />
                  <div>
                    <h3>{selected.name}</h3>
                    <p>{selected.route ?? "NetherNet"}</p>
                  </div>
                </div>
                <div
                  className={`source-browser-status ${p?.reachable ? "is-reachable" : ""}`}
                >
                  {p?.reachable ? (
                    <CheckCircle2 size={16} />
                  ) : (
                    <AlertCircle size={16} />
                  )}
                  <strong>
                    {friend
                      ? "Xbox session listed"
                      : !p && selected.onDemand
                        ? "Session assigned on demand"
                        : status(p)}
                  </strong>
                </div>
                <dl>
                  {selected.profileName && (
                    <>
                      <dt>Minecraft profile</dt>
                      <dd>{selected.profileName}</dd>
                    </>
                  )}
                  {p?.name && p.name !== selected.name && (
                    <>
                      <dt>Server message</dt>
                      <dd>{p.name}</dd>
                    </>
                  )}
                  <dt>
                    {selected.onDemand
                      ? p?.players === undefined
                        ? "Experience players"
                        : "Session players"
                      : "Players"}
                  </dt>
                  <dd>
                    {selected.onDemand &&
                    p?.players === undefined &&
                    selected.players === undefined
                      ? selected.playerCountError
                        ? selected.playerCountError
                        : "Not published"
                      : count(selected)}
                  </dd>
                  {selected.onDemand &&
                    selected.players !== undefined &&
                    p?.players === undefined && (
                      <>
                        <dt>Count scope</dt>
                        <dd>Across sessions</dd>
                      </>
                    )}
                  {selected.onDemand &&
                    selected.players !== undefined &&
                    p?.players !== undefined && (
                      <>
                        <dt>Experience players</dt>
                        <dd>{selected.players}</dd>
                      </>
                    )}
                  <dt>Game version</dt>
                  <dd>{p?.version || selected.version || "Not provided"}</dd>
                  <dt>Protocol</dt>
                  <dd>{p?.protocol ?? selected.protocol ?? "Not provided"}</dd>
                  {p?.compatible !== undefined && (
                    <>
                      <dt>Compatibility</dt>
                      <dd
                        className={p.compatible ? "" : "source-browser-warning"}
                      >
                        {p.compatible
                          ? "Matches CraftBox"
                          : "Different protocol"}
                      </dd>
                    </>
                  )}
                  {p?.discovery === "session" && (
                    <>
                      <dt>Connection path</dt>
                      <dd>
                        {p.networkProtocol?.toLowerCase().includes("jsonrpc")
                          ? "NetherNet · JSON-RPC messaging"
                          : "NetherNet · WebSocket signaling"}
                      </dd>
                    </>
                  )}
                  <dt>Response time</dt>
                  <dd>
                    {p?.reachable && p.discovery !== "session"
                      ? `${p.latencyMs} ms`
                      : "Not measured"}
                  </dd>
                  <dt>World</dt>
                  <dd>{p?.level || "Not provided"}</dd>
                  <dt>Game mode</dt>
                  <dd>
                    {p?.gameType === undefined
                      ? "Not provided"
                      : (["Survival", "Creative", "Adventure"][p.gameType] ??
                        `Mode ${p.gameType}`)}
                  </dd>
                  {p?.onlineAuth !== undefined && (
                    <>
                      <dt>Microsoft authentication</dt>
                      <dd>{p.onlineAuth ? "Supported" : "Not advertised"}</dd>
                    </>
                  )}
                  {p?.selfSignedAuth !== undefined && (
                    <>
                      <dt>Self-signed authentication</dt>
                      <dd>
                        {p.selfSignedAuth ? "Supported" : "Not supported"}
                      </dd>
                    </>
                  )}
                  {p?.editor !== undefined && (
                    <>
                      <dt>Editor world</dt>
                      <dd>{p.editor ? "Yes" : "No"}</dd>
                    </>
                  )}
                  {p?.hardcore !== undefined && (
                    <>
                      <dt>Hardcore</dt>
                      <dd>{p.hardcore ? "Yes" : "No"}</dd>
                    </>
                  )}
                  {p?.transportLayer !== undefined && (
                    <>
                      <dt>Transport layer</dt>
                      <dd>
                        {p.transportLayer === 2
                          ? "NetherNet (2)"
                          : p.transportLayer}
                      </dd>
                    </>
                  )}
                  {p?.connection !== undefined && (
                    <>
                      <dt>Connection type</dt>
                      <dd>
                        {p.connection === 4
                          ? "HTTP signaling (4)"
                          : p.connection}
                      </dd>
                    </>
                  )}
                  {p?.dataVersion !== undefined && (
                    <>
                      <dt>Discovery format</dt>
                      <dd>{p.dataVersion}</dd>
                    </>
                  )}
                  {selected.host && (
                    <>
                      <dt>Host</dt>
                      <dd>{selected.host}</dd>
                    </>
                  )}
                  {selected.joinability && (
                    <>
                      <dt>Joinability</dt>
                      <dd>{selected.joinability}</dd>
                    </>
                  )}
                  {selected.hardcore && (
                    <>
                      <dt>Hardcore</dt>
                      <dd>Yes</dd>
                    </>
                  )}
                  {selected.editorWorld && (
                    <>
                      <dt>Editor world</dt>
                      <dd>Yes</dd>
                    </>
                  )}
                  {selected.crossPlayDisabled && (
                    <>
                      <dt>Cross-play</dt>
                      <dd>Disabled</dd>
                    </>
                  )}
                  {selected.startsAt && (
                    <>
                      <dt>Event starts</dt>
                      <dd>{new Date(selected.startsAt).toLocaleString()}</dd>
                      <dt>Event ends</dt>
                      <dd>{new Date(selected.endsAt!).toLocaleString()}</dd>
                    </>
                  )}
                </dl>
                <label className="source-browser-endpoint-label">
                  {selected.onDemand
                    ? "Experience connection"
                    : p?.endpoint
                      ? "Checked endpoint"
                      : "Source address"}
                  <span className="source-browser-endpoint">
                    {p?.discovery === "session"
                      ? "Authenticated NetherNet session"
                      : selected.onDemand && !p?.endpoint
                        ? "Assigned on demand"
                        : p?.endpoint || selected.address}
                  </span>
                </label>
                {selected.onDemand && (
                  <p className="source-browser-footnote">On-demand session.</p>
                )}
                {p?.error && (
                  <FeedbackFlyout title="Connection source" tone="warning">
                    {p.error}
                  </FeedbackFlyout>
                )}
                <div className="source-browser-connection">
                  <strong>
                    {p?.transport === "connected"
                      ? "WebRTC connected"
                      : p?.transport === "failed"
                        ? "Connection failed"
                        : "Connection not tested"}
                  </strong>
                  <p>
                    {p?.transport === "connected"
                      ? "Connection test passed. Minecraft sign-in is checked when capture starts."
                      : p?.discovery === "session"
                        ? "Session assigned."
                        : friend
                          ? "The Xbox connection is checked when capture starts."
                          : "Status checked; connection not tested."}
                  </p>
                  {p?.transportError && (
                    <FeedbackFlyout title="Connection source" tone="warning">
                      {p.transportError}
                    </FeedbackFlyout>
                  )}
                  {p?.transportError?.includes("401") && (
                    <FeedbackFlyout title="Connection source" tone="warning">
                      {loggedIn
                        ? "Account rejected. Check access or sign in again."
                        : "Microsoft sign-in required."}
                    </FeedbackFlyout>
                  )}
                  {p?.onlineAuth && !loggedIn && (
                    <Button variant="outline" onClick={onSignIn}>
                      Sign in
                    </Button>
                  )}
                  {!friend && (
                    <div>
                      <Button
                        variant="outline"
                        disabled={busy || !selected.available}
                        onClick={() =>
                          kind === "custom"
                            ? void checkCustom()
                            : void inspect(selected, false)
                        }
                      >
                        <RefreshCw />
                        {selected.onDemand
                          ? "Find a session"
                          : "Refresh status"}
                      </Button>
                      <Button
                        variant="outline"
                        disabled={
                          busy || (!p?.reachable && p?.discovery !== "session")
                        }
                        onClick={() =>
                          kind === "custom"
                            ? void checkCustom(true)
                            : void inspect(selected, true)
                        }
                      >
                        {busy ? (
                          <LoaderCircle className="spin" />
                        ) : (
                          <CheckCircle2 />
                        )}
                        Test connection
                      </Button>
                    </div>
                  )}
                </div>
                {!loggedIn && (
                  <p className="source-browser-footnote">
                    Connection tests may require sign-in.
                  </p>
                )}
                {p?.checkedAt && (
                  <p className="source-browser-footnote">
                    Checked {new Date(p.checkedAt).toLocaleTimeString()} ·
                    CraftBox MC {minecraft ?? "current"}
                  </p>
                )}
                {((kind === "custom" && p?.reachable) ||
                  kind === "minecraft") && (
                  <Button
                    variant="ghost"
                    onClick={() => saveAddress(selected.address)}
                  >
                    <Star size={15} />
                    {saved.includes(selected.address)
                      ? "Address saved"
                      : kind === "minecraft"
                        ? "Save address in CraftBox"
                        : "Save address"}
                  </Button>
                )}
              </>
            ) : (
              <div className="source-browser-empty">
                <Server />
                <h3>
                  {kind === "custom" ? "Check an address" : "Select a source"}
                </h3>
                <p>Players · Version · Connection</p>
                <p>? = not provided.</p>
              </div>
            )}
            {kind === "custom" && error && (
              <FeedbackFlyout title="Connection source" tone="warning">
                {error}
              </FeedbackFlyout>
            )}
          </aside>
        </div>
        <DialogFooter>
          <span className="dialog-footer-note">
            {sourceNames[kind]}
            {kind !== "custom"
              ? ` · ${visible.length} / ${sources.length} sources`
              : " · NetherNet"}
          </span>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={
              !selected ||
              !selectableSource(selected, kind) ||
              busy ||
              (kind === "custom" && !p?.reachable)
            }
            onClick={() => {
              if (selected && selectableSource(selected, kind)) {
                onSelect(selected);
                onOpenChange(false);
              }
            }}
          >
            Use source
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
