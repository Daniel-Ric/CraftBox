import { useEffect, useState } from "react";
import { api, type DiscordPresenceStatus as PresenceStatus } from "./bridge";
import { useWorkspacePreferences } from "./workspace-preferences";
import { ImageIcon, RefreshCw } from "lucide-react";
import "./discord-presence.css";

export default function DiscordPresenceStatus() {
  const [status, setStatus] = useState<PresenceStatus>({
    connected: false,
    message: "Checking Discord connection…",
    activity: "In the workspace",
    startedAt: 0,
  });
  const [retrying, setRetrying] = useState(false);
  const [error, setError] = useState("");
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  const p = useWorkspacePreferences();
  useEffect(() => {
    let active = true;
    const refresh = () =>
      void api
        .GetDiscordPresenceStatus()
        .then((s) => {
          if (active) {
            setStatus(s);
            setError("");
          }
        })
        .catch((e) => {
          if (active) setError(String(e));
        });
    refresh();
    const poll = window.setInterval(refresh, 2000);
    const clock = window.setInterval(
      () => setNow(Math.floor(Date.now() / 1000)),
      1000,
    );
    return () => {
      active = false;
      window.clearInterval(poll);
      window.clearInterval(clock);
    };
  }, []);
  const retry = async () => {
    setRetrying(true);
    setError("");
    try {
      await api.RetryDiscordPresence();
      setStatus(await api.GetDiscordPresenceStatus());
    } catch (e) {
      setError(String(e));
    } finally {
      setRetrying(false);
    }
  };
  const connected = p.discordEnabled && status.connected && !error;
  const elapsed =
    status.startedAt > 0 ? Math.max(0, now - status.startedAt) : 0;
  const timer = [
    ...(elapsed >= 3600 ? [Math.floor(elapsed / 3600)] : []),
    Math.floor(elapsed / 60) % 60,
    elapsed % 60,
  ]
    .map((n) => String(n).padStart(2, "0"))
    .join(":");
  const badge = /^https?:\/\//i.test(p.discordSmallImage.trim())
    ? p.discordSmallImage.trim()
    : "";
  const message = !p.discordEnabled
    ? "Turn on sharing when you're ready. Your preferences are kept."
    : error ||
      (status.message === "Disabled"
        ? "Open Discord desktop to share your activity."
        : status.message);
  return (
    <>
      <div className="discord-connection" data-connected={connected}>
        <span className="discord-connection-dot" />
        <div>
          <strong role="status">
            {!p.discordEnabled
              ? "Not sharing"
              : connected
                ? "Connected to Discord"
                : retrying
                  ? "Trying to connect…"
                  : "Waiting for Discord"}
          </strong>
          <small>{message}</small>
        </div>
        <button
          className="studio-button"
          disabled={!p.discordEnabled || retrying}
          onClick={() => void retry()}
        >
          <RefreshCw
            size={12}
            className={retrying ? "discord-retrying" : undefined}
          />
          {connected ? "Refresh now" : "Try again"}
        </button>
      </div>
      <div className="discord-preview" aria-label="Discord presence preview">
        <div className="discord-preview-heading">
          PROFILE PREVIEW{" "}
          <span>{connected ? "Sent to Discord" : "Preview only"}</span>
        </div>
        <div className="discord-preview-content">
          <div
            className="discord-preview-art"
            title={p.discordLargeText || undefined}
          >
            <img
              src="/craft-box-icon.png"
              alt="CraftBox logo"
              width={64}
              height={64}
            />
            {badge && (
              <span
                className="discord-preview-badge"
                title={p.discordSmallText || undefined}
              >
                <ImageIcon size={12} />
                <img
                  key={badge}
                  src={badge}
                  alt="Status badge"
                  referrerPolicy="no-referrer"
                  onError={(e) => {
                    e.currentTarget.style.display = "none";
                  }}
                />
              </span>
            )}
          </div>
          <div className="discord-preview-lines">
            <strong>CraftBox</strong>
            {p.discordDetails && <span>{p.discordDetails}</span>}
            <span>
              {p.discordAutomatic
                ? status.activity || "In the workspace"
                : p.discordState}
            </span>
            {p.discordShowElapsed && <small>{timer} elapsed</small>}
          </div>
        </div>
        {[
          ...(p.discordGitHubButton
            ? [
                {
                  label: "CraftBox on GitHub",
                  url: "https://github.com/Daniel-Ric/CraftBox",
                },
              ]
            : []),
          { label: p.discordButton1Label, url: p.discordButton1Url },
        ]
          .filter((b) => b.label.trim() && b.url.trim())
          .map((b, i) => (
            <div className="discord-preview-button" key={i} title={b.url}>
              {b.label}
            </div>
          ))}
        <small className="discord-preview-note">
          Preview of your selected settings. Discord can take a few seconds to
          update.
        </small>
      </div>
    </>
  );
}
