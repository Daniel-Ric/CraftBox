import { useEffect } from "react";
import { api } from "./bridge";
import { useWorkspacePreferences } from "./workspace-preferences";
import { notify } from "./StudioFlyouts";

export function useDiscordPresence(ready: boolean) {
  const p = useWorkspacePreferences();
  useEffect(() => {
    if (!ready) return;
    void api
      .ConfigureDiscordPresence({
        enabled: p.discordEnabled,
        details: p.discordDetails,
        state: p.discordState,
        automatic: p.discordAutomatic,
        showElapsed: p.discordShowElapsed,
        largeText: p.discordLargeText,
        smallImage: p.discordSmallImage,
        smallText: p.discordSmallText,
        hideGitHub: !p.discordGitHubButton,
        button1Label: p.discordButton1Label,
        button1Url: p.discordButton1Url,
      })
      .catch((e) =>
        notify({
          id: "discord-presence-error",
          tone: "error",
          title: "Discord presence",
          description: String(e),
        }),
      );
  }, [
    ready,
    p.discordEnabled,
    p.discordDetails,
    p.discordState,
    p.discordAutomatic,
    p.discordShowElapsed,
    p.discordLargeText,
    p.discordSmallImage,
    p.discordSmallText,
    p.discordGitHubButton,
    p.discordButton1Label,
    p.discordButton1Url,
  ]);
}
