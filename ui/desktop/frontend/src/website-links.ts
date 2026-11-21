type ReleaseReference = { version?: string; releaseURL?: string };
type Channel = "stable" | "preview";

export const websiteHome = "https://craftbox.spindexgfx.com/";

const workspaceGuides: Record<string, string> = {
  home: "docs/desktop-quickstart/",
  capture: "docs/world-capture/",
  worlds: "docs/world-capture/",
  map: "docs/maps-and-regions/",
  packs: "docs/asset-export/",
  skins: "features/player-skins/",
  packets: "docs/session-recordings/",
  structures: "docs/structures/",
  bedrock: "features/archive/",
  marketplace: "docs/marketplace/",
  render: "docs/render-and-cinematics/",
  cinematic: "docs/render-and-cinematics/#build-a-camera-move",
};

export function workspaceGuide(workspace: string) {
  const page = Object.hasOwn(workspaceGuides, workspace)
    ? workspaceGuides[workspace]
    : "docs/";
  return new URL(page, websiteHome).href;
}

function publishedVersion(value?: string) {
  if (!value || /-dev(?:[.-]|$)/i.test(value)) return undefined;
  return /^(?:v|r)?\d+\.\d+\.\d+(?:-[a-z0-9.-]+)?$/i.test(value)
    ? value
    : undefined;
}

function releaseVersion(release?: ReleaseReference) {
  if (release?.releaseURL) {
    try {
      const link = new URL(release.releaseURL);
      if (link.origin === "https://github.com") {
        const tag = /^\/Daniel-Ric\/CraftBox\/releases\/tag\/([^/]+)\/?$/i.exec(
          link.pathname,
        )?.[1];
        const version = publishedVersion(tag && decodeURIComponent(tag));
        if (version) return version;
      }
    } catch {}
  }
  return publishedVersion(release?.version);
}

export function websiteChangelog(release?: ReleaseReference) {
  const url = new URL("changelog/", websiteHome);
  const version = releaseVersion(release);
  if (version) url.searchParams.set("version", version);
  return url.href;
}

export function websiteDownloads(
  channel: Channel,
  release?: ReleaseReference,
  platform?: string,
) {
  const url = new URL("download/", websiteHome);
  url.searchParams.set("channel", channel);
  const version = releaseVersion(release);
  if (version) url.searchParams.set("version", version);
  if (platform === "darwin" || platform === "windows" || platform === "linux") {
    url.searchParams.set(
      "platform",
      platform === "darwin" ? "macos" : platform,
    );
    url.searchParams.set("variant", "desktop");
  }
  return url.href;
}
